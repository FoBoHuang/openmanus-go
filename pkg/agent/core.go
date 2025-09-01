package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"openmanus-go/pkg/config"
	"openmanus-go/pkg/llm"
	"openmanus-go/pkg/logger"
	"openmanus-go/pkg/state"
	"openmanus-go/pkg/tool"
)

// Agent 定义 Agent 接口
type Agent interface {
	// Plan 根据目标和轨迹进行规划，返回下一步动作
	Plan(ctx context.Context, goal string, trace *state.Trace) (state.Action, error)

	// Act 执行动作并返回观测结果
	Act(ctx context.Context, action state.Action) (*state.Observation, error)

	// Reflect 基于轨迹进行反思
	Reflect(ctx context.Context, trace *state.Trace) (*state.ReflectionResult, error)

	// ShouldStop 判断是否应该停止
	ShouldStop(trace *state.Trace) bool

	// Loop 执行完整的控制循环
	Loop(ctx context.Context, goal string) (string, error)

	// GetTrace 获取最近的执行轨迹
	GetTrace() *state.Trace
}

// BaseAgent Agent 的基础实现
type BaseAgent struct {
	llmClient    llm.Client
	toolExecutor *tool.Executor
	planner      *Planner
	memory       *Memory
	reflector    *Reflector
	config       *Config
	mcpExecutor  *MCPExecutor // MCP 执行器
	// 移除了 taskAnalyzer 和 taskManager，采用统一线性执行策略
}

