package flow

import (
	"context"
	"fmt"
	"time"

	"openmanus-go/pkg/agent"
	"openmanus-go/pkg/logger"
)

// DefaultTaskExecutor 默认任务执行器
type DefaultTaskExecutor struct {
	agentFactory AgentFactory
}

// NewDefaultTaskExecutor 创建默认任务执行器
func NewDefaultTaskExecutor(agentFactory AgentFactory) *DefaultTaskExecutor {
	return &DefaultTaskExecutor{
		agentFactory: agentFactory,
	}
}

// Execute 执行任务
func (e *DefaultTaskExecutor) Execute(ctx context.Context, task *Task, execution *FlowExecution) error {
	// 检查任务状态
	if task.Status != TaskStatusPending {
		return fmt.Errorf("task %s is not in pending status: %s", task.ID, task.Status)
	}

	// 检查依赖是否满足
	if !task.CanExecute(execution) {
		return fmt.Errorf("task %s dependencies not satisfied", task.ID)
	}

	// 开始执行任务
	task.Start()
	logger.Infow("task.start", "execution_id", execution.ID, "task_id", task.ID, "task_name", task.Name, "agent_type", task.AgentType)
	execution.EmitEvent(FlowEventTypeTaskStarted, task.ID, map[string]interface{}{
		"agent_type": task.AgentType,
		"goal":       task.Goal,
	}, fmt.Sprintf("Task %s started", task.Name))

	// 获取或创建 Agent
	ag, err := e.getOrCreateAgent(task, execution)
	if err != nil {
		task.Fail(err)
		logger.Errorw("task.agent_create_failed", "task_id", task.ID, "error", err)
		execution.EmitEvent(FlowEventTypeTaskFailed, task.ID, map[string]interface{}{
			"error": err.Error(),
		}, fmt.Sprintf("Failed to create agent for task %s", task.Name))
		return err
	}

	// 准备任务输入
	taskInput := e.prepareTaskInput(task, execution)

	// 构建任务目标
	goal := e.buildTaskGoal(task, taskInput)
	logger.Debugw("task.goal", "task_id", task.ID, "goal_preview", preview(goal))

	// 执行任务
	result, err := ag.Loop(ctx, goal)
	if err != nil {
		task.Fail(err)
		logger.Errorw("task.failed", "task_id", task.ID, "error", err)
		execution.EmitEvent(FlowEventTypeTaskFailed, task.ID, map[string]interface{}{
			"error": err.Error(),
		}, fmt.Sprintf("Task %s failed: %v", task.Name, err))
		return err
	}

	// 处理任务输出
	output := e.processTaskOutput(task, result, ag)

	// 完成任务
	task.Complete(output)
	execution.SetTaskResult(task.ID, output)
	logger.Infow("task.completed", "task_id", task.ID, "duration_sec", task.Duration.Seconds(), "steps", output["steps_count"], "status", output["status"])
	execution.EmitEvent(FlowEventTypeTaskCompleted, task.ID, output, fmt.Sprintf("Task %s completed", task.Name))

	// 保存执行轨迹
	if trace := ag.GetTrace(); trace != nil {
		task.Trace = trace
	}

	return nil
}

// getOrCreateAgent 获取或创建 Agent
func (e *DefaultTaskExecutor) getOrCreateAgent(task *Task, execution *FlowExecution) (agent.Agent, error) {
	// 尝试从 Agent 池获取
	if ag, exists := execution.GetAgent(task.AgentType); exists {
		return ag, nil
	}

	// 创建新的 Agent
	agentConfig := make(map[string]interface{})

	// 从任务输入中提取 Agent 配置
	if config, ok := task.Input["agent_config"].(map[string]interface{}); ok {
		agentConfig = config
	}

	ag, err := e.agentFactory.CreateAgent(task.AgentType, agentConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent of type %s: %w", task.AgentType, err)
	}

	// 将 Agent 添加到池中
	execution.SetAgent(task.AgentType, ag)

	return ag, nil
}

// prepareTaskInput 准备任务输入
func (e *DefaultTaskExecutor) prepareTaskInput(task *Task, execution *FlowExecution) map[string]interface{} {
	taskInput := make(map[string]interface{})

	// 复制任务的输入数据
	for k, v := range task.Input {
		taskInput[k] = v
	}

	// 添加流程级别的输入
	for k, v := range execution.Input {
		if _, exists := taskInput[k]; !exists {
			taskInput[k] = v
		}
	}

	// 添加共享上下文
	for k, v := range execution.Context {
		if _, exists := taskInput[k]; !exists {
			taskInput[k] = v
		}
	}

	// 添加依赖任务的输出
	for _, depID := range task.Dependencies {
		if result, exists := execution.GetTaskResult(depID); exists {
			taskInput[fmt.Sprintf("dep_%s", depID)] = result
		}
	}

	return taskInput
}

// buildTaskGoal 构建任务目标
func (e *DefaultTaskExecutor) buildTaskGoal(task *Task, taskInput map[string]interface{}) string {
	goal := task.Goal

	// 如果输入中有特定的目标模板，使用它
	if goalTemplate, ok := taskInput["goal_template"].(string); ok {
		goal = goalTemplate
	}

	// 添加上下文信息
	if len(taskInput) > 0 {
		goal += "\n\n可用的输入数据："
		for k, v := range taskInput {
			if k != "agent_config" && k != "goal_template" {
				goal += fmt.Sprintf("\n- %s: %v", k, v)
			}
		}
	}

	return goal
}

