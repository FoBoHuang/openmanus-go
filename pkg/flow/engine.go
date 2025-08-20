package flow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DefaultFlowEngine 默认流程引擎
type DefaultFlowEngine struct {
	mu                 sync.RWMutex
	executions         map[string]*FlowExecution
	taskExecutor       TaskExecutor
	dependencyResolver DependencyResolver
	agentFactory       AgentFactory
	maxConcurrency     int
}

// NewDefaultFlowEngine 创建默认流程引擎
func NewDefaultFlowEngine(agentFactory AgentFactory, maxConcurrency int) *DefaultFlowEngine {
	taskExecutor := NewDefaultTaskExecutor(agentFactory)
	dependencyResolver := NewDefaultDependencyResolver()

	if maxConcurrency <= 0 {
		maxConcurrency = 10 // 默认最大并发数
	}

	return &DefaultFlowEngine{
		executions:         make(map[string]*FlowExecution),
		taskExecutor:       taskExecutor,
		dependencyResolver: dependencyResolver,
		agentFactory:       agentFactory,
		maxConcurrency:     maxConcurrency,
	}
}

// Execute 执行工作流
func (e *DefaultFlowEngine) Execute(ctx context.Context, workflow *Workflow, input map[string]interface{}) (*FlowExecution, error) {
	// 验证工作流
	if err := workflow.Validate(); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	// 验证任务
	for _, task := range workflow.Tasks {
		if err := e.taskExecutor.(*DefaultTaskExecutor).ValidateTask(task); err != nil {
			return nil, fmt.Errorf("task %s validation failed: %w", task.ID, err)
		}
	}

	// 创建执行实例
	executionID := uuid.New().String()
	execution := NewFlowExecution(executionID, workflow)
	execution.Input = input

	// 创建可取消的上下文
	execCtx, cancel := context.WithCancel(ctx)
	execution.SetCancelFunc(cancel)

	// 注册执行实例
	e.mu.Lock()
	e.executions[executionID] = execution
	e.mu.Unlock()

	// 开始执行
	execution.Start()

	// 根据执行模式选择执行策略
	go func() {
		defer func() {
			if r := recover(); r != nil {
				execution.Fail(fmt.Errorf("panic during execution: %v", r))
			}
		}()

		var err error
		switch workflow.Mode {
		case ExecutionModeSequential:
			err = e.executeSequential(execCtx, execution)
		case ExecutionModeParallel:
			err = e.executeParallel(execCtx, execution)
		case ExecutionModeDAG:
			err = e.executeDAG(execCtx, execution)
		default:
			err = fmt.Errorf("unsupported execution mode: %s", workflow.Mode)
		}

		if err != nil {
			execution.Fail(err)
		} else {
			// 收集所有任务的输出作为流程输出
			output := e.collectFlowOutput(execution)
			execution.Complete(output)
		}
	}()

	return execution, nil
}

// executeSequential 顺序执行
func (e *DefaultFlowEngine) executeSequential(ctx context.Context, execution *FlowExecution) error {
	for _, task := range execution.Workflow.Tasks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := e.taskExecutor.Execute(ctx, task, execution); err != nil {
			return fmt.Errorf("task %s failed: %w", task.ID, err)
		}
	}
	return nil
}

