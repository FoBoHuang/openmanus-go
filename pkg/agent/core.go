package agent

import (
	"context"
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
	mcpExecutor  *MCPExecutor            // MCP 执行器
	taskAnalyzer *TaskCompletionAnalyzer // 任务完成度分析器
	taskManager  *TaskManager            // 多步任务管理器
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
		MaxSteps:        10,
		MaxTokens:       8000,
		MaxDuration:     5 * time.Minute,
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
	taskAnalyzer := NewTaskCompletionAnalyzer(llmClient)
	taskManager := NewTaskManager(llmClient)

	return &BaseAgent{
		llmClient:    llmClient,
		toolExecutor: toolExecutor,
		planner:      planner,
		memory:       memory,
		reflector:    reflector,
		config:       config,
		taskAnalyzer: taskAnalyzer,
		taskManager:  taskManager,
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

		// 启动 MCP 发现服务并注册工具到统一注册表
		go func() {
			ctx := context.Background()
			if err := mcpDiscovery.Start(ctx); err != nil {
				logger.Get().Sugar().Warnw("Failed to start MCP discovery service", "error", err)
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
					logger.Get().Sugar().Warnw("Failed to register MCP tools", "error", err)
				} else {
					logger.Get().Sugar().Infow("Successfully registered MCP tools to unified registry", "count", len(mcpToolInfos))
				}
			}
		}()
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
		taskAnalyzer: NewTaskCompletionAnalyzer(llmClient),
		taskManager:  NewTaskManager(llmClient),
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

// LoopWithTaskManagement 使用多步任务管理的执行循环
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

// Loop 执行完整的控制循环（保持向后兼容）
func (a *BaseAgent) Loop(ctx context.Context, goal string) (string, error) {
	// 智能选择执行模式
	if a.isComplexGoal(goal) {
		logger.Infow("agent.loop.using_task_management", "goal", goal)
		return a.LoopWithTaskManagement(ctx, goal)
	}

	// 对于简单目标，使用原有逻辑
	logger.Infow("agent.loop.using_standard_mode", "goal", goal)
	return a.standardLoop(ctx, goal)
}

// isComplexGoal 判断是否为复杂目标
func (a *BaseAgent) isComplexGoal(goal string) bool {
	goalLower := strings.ToLower(goal)

	// 检测多步任务的关键词
	multiStepKeywords := []string{"并", "然后", "and", "also", "additionally", "保存", "写入", "文件", "总结", "分析"}
	keywordCount := 0

	for _, keyword := range multiStepKeywords {
		if strings.Contains(goalLower, keyword) {
			keywordCount++
		}
	}

	// 如果包含多个关键词，或者明确包含文件操作，认为是复杂目标
	return keywordCount >= 2 ||
		strings.Contains(goalLower, "保存") ||
		strings.Contains(goalLower, "写入") ||
		strings.Contains(goalLower, "文件")
}

// standardLoop 标准执行循环（原有逻辑）
func (a *BaseAgent) standardLoop(ctx context.Context, goal string) (string, error) {
	// 创建初始轨迹
	trace := &state.Trace{
		Goal:  goal,
		Steps: []state.Step{},
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

	logger.Infof("🚀 [AGENT] Starting execution: %s", goal)
	logger.Infof("📊 [BUDGET] Max steps: %d | Max tokens: %d | Max duration: %s", a.config.MaxSteps, a.config.MaxTokens, a.config.MaxDuration.String())

	var finalResult string

	for !a.ShouldStop(trace) {
		select {
		case <-ctx.Done():
			trace.Status = state.TraceStatusCanceled
			logger.Warnw("agent.loop.canceled", "goal", goal)
			return "", ctx.Err()
		default:
		}

		// 规划下一步
		logger.Infof("\n🤔 [STEP %d] Planning next action...", len(trace.Steps)+1)
		action, err := a.Plan(ctx, goal, trace)
		if err != nil {
			trace.Status = state.TraceStatusFailed
			logger.Errorf("❌ [PLAN] Planning failed: %v", err)
			return "", fmt.Errorf("planning failed: %w", err)
		}
		logger.Infof("✅ [PLAN] Selected action: %s", action.Name)
		if len(fmt.Sprintf("%v", action.Args)) < 200 {
			logger.Debugf("🔧 [ARGS] %v", action.Args)
		}

		// 添加步骤到轨迹
		_ = trace.AddStep(action)

		// 处理直接回答 - 使用任务完成度分析来验证
		if action.Name == "direct_answer" {
			potentialResult := getStringFromArgs(action.Args, "answer")

			// 使用任务完成度分析器来验证任务是否真正完成
			if a.taskAnalyzer != nil {
				completionResult, err := a.taskAnalyzer.AnalyzeTaskCompletion(ctx, goal, trace)
				if err != nil {
					logger.Warnw("Task completion analysis failed, accepting direct answer", "error", err)
					finalResult = potentialResult
					trace.Status = state.TraceStatusCompleted
					break
				}

				if completionResult.IsComplete {
					// 任务确实完成了
					finalResult = potentialResult
					trace.Status = state.TraceStatusCompleted
					logger.Infof("✅ [ANSWER] Task verified as complete (confidence: %.1f)", completionResult.Confidence)
					logger.Infof("📋 [SUMMARY] Completed %d tasks", len(completionResult.CompletedTasks))
					break
				} else {
					// 任务还未完成，继续执行
					logger.Infof("⏳ [CONTINUE] Task incomplete - %d pending tasks", len(completionResult.PendingTasks))
					logger.Debugf("💡 [REASON] %s", completionResult.Reason)

					// 不执行 direct_answer，而是继续循环让 Agent 完成剩余任务
					// 移除最后一个 direct_answer 步骤，因为任务未完成
					if len(trace.Steps) > 0 {
						trace.Steps = trace.Steps[:len(trace.Steps)-1]
					}
					continue
				}
			} else {
				// 如果没有任务分析器，使用原来的逻辑
				finalResult = potentialResult
				trace.Status = state.TraceStatusCompleted
				logger.Infof("✅ [ANSWER] Task completed")
				break
			}
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
		}
		if observation != nil {
			if observation.ErrMsg != "" {
				logger.Warnf("⚠️  [RESULT] %s failed: %s (%.0fms)", action.Name, observation.ErrMsg, float64(observation.Latency))
			} else {
				logger.Infof("✅ [RESULT] %s completed successfully (%.0fms)", action.Name, float64(observation.Latency))
				if preview := previewAny(observation.Output); preview != nil {
					logger.Debugf("📄 [OUTPUT] %v", preview)
				}
			}
		}

		// 更新观测结果
		trace.UpdateObservation(observation)

		// 定期进行反思 (避免除零错误)
		if a.config.ReflectionSteps > 0 && len(trace.Steps)%a.config.ReflectionSteps == 0 {
			logger.Infof("🤖 [REFLECT] Analyzing progress after %d steps...", len(trace.Steps))
			reflection, err := a.Reflect(ctx, trace)
			if err == nil && reflection.ShouldStop {
				finalResult = reflection.Reason
				trace.Status = state.TraceStatusCompleted
				logger.Infof("🎯 [REFLECT] Task completed: %s (confidence: %.1f)", reflection.Reason, reflection.Confidence)
				break
			}
			if err != nil {
				logger.Warnf("⚠️  [REFLECT] Reflection failed: %v", err)
			} else {
				if reflection.RevisePlan {
					logger.Debugf("💭 [REFLECT] Continue with plan revision")
				} else {
					logger.Debugf("💭 [REFLECT] Continue without plan revision")
				}
			}
		}
	}

	// 如果没有明确的结果，生成总结
	if finalResult == "" {
		finalResult = a.generateSummary(trace)
	}

	logger.Infof("🏁 [DONE] Execution completed: %s | Steps: %d | Status: %s", goal, len(trace.Steps), trace.Status)

	return finalResult, nil
}

// generateSummary 生成执行总结
func (a *BaseAgent) generateSummary(trace *state.Trace) string {
	if len(trace.Steps) == 0 {
		return "No actions were taken."
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Executed %d steps for goal: %s\n", len(trace.Steps), trace.Goal))

	for i, step := range trace.Steps {
		summary.WriteString(fmt.Sprintf("Step %d: %s", i+1, step.Action.Name))
		if step.Observation != nil {
			if step.Observation.ErrMsg != "" {
				summary.WriteString(" (failed)")
			} else {
				summary.WriteString(" (success)")
			}
		}
		summary.WriteString("\n")
	}

	switch trace.Status {
	case state.TraceStatusCompleted:
		summary.WriteString("Task completed successfully.")
	case state.TraceStatusFailed:
		summary.WriteString("Task failed.")
	case state.TraceStatusCanceled:
		summary.WriteString("Task was canceled.")
	default:
		summary.WriteString("Task execution stopped.")
	}

	return summary.String()
}

// executeSubTask 执行单个子任务
func (a *BaseAgent) executeSubTask(ctx context.Context, task *SubTask, trace *state.Trace) (map[string]any, error) {
	logger.Infow("agent.subtask.start", "task_id", task.ID, "task_type", task.Type)

	// 构建针对子任务的目标
	subGoal := fmt.Sprintf("%s: %s", task.Title, task.Description)

	// 限制子任务的步数，避免无限循环
	maxSubSteps := 3
	stepCount := 0

	for stepCount < maxSubSteps {
		// 规划单步动作
		action, err := a.Plan(ctx, subGoal, trace)
		if err != nil {
			return nil, fmt.Errorf("planning failed for subtask %s: %w", task.ID, err)
		}

		logger.Infow("agent.subtask.action", "task_id", task.ID, "action", action.Name)

		// 添加步骤到轨迹
		_ = trace.AddStep(action)
		stepCount++

		// 执行动作
		observation, err := a.Act(ctx, action)
		if err != nil {
			observation = &state.Observation{
				Tool:   action.Name,
				ErrMsg: err.Error(),
			}
		}

		// 更新观测结果
		trace.UpdateObservation(observation)

		// 检查是否成功完成
		if observation != nil && observation.ErrMsg == "" {
			// 根据任务类型判断是否完成
			if a.isSubTaskCompleted(task, action, observation) {
				result := map[string]any{
					"action":      action.Name,
					"args":        action.Args,
					"observation": observation.Output,
				}
				logger.Infow("agent.subtask.completed", "task_id", task.ID, "action", action.Name)
				return result, nil
			}
		}

		// 如果是direct_answer，也认为任务完成
		if action.Name == "direct_answer" {
			result := map[string]any{
				"action": action.Name,
				"answer": getStringFromArgs(action.Args, "answer"),
			}
			return result, nil
		}
	}

	return nil, fmt.Errorf("subtask %s exceeded maximum steps (%d)", task.ID, maxSubSteps)
}

// isSubTaskCompleted 判断子任务是否完成
func (a *BaseAgent) isSubTaskCompleted(task *SubTask, action state.Action, observation *state.Observation) bool {
	switch task.Type {
	case "data_collection":
		// 数据收集任务：成功的crawler、http调用
		return action.Name == "crawler" || action.Name == "http" || action.Name == "http_client"

	case "file_operation":
		// 文件操作任务：成功的fs、file_copy调用
		return action.Name == "fs" || action.Name == "file_copy"

	case "content_generation":
		// 内容生成任务：direct_answer或成功的分析调用
		return action.Name == "direct_answer"

	case "analysis":
		// 分析任务：任何成功的工具调用
		return observation.ErrMsg == ""

	default:
		// 其他任务：任何成功的工具调用
		return observation.ErrMsg == ""
	}
}

// generateTaskCompletionSummary 生成任务完成摘要
func (a *BaseAgent) generateTaskCompletionSummary(plan *TaskPlan, summary map[string]any) string {
	var result strings.Builder

	result.WriteString(fmt.Sprintf("## 任务执行摘要\n\n"))
	result.WriteString(fmt.Sprintf("**原始目标**: %s\n\n", plan.OriginalGoal))

	completionRate := summary["completion_rate"].(float64)
	result.WriteString(fmt.Sprintf("**完成进度**: %.1f%% (%d/%d 个子任务)\n\n",
		completionRate, summary["completed"], summary["total_tasks"]))

	// 列出已完成的任务
	if summary["completed"].(int) > 0 {
		result.WriteString("### ✅ 已完成的任务:\n")
		for _, task := range plan.SubTasks {
			if task.Status == TaskStatusCompleted {
				result.WriteString(fmt.Sprintf("- **%s**: %s\n", task.Title, task.Description))
				if len(task.Evidence) > 0 {
					result.WriteString(fmt.Sprintf("  - 证据: %s\n", strings.Join(task.Evidence, ", ")))
				}
			}
		}
		result.WriteString("\n")
	}

	// 列出失败的任务
	if summary["failed"].(int) > 0 {
		result.WriteString("### ❌ 失败的任务:\n")
		for _, task := range plan.SubTasks {
			if task.Status == TaskStatusFailed {
				result.WriteString(fmt.Sprintf("- **%s**: %s\n", task.Title, task.Description))
			}
		}
		result.WriteString("\n")
	}

	// 列出待完成的任务
	if summary["pending"].(int) > 0 {
		result.WriteString("### ⏳ 待完成的任务:\n")
		for _, task := range plan.SubTasks {
			if task.Status == TaskStatusPending || task.Status == TaskStatusInProgress {
				result.WriteString(fmt.Sprintf("- **%s**: %s\n", task.Title, task.Description))
			}
		}
		result.WriteString("\n")
	}

	// 总结
	if summary["all_completed"].(bool) {
		result.WriteString("🎉 **所有任务已成功完成！**")
	} else {
		result.WriteString("⚠️ **任务未完全完成，请检查失败或待完成的任务。**")
	}

	return result.String()
}

// getStringFromArgs 从参数中获取字符串值
func getStringFromArgs(args map[string]any, key string) string {
	if value, ok := args[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// previewString 返回内容的简要预览
func previewString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// previewAny 对常见输出结构提供预览
func previewAny(m map[string]any) any {
	if m == nil {
		return nil
	}
	if r, ok := m["result"]; ok {
		if rs, ok := r.(string); ok {
			return previewString(rs, 160)
		}
	}
	return m
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