// processTaskOutput 处理任务输出
func (e *DefaultTaskExecutor) processTaskOutput(task *Task, result string, ag agent.Agent) map[string]interface{} {
	output := map[string]interface{}{
		"result":    result,
		"task_id":   task.ID,
		"task_name": task.Name,
		"timestamp": time.Now(),
	}

	// 添加执行统计信息
	if trace := ag.GetTrace(); trace != nil {
		output["steps_count"] = len(trace.Steps)
		output["status"] = trace.Status
		output["created_at"] = trace.CreatedAt
		output["updated_at"] = trace.UpdatedAt
		output["execution_time"] = trace.UpdatedAt.Sub(trace.CreatedAt).Seconds()
	}

	// 尝试解析结构化输出
	if structuredOutput := e.parseStructuredOutput(result); structuredOutput != nil {
		output["structured"] = structuredOutput
	}

	logger.Debugw("task.output", "task_id", task.ID, "result_preview", preview(result))

	return output
}

// parseStructuredOutput 解析结构化输出
func (e *DefaultTaskExecutor) parseStructuredOutput(result string) map[string]interface{} {
	// 这里可以实现更复杂的输出解析逻辑
	// 例如：JSON 解析、表格提取、文件路径提取等

	// 简单实现：检查是否包含特定关键词
	structured := make(map[string]interface{})

	if len(result) > 1000 {
		structured["is_long_output"] = true
		structured["length"] = len(result)
	}

	// 检查是否包含文件路径
	if containsFilePath(result) {
		structured["contains_file_path"] = true
	}

	// 检查是否包含 URL
	if containsURL(result) {
		structured["contains_url"] = true
	}

	if len(structured) == 0 {
		return nil
	}

	return structured
}

// containsFilePath 检查是否包含文件路径
func containsFilePath(text string) bool {
	// 简单的文件路径检测
	return len(text) > 0 && (text[0] == '/' || // Unix 绝对路径
		(len(text) > 2 && text[1] == ':') || // Windows 绝对路径
		text[:2] == "./" || text[:3] == "../") // 相对路径
}

// containsURL 检查是否包含 URL
func containsURL(text string) bool {
	// 简单的 URL 检测
	return len(text) > 7 && (text[:7] == "http://" ||
		text[:8] == "https://")
}

// ExecuteWithTimeout 带超时的任务执行
func (e *DefaultTaskExecutor) ExecuteWithTimeout(ctx context.Context, task *Task, execution *FlowExecution, timeout time.Duration) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- e.Execute(timeoutCtx, task, execution)
	}()

	select {
	case err := <-done:
		return err
	case <-timeoutCtx.Done():
		task.Fail(fmt.Errorf("task execution timeout after %v", timeout))
		logger.Warnw("task.timeout", "task_id", task.ID, "timeout", timeout.String())
		execution.EmitEvent(FlowEventTypeTaskFailed, task.ID, map[string]interface{}{
			"error":   "timeout",
			"timeout": timeout.String(),
		}, fmt.Sprintf("Task %s timed out", task.Name))
		return fmt.Errorf("task %s execution timeout", task.ID)
	}
}

// ValidateTask 验证任务是否可以执行
func (e *DefaultTaskExecutor) ValidateTask(task *Task) error {
	if task.ID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	if task.Name == "" {
		return fmt.Errorf("task name cannot be empty")
	}

	if task.AgentType == "" {
		return fmt.Errorf("task agent type cannot be empty")
	}

	if task.Goal == "" {
		return fmt.Errorf("task goal cannot be empty")
	}

	// 验证 Agent 类型是否支持
	supportedTypes := e.agentFactory.GetSupportedTypes()
	found := false
	for _, t := range supportedTypes {
		if t == task.AgentType {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("unsupported agent type: %s", task.AgentType)
	}

	// 验证 Agent 配置
	if agentConfig, ok := task.Input["agent_config"].(map[string]interface{}); ok {
		if err := e.agentFactory.ValidateAgentConfig(task.AgentType, agentConfig); err != nil {
			return fmt.Errorf("invalid agent config: %w", err)
		}
	}

	return nil
}

// GetExecutionSummary 获取执行摘要
func (e *DefaultTaskExecutor) GetExecutionSummary(task *Task) map[string]interface{} {
	summary := map[string]interface{}{
		"task_id":    task.ID,
		"task_name":  task.Name,
		"status":     task.Status,
		"agent_type": task.AgentType,
	}

	if task.StartTime != nil {
		summary["start_time"] = task.StartTime
	}

	if task.EndTime != nil {
		summary["end_time"] = task.EndTime
		summary["duration"] = task.Duration.Seconds()
	}

	if task.Error != "" {
		summary["error"] = task.Error
	}

	if task.Trace != nil {
		summary["steps_count"] = len(task.Trace.Steps)
		summary["trace_status"] = task.Trace.Status
	}

	return summary
}

func preview(s string) string {
	if len(s) <= 160 {
		return s
	}
	return s[:160] + "..."
}