// executeParallel 并行执行
func (e *DefaultFlowEngine) executeParallel(ctx context.Context, execution *FlowExecution) error {
	// 解析依赖关系，获取可并行执行的任务组
	levels, err := e.dependencyResolver.Resolve(execution.Workflow.Tasks)
	if err != nil {
		return fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	// 按层级执行
	for levelIndex, level := range levels {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := e.executeTaskLevel(ctx, level, execution, levelIndex); err != nil {
			return err
		}
	}

	return nil
}

// executeDAG DAG 执行（与并行执行类似，但有更复杂的依赖处理）
func (e *DefaultFlowEngine) executeDAG(ctx context.Context, execution *FlowExecution) error {
	return e.executeParallel(ctx, execution) // 目前与并行执行相同
}

// executeTaskLevel 执行任务层级
func (e *DefaultFlowEngine) executeTaskLevel(ctx context.Context, tasks []*Task, execution *FlowExecution, levelIndex int) error {
	if len(tasks) == 0 {
		return nil
	}

	// 限制并发数
	concurrency := len(tasks)
	if concurrency > e.maxConcurrency {
		concurrency = e.maxConcurrency
	}

	// 创建信号量控制并发
	semaphore := make(chan struct{}, concurrency)
	errChan := make(chan error, len(tasks))
	var wg sync.WaitGroup

	// 并行执行当前层级的所有任务
	for _, task := range tasks {
		wg.Add(1)
		go func(t *Task) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := e.taskExecutor.Execute(ctx, t, execution); err != nil {
				errChan <- fmt.Errorf("task %s failed: %w", t.ID, err)
				return
			}
			errChan <- nil
		}(task)
	}

	// 等待所有任务完成
	wg.Wait()
	close(errChan)

	// 检查错误
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// collectFlowOutput 收集流程输出
func (e *DefaultFlowEngine) collectFlowOutput(execution *FlowExecution) map[string]interface{} {
	output := make(map[string]interface{})

	// 收集所有任务的输出
	taskOutputs := make(map[string]interface{})
	for _, task := range execution.Workflow.Tasks {
		if task.IsCompleted() {
			taskOutputs[task.ID] = task.Output
		}
	}
	output["tasks"] = taskOutputs

	// 添加执行统计信息
	stats := e.getExecutionStats(execution)
	output["stats"] = stats

	// 添加流程级别的结果
	output["flow_id"] = execution.ID
	output["workflow_name"] = execution.Workflow.Name
	output["execution_mode"] = execution.Workflow.Mode
	output["completed_at"] = time.Now()

	return output
}

// getExecutionStats 获取执行统计信息
func (e *DefaultFlowEngine) getExecutionStats(execution *FlowExecution) map[string]interface{} {
	stats := map[string]interface{}{
		"total_tasks":     len(execution.Workflow.Tasks),
		"completed_tasks": 0,
		"failed_tasks":    0,
		"skipped_tasks":   0,
	}

	totalSteps := 0
	for _, task := range execution.Workflow.Tasks {
		switch task.Status {
		case TaskStatusCompleted:
			stats["completed_tasks"] = stats["completed_tasks"].(int) + 1
		case TaskStatusFailed:
			stats["failed_tasks"] = stats["failed_tasks"].(int) + 1
		case TaskStatusSkipped:
			stats["skipped_tasks"] = stats["skipped_tasks"].(int) + 1
		}

		if task.Trace != nil {
			totalSteps += len(task.Trace.Steps)
		}
	}

	stats["total_steps"] = totalSteps
	stats["success_rate"] = float64(stats["completed_tasks"].(int)) / float64(stats["total_tasks"].(int))

	if execution.StartTime != nil && execution.EndTime != nil {
		stats["total_duration"] = execution.EndTime.Sub(*execution.StartTime).Seconds()
	}

	return stats
}

// GetExecution 获取执行状态
func (e *DefaultFlowEngine) GetExecution(id string) (*FlowExecution, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	execution, exists := e.executions[id]
	if !exists {
		return nil, fmt.Errorf("execution not found: %s", id)
	}

	return execution, nil
}

// CancelExecution 取消执行
func (e *DefaultFlowEngine) CancelExecution(id string) error {
	e.mu.RLock()
	execution, exists := e.executions[id]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("execution not found: %s", id)
	}

	execution.Cancel()
	return nil
}

// Subscribe 监听事件
func (e *DefaultFlowEngine) Subscribe(executionID string) (<-chan *FlowEvent, error) {
	execution, err := e.GetExecution(executionID)
	if err != nil {
		return nil, err
	}

	return execution.GetEventChannel(), nil
}

// Cleanup 清理资源
func (e *DefaultFlowEngine) Cleanup(executionID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	execution, exists := e.executions[executionID]
	if !exists {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	// 取消执行（如果还在运行）
	if execution.Status == FlowStatusRunning {
		execution.Cancel()
	}

	// 关闭事件通道
	close(execution.eventChannel)

	// 从执行列表中移除
	delete(e.executions, executionID)

	return nil
}

// ListExecutions 列出所有执行实例
func (e *DefaultFlowEngine) ListExecutions() []*FlowExecution {
	e.mu.RLock()
	defer e.mu.RUnlock()

	executions := make([]*FlowExecution, 0, len(e.executions))
	for _, execution := range e.executions {
		executions = append(executions, execution)
	}

	return executions
}

// GetExecutionSummary 获取执行摘要
func (e *DefaultFlowEngine) GetExecutionSummary(id string) (map[string]interface{}, error) {
	execution, err := e.GetExecution(id)
	if err != nil {
		return nil, err
	}

	summary := map[string]interface{}{
		"id":            execution.ID,
		"workflow_name": execution.Workflow.Name,
		"status":        execution.Status,
		"mode":          execution.Workflow.Mode,
		"created_at":    execution.Workflow.CreatedAt,
	}

	if execution.StartTime != nil {
		summary["start_time"] = execution.StartTime
	}

	if execution.EndTime != nil {
		summary["end_time"] = execution.EndTime
		summary["duration"] = execution.Duration.Seconds()
	}

	if execution.Error != "" {
		summary["error"] = execution.Error
	}

	// 添加任务摘要
	taskSummaries := make([]map[string]interface{}, 0, len(execution.Workflow.Tasks))
	for _, task := range execution.Workflow.Tasks {
		taskSummary := e.taskExecutor.(*DefaultTaskExecutor).GetExecutionSummary(task)
		taskSummaries = append(taskSummaries, taskSummary)
	}
	summary["tasks"] = taskSummaries

	// 添加统计信息
	summary["stats"] = e.getExecutionStats(execution)

	return summary, nil
}

// SetMaxConcurrency 设置最大并发数
func (e *DefaultFlowEngine) SetMaxConcurrency(maxConcurrency int) {
	if maxConcurrency > 0 {
		e.maxConcurrency = maxConcurrency
	}
}

// GetMaxConcurrency 获取最大并发数
func (e *DefaultFlowEngine) GetMaxConcurrency() int {
	return e.maxConcurrency
}