// Config Agent 配置
type Config struct {
	MaxSteps        int           `json:"max_steps" mapstructure:"max_steps"`
	MaxTokens       int           `json:"max_tokens" mapstructure:"max_tokens"`
	MaxDuration     time.Duration `json:"max_duration" mapstructure:"max_duration"`
	Temperature     float64       `json:"temperature" mapstructure:"temperature"`
	ReflectionSteps int           `json:"reflection_steps" mapstructure:"reflection_steps"` // 每隔几步进行反思
	MaxRetries      int           `json:"max_retries" mapstructure:"max_retries"`
	RetryBackoff    time.Duration `json:"retry_backoff" mapstructure:"retry_backoff"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		MaxSteps:        30, // 增加到30步，类似OpenManus的策略
		MaxTokens:       8000,
		MaxDuration:     10 * time.Minute, // 增加执行时间限制
		Temperature:     0.1,
		ReflectionSteps: 3,
		MaxRetries:      2,
		RetryBackoff:    time.Second,
	}
}

// NewBaseAgent 创建基础 Agent
func NewBaseAgent(llmClient llm.Client, toolRegistry *tool.Registry, config *Config) *BaseAgent {
	if config == nil {
		config = DefaultConfig()
	}

	if toolRegistry == nil {
		toolRegistry = tool.DefaultRegistry
	}

	toolExecutor := tool.NewExecutor(toolRegistry, 30*time.Second)
	planner := NewPlanner(llmClient, toolRegistry)
	memory := NewMemory()
	reflector := NewReflector(llmClient)
	// 移除了 taskAnalyzer 和 taskManager，采用统一线性执行策略

	return &BaseAgent{
		llmClient:    llmClient,
		toolExecutor: toolExecutor,
		planner:      planner,
		memory:       memory,
		reflector:    reflector,
		config:       config,
		// 移除了 taskAnalyzer 和 taskManager，采用统一线性执行策略
	}
}

// NewBaseAgentWithMCP 创建带 MCP 功能的基础 Agent（采用统一工具集合策略）
func NewBaseAgentWithMCP(llmClient llm.Client, toolRegistry *tool.Registry, agentConfig *Config, appConfig *config.Config) *BaseAgent {
	if agentConfig == nil {
		agentConfig = DefaultConfig()
	}

	if toolRegistry == nil {
		toolRegistry = tool.DefaultRegistry
	}

	// 创建基础组件
	memory := NewMemory()
	reflector := NewReflector(llmClient)

	// 如果有 MCP 配置，将 MCP 工具集成到统一注册表中
	var mcpExecutor *MCPExecutor
	if appConfig != nil && len(appConfig.MCP.Servers) > 0 {
		// 创建 MCP 发现服务
		mcpDiscovery := NewMCPDiscoveryService(appConfig)

		// 创建 MCP 执行器
		mcpExecutor = NewMCPExecutor(appConfig, mcpDiscovery)

		// 同步启动 MCP 发现服务并注册工具到统一注册表
		// 使用 channel 来等待 MCP 工具注册完成
		mcpReady := make(chan struct{})

		go func() {
			defer close(mcpReady)

			ctx := context.Background()
			if err := mcpDiscovery.Start(ctx); err != nil {
				logger.Warnw("Failed to start MCP discovery service", "error", err)
				return
			}

			// 等待一段时间让MCP工具发现完成
			time.Sleep(2 * time.Second)

			// 将发现的MCP工具注册到统一注册表
			allTools := mcpDiscovery.GetAllTools()
			mcpToolInfos := make([]tool.ToolInfo, 0, len(allTools))
			for _, mcpTool := range allTools {
				mcpToolInfos = append(mcpToolInfos, tool.ToolInfo{
					Name:         mcpTool.Name,
					Description:  mcpTool.Description,
					InputSchema:  mcpTool.InputSchema,
					OutputSchema: make(map[string]any), // MCP工具通常没有预定义的输出schema
					Type:         tool.ToolTypeMCP,
					ServerName:   mcpTool.ServerName,
				})
			}

			if len(mcpToolInfos) > 0 {
				if err := toolRegistry.RegisterMCPTools(mcpToolInfos, mcpExecutor); err != nil {
					logger.Warnw("Failed to register MCP tools", "error", err)
				} else {
					logger.Infow("Successfully registered MCP tools to unified registry", "count", len(mcpToolInfos))
				}
			}
		}()

		// 等待 MCP 工具注册完成，但设置超时避免无限等待
		select {
		case <-mcpReady:
			logger.Infow("MCP tools registration completed")
		case <-time.After(5 * time.Second):
			logger.Warnw("MCP tools registration timeout, proceeding without MCP tools")
		}
	}

	// 创建统一的工具执行器和规划器
	toolExecutor := tool.NewExecutor(toolRegistry, 30*time.Second)
	planner := NewPlanner(llmClient, toolRegistry) // 使用统一的规划器，不需要特殊的MCP逻辑

	return &BaseAgent{
		llmClient:    llmClient,
		toolExecutor: toolExecutor,
		planner:      planner,
		memory:       memory,
		reflector:    reflector,
		config:       agentConfig,
		mcpExecutor:  mcpExecutor, // 保留引用用于清理
		// 移除了 taskAnalyzer 和 taskManager，采用统一线性执行策略
	}
}

// Plan 进行规划
func (a *BaseAgent) Plan(ctx context.Context, goal string, trace *state.Trace) (state.Action, error) {
	return a.planner.Plan(ctx, goal, trace)
}

// Act 执行动作（统一执行接口）
func (a *BaseAgent) Act(ctx context.Context, action state.Action) (*state.Observation, error) {
	// 统一通过工具执行器执行，不区分工具类型
	// 工具执行器会根据工具的实现自动路由到正确的执行方法
	return a.toolExecutor.Execute(ctx, action)
}

// Reflect 进行反思
func (a *BaseAgent) Reflect(ctx context.Context, trace *state.Trace) (*state.ReflectionResult, error) {
	return a.reflector.Reflect(ctx, trace)
}

// ShouldStop 判断是否应该停止
func (a *BaseAgent) ShouldStop(trace *state.Trace) bool {
	// 检查预算限制
	if trace.IsExceededBudget() {
		return true
	}

	// 检查状态
	if trace.Status != state.TraceStatusRunning {
		return true
	}

	// 检查最近的步骤是否明确表明应该停止
	if len(trace.Steps) > 0 {
		lastStep := trace.Steps[len(trace.Steps)-1]

		// 只有明确的 stop 动作才立即停止
		if lastStep.Action.Name == "stop" {
			return true
		}

		// 对于 direct_answer，我们需要使用任务完成度分析来决定
		// 这里不再直接停止，而是让循环继续，由 Loop 方法来处理
	}

	return false
}

/* LoopWithTaskManagement 已移除 - 现在使用统一线性执行策略
   原来的任务分解和管理功能已被简化为统一的线性执行模式，与 OpenManus 保持一致
func (a *BaseAgent) LoopWithTaskManagement(ctx context.Context, goal string) (string, error) {
	logger.Infow("agent.task_loop.start", "goal", goal)

	// 第一步：分解目标为子任务
	plan, err := a.taskManager.DecomposeGoal(ctx, goal)
	if err != nil {
		logger.Errorw("Failed to decompose goal", "error", err)
		return "", fmt.Errorf("goal decomposition failed: %w", err)
	}

	logger.Infow("agent.task_loop.plan_created",
		"plan_id", plan.ID,
		"subtasks", len(plan.SubTasks),
		"execution_order", plan.ExecutionOrder)

	// 创建执行轨迹
	trace := &state.Trace{
		Goal:  goal,
		Steps: []state.Step{},
		Budget: state.Budget{
			MaxSteps:    a.config.MaxSteps * len(plan.SubTasks), // 扩展预算以支持多任务
			MaxTokens:   a.config.MaxTokens,
			MaxDuration: a.config.MaxDuration,
			StartTime:   time.Now(),
		},
		Status:    state.TraceStatusRunning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 执行任务循环
	for !a.taskManager.IsAllTasksCompleted() && !a.ShouldStop(trace) {
		select {
		case <-ctx.Done():
			trace.Status = state.TraceStatusCanceled
			logger.Warnw("agent.task_loop.canceled", "goal", goal)
			return "", ctx.Err()
		default:
		}

		// 获取下一个要执行的任务
		nextTask := a.taskManager.GetNextTask()
		if nextTask == nil {
			logger.Warnw("No next task available but not all tasks completed")
			break
		}

		logger.Infow("agent.task_loop.executing_task",
			"task_id", nextTask.ID,
			"task_title", nextTask.Title,
			"task_type", nextTask.Type)

		// 标记任务为进行中
		a.taskManager.UpdateTaskStatus(nextTask.ID, TaskStatusInProgress, nil, nil)

		// 执行单个任务
		taskResult, err := a.executeSubTask(ctx, nextTask, trace)
		if err != nil {
			logger.Errorw("Task execution failed",
				"task_id", nextTask.ID,
				"error", err)
			a.taskManager.UpdateTaskStatus(nextTask.ID, TaskStatusFailed, nil, nil)
			continue // 继续执行其他任务
		}

		// 标记任务完成
		evidence := []string{fmt.Sprintf("subtask_completed:%s", nextTask.ID)}
		a.taskManager.UpdateTaskStatus(nextTask.ID, TaskStatusCompleted, taskResult, evidence)

		logger.Infow("agent.task_loop.task_completed",
			"task_id", nextTask.ID,
			"task_title", nextTask.Title)

		// 检查是否所有任务都已完成
		if a.taskManager.IsAllTasksCompleted() {
			logger.Infow("agent.task_loop.all_tasks_completed")
			break
		}
	}

	// 生成最终结果
	summary := a.taskManager.GetCompletionSummary()
	if summary["all_completed"].(bool) {
		trace.Status = state.TraceStatusCompleted
		finalResult := a.generateTaskCompletionSummary(plan, summary)
		logger.Infow("agent.task_loop.success",
			"completion_rate", summary["completion_rate"],
			"total_tasks", summary["total_tasks"])
		return finalResult, nil
	} else {
		trace.Status = state.TraceStatusFailed
		logger.Warnw("agent.task_loop.incomplete",
			"completion_rate", summary["completion_rate"],
			"pending_tasks", summary["pending"],
			"failed_tasks", summary["failed"])
		return a.generateTaskCompletionSummary(plan, summary), nil
	}
}
*/

// Loop 执行完整的控制循环（统一线性执行策略）
func (a *BaseAgent) Loop(ctx context.Context, goal string) (string, error) {
	// 使用统一的线性执行策略，与 OpenManus 保持一致
	logger.Infow("agent.loop.unified_execution", "goal", goal)
	return a.unifiedLoop(ctx, goal)
}

// 移除了 isComplexGoal 函数，现在使用统一的线性执行策略

// unifiedLoop 统一线性执行循环（类似 OpenManus 的策略）
func (a *BaseAgent) unifiedLoop(ctx context.Context, goal string) (string, error) {
	// 创建初始轨迹
	trace := &state.Trace{
		Goal: goal,
		Budget: state.Budget{
			MaxSteps:    a.config.MaxSteps,
			MaxTokens:   a.config.MaxTokens,
			MaxDuration: a.config.MaxDuration,
			StartTime:   time.Now(),
		},
		Status:    state.TraceStatusRunning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	var finalResult string

	logger.Infof("🚀 [AGENT] Starting unified execution: %s", goal)
	logger.Infof("📊 [BUDGET] Max steps: %d | Max tokens: %d | Max duration: %s", a.config.MaxSteps, a.config.MaxTokens, a.config.MaxDuration.String())
	logger.Infof("═══════════════════════════════════════════════════════════════")

	for !a.ShouldStop(trace) {
		// 检查上下文取消
		select {
		case <-ctx.Done():
			trace.Status = state.TraceStatusCanceled
			return "", ctx.Err()
		default:
		}

		stepNum := len(trace.Steps) + 1
		logger.Infof("")
		logger.Infof("🤔 [STEP %d/%d] Planning next action...", stepNum, a.config.MaxSteps)
		logger.Infof("⏱️  [PROGRESS] %.1f%% complete | Elapsed: %v",
			float64(stepNum-1)/float64(a.config.MaxSteps)*100,
			time.Since(trace.Budget.StartTime).Round(time.Second))

		// 规划下一步动作
		action, err := a.Plan(ctx, goal, trace)
		if err != nil {
			trace.Status = state.TraceStatusFailed
			logger.Errorf("❌ [PLAN] Planning failed: %v", err)
			return "", fmt.Errorf("planning failed: %w", err)
		}

		// 添加步骤到轨迹
		_ = trace.AddStep(action)

		// 处理直接回答 - 简化处理，直接接受
		if action.Name == "direct_answer" {
			potentialResult := getStringFromArgs(action.Args, "answer")
			finalResult = potentialResult
			trace.Status = state.TraceStatusCompleted
			logger.Infof("✅ [ANSWER] Task completed with direct answer")
			break
		}

		// 处理停止指令
		if action.Name == "stop" {
			finalResult = getStringFromArgs(action.Args, "reason")
			trace.Status = state.TraceStatusCompleted
			logger.Infof("🛑 [STOP] %s", finalResult)
			break
		}

		// 执行工具调用
		logger.Infof("⚡ [EXEC] Executing %s...", action.Name)
		observation, err := a.Act(ctx, action)

		if err != nil {
			// 执行失败，但继续运行让 Agent 处理错误
			observation = &state.Observation{
				Tool:   action.Name,
				ErrMsg: err.Error(),
			}
			logger.Warnf("⚠️  [ERROR] Tool execution failed: %v", err)
		}

		// 更新观测结果
		trace.UpdateObservation(observation)

		// 记录执行结果
		if observation.ErrMsg != "" {
			logger.Warnf("❌ [RESULT] %s failed: %s", action.Name, observation.ErrMsg)
		} else {
			outputPreview := ""
			if observation.Output != nil {
				if outputBytes, err := json.Marshal(observation.Output); err == nil {
					outputPreview = truncateString(string(outputBytes), 150)
				}
			}
			logger.Infof("✅ [RESULT] %s completed: %s", action.Name, outputPreview)
		}

		// 检查预算
		if trace.IsExceededBudget() {
			trace.Status = state.TraceStatusFailed // 使用现有的状态
			finalResult = fmt.Sprintf("Execution stopped due to budget limits. Completed %d steps.", len(trace.Steps))
			logger.Warnf("💰 [BUDGET] Execution stopped due to budget limits")
			break
		}
	}

	// 如果没有明确的结果，生成默认摘要
	if finalResult == "" {
		finalResult = a.generateExecutionSummary(trace)
	}

	logger.Infof("")
	logger.Infof("═══════════════════════════════════════════════════════════════")
	logger.Infof("🏁 [DONE] Execution completed!")
	logger.Infof("📋 [SUMMARY] Goal: %s", goal)
	logger.Infof("📊 [STATS] Steps: %d/%d | Status: %s | Duration: %v",
		len(trace.Steps), a.config.MaxSteps, trace.Status, time.Since(trace.Budget.StartTime).Round(time.Second))
	if len(trace.Steps) > 0 {
		logger.Infof("🔍 [STEPS] Execution trace:")
		for i, step := range trace.Steps {
			status := "✅"
			if step.Observation != nil && step.Observation.ErrMsg != "" {
				status = "❌"
			}
			logger.Infof("   %d. %s %s", i+1, status, step.Action.Name)
		}
	}
	logger.Infof("═══════════════════════════════════════════════════════════════")

	return finalResult, nil
}

// generateExecutionSummary 生成执行摘要
func (a *BaseAgent) generateExecutionSummary(trace *state.Trace) string {
	var summary strings.Builder

	summary.WriteString("Execution Summary:\n")
	summary.WriteString(fmt.Sprintf("Goal: %s\n", trace.Goal))
	summary.WriteString(fmt.Sprintf("Status: %s\n", trace.Status))
	summary.WriteString(fmt.Sprintf("Steps: %d\n", len(trace.Steps)))
	summary.WriteString(fmt.Sprintf("Duration: %v\n", time.Since(trace.Budget.StartTime).Round(time.Second)))

	if len(trace.Steps) > 0 {
		summary.WriteString("\nSteps executed:\n")
		for i, step := range trace.Steps {
			status := "✅"
			if step.Observation != nil && step.Observation.ErrMsg != "" {
				status = "❌"
			}
			summary.WriteString(fmt.Sprintf("%d. %s %s\n", i+1, status, step.Action.Name))
		}
	}

	return summary.String()
}

// truncateString 在 planner.go 中已定义，这里移除重复定义

// getStringFromArgs 从参数中获取字符串值
func getStringFromArgs(args map[string]any, key string) string {
	if value, ok := args[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// SetConfig 设置配置
func (a *BaseAgent) SetConfig(config *Config) {
	a.config = config
}

// GetConfig 获取配置
func (a *BaseAgent) GetConfig() *Config {
	return a.config
}

// GetTrace 获取最近的执行轨迹（如果有的话）
func (a *BaseAgent) GetTrace() *state.Trace {
	return a.memory.GetCurrentTrace()
}

// SaveTrace 保存轨迹
func (a *BaseAgent) SaveTrace(trace *state.Trace, store state.Store) error {
	return store.Save(trace)
}

// LoadTrace 加载轨迹
func (a *BaseAgent) LoadTrace(id string, store state.Store) (*state.Trace, error) {
	return store.Load(id)
}
