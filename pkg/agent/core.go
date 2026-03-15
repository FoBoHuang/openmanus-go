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
	mcpExecutor  *MCPExecutor // MCP 执行器
}

// Config Agent 配置
type Config struct {
	MaxSteps        int           `json:"max_steps" mapstructure:"max_steps"`
	MaxTokens       int           `json:"max_tokens" mapstructure:"max_tokens"`
	MaxDuration     time.Duration `json:"max_duration" mapstructure:"max_duration"`
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
		ReflectionSteps: 3,
		MaxRetries:      2,
		RetryBackoff:    time.Second,
	}
}

// ConfigFromAppConfig 从应用配置创建 Agent 配置
func ConfigFromAppConfig(appConfig *config.Config) (*Config, error) {
	if appConfig == nil {
		return DefaultConfig(), nil
	}

	agentConfig := DefaultConfig()

	// 转换基本字段
	if appConfig.Agent.MaxSteps > 0 {
		agentConfig.MaxSteps = appConfig.Agent.MaxSteps
	}
	if appConfig.Agent.MaxTokens > 0 {
		agentConfig.MaxTokens = appConfig.Agent.MaxTokens
	}
	if appConfig.Agent.ReflectionSteps > 0 {
		agentConfig.ReflectionSteps = appConfig.Agent.ReflectionSteps
	}
	if appConfig.Agent.MaxRetries > 0 {
		agentConfig.MaxRetries = appConfig.Agent.MaxRetries
	}

	// 转换持续时间字段
	if appConfig.Agent.MaxDuration != "" {
		duration, err := time.ParseDuration(appConfig.Agent.MaxDuration)
		if err != nil {
			return nil, fmt.Errorf("invalid max_duration: %w", err)
		}
		agentConfig.MaxDuration = duration
	}

	if appConfig.Agent.RetryBackoff != "" {
		backoff, err := time.ParseDuration(appConfig.Agent.RetryBackoff)
		if err != nil {
			return nil, fmt.Errorf("invalid retry_backoff: %w", err)
		}
		agentConfig.RetryBackoff = backoff
	}

	return agentConfig, nil
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
	memory := NewMemoryWithConfig(DefaultMemoryConfig())
	planner := NewPlanner(llmClient, toolRegistry, memory)
	reflector := NewReflector(llmClient, memory)

	return &BaseAgent{
		llmClient:    llmClient,
		toolExecutor: toolExecutor,
		planner:      planner,
		memory:       memory,
		reflector:    reflector,
		config:       config,
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

	// 创建基础组件（使用配置中的 MemoryPath 实现长期记忆持久化）
	memCfg := DefaultMemoryConfig()
	if appConfig != nil && appConfig.Agent.MemoryPath != "" {
		memCfg.LongTermPath = appConfig.Agent.MemoryPath
	}
	memory := NewMemoryWithConfig(memCfg)
	reflector := NewReflector(llmClient, memory)

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
	planner := NewPlanner(llmClient, toolRegistry, memory) // 使用统一的规划器，传入 Memory

	return &BaseAgent{
		llmClient:    llmClient,
		toolExecutor: toolExecutor,
		planner:      planner,
		memory:       memory,
		reflector:    reflector,
		config:       agentConfig,
		mcpExecutor:  mcpExecutor, // 保留引用用于清理
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

// Loop 执行完整的控制循环（统一线性执行策略）
func (a *BaseAgent) Loop(ctx context.Context, goal string) (string, error) {
	// 使用统一的线性执行策略，与 OpenManus 保持一致
	logger.Infow("agent.loop.unified_execution", "goal", goal)
	return a.unifiedLoop(ctx, goal)
}

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

	// 将轨迹保存到memory中
	a.memory.SetCurrentTrace(trace)

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
		logger.Infof("═══════════════════════════════════════════════════════════════")
		logger.Infof("🤔 [STEP %d/%d] Planning next action...", stepNum, a.config.MaxSteps)
		logger.Infof("⏱️  [PROGRESS] %.1f%% complete | Elapsed: %v",
			float64(stepNum-1)/float64(a.config.MaxSteps)*100,
			time.Since(trace.Budget.StartTime).Round(time.Second))

		// 打印当前状态和目标
		logger.Infof("🎯 [CURRENT_GOAL] %s", goal)
		if len(trace.Steps) > 0 {
			lastStep := trace.Steps[len(trace.Steps)-1]
			if lastStep.Observation != nil {
				if lastStep.Observation.ErrMsg != "" {
					logger.Infof("📋 [LAST_RESULT] ❌ Failed: %s", lastStep.Observation.ErrMsg)
				} else {
					logger.Infof("📋 [LAST_RESULT] ✅ Success: %s", a.summarizeObservation(lastStep.Observation))
				}
			}
		}

		logger.Infof("🤔 [PLANNING] Starting reasoning process...")

		// 1. Plan: 规划下一步动作
		action, err := a.Plan(ctx, goal, trace)
		if err != nil {
			trace.Status = state.TraceStatusFailed
			logger.Errorf("❌ [PLAN] Planning failed: %v", err)
			return "", fmt.Errorf("planning failed: %w", err)
		}

		// 打印规划结果
		logger.Infof("✅ [PLAN_COMPLETE] Decision made: %s", action.Name)
		if action.Reason != "" {
			logger.Infof("💭 [PLAN_REASON] %s", action.Reason)
		}

		// 添加步骤到轨迹
		_ = trace.AddStep(action)

		// 处理直接回答 - 简化处理，直接接受
		if action.Name == "direct_answer" {
			potentialResult := getStringFromArgs(action.Args, "answer")
			finalResult = potentialResult
			trace.Status = state.TraceStatusCompleted
			logger.Infof("✅ [DIRECT_ANSWER] Task completed with direct answer:")
			logger.Infof("📝 [ANSWER_CONTENT] %s", potentialResult)
			break
		}

		// 处理停止指令
		if action.Name == "stop" {
			finalResult = getStringFromArgs(action.Args, "reason")
			trace.Status = state.TraceStatusCompleted
			logger.Infof("🛑 [STOP_EXECUTION] Stopping execution")
			logger.Infof("📝 [STOP_REASON] %s", finalResult)
			break
		}

		// 执行工具调用 - 详细信息
		logger.Infof("⚡ [EXECUTION_START] Preparing to execute: %s", action.Name)
		logger.Infof("🔧 [TOOL_PARAMS] Parameters:")
		for key, value := range action.Args {
			if valueStr := fmt.Sprintf("%v", value); len(valueStr) > 100 {
				logger.Infof("    %s: <%s, %d chars>", key, a.getValueType(value), len(valueStr))
			} else {
				logger.Infof("    %s: %v", key, value)
			}
		}
		logger.Infof("⚡ [EXECUTING] Running %s now...", action.Name)
		// 2. Act: 执行动作
		observation, err := a.Act(ctx, action)

		if err != nil {
			// 执行失败，但继续运行让 Agent 处理错误
			observation = &state.Observation{
				Tool:   action.Name,
				ErrMsg: err.Error(),
			}
			logger.Warnf("⚠️  [EXECUTION_ERROR] Tool execution failed: %v", err)
		}

		// 更新观测结果
		trace.UpdateObservation(observation)

		// 生成步骤摘要，用于轨迹压缩时保留关键信息
		trace.UpdateSummary(a.summarizeStep(action, observation))

		// 详细记录执行结果并学习
		logger.Infof("📊 [EXECUTION_COMPLETE] Tool execution finished: %s", action.Name)
		logger.Infof("⏱️  [EXECUTION_TIME] Latency: %d ms", observation.Latency)

		if observation.ErrMsg != "" {
			logger.Warnf("❌ [EXECUTION_FAILED] %s failed with error:", action.Name)
			logger.Warnf("📝 [ERROR_DETAILS] %s", observation.ErrMsg)
			logger.Warnf("🤖 [LEARNING] Recording failure pattern for future reference")
			// 学习失败模式，避免重复犯错
			a.memory.AddContextualInfo(fmt.Sprintf("failed_%s_reasons", action.Name), observation.ErrMsg)
		} else {
			logger.Infof("✅ [EXECUTION_SUCCESS] %s completed successfully", action.Name)
			logger.Infof("📊 [OUTPUT_ANALYSIS] Processing execution results...")

			if observation.Output != nil {
				logger.Infof("📝 [OUTPUT_DETAILS] Result data:")
				for key, value := range observation.Output {
					if valueStr := fmt.Sprintf("%v", value); len(valueStr) > 100 {
						logger.Infof("    %s: <%s, %d chars>", key, a.getValueType(value), len(valueStr))
					} else {
						logger.Infof("    %s: %v", key, value)
					}
				}
			} else {
				logger.Infof("📝 [OUTPUT_DETAILS] No output data returned")
			}

			logger.Infof("🤖 [LEARNING] Recording success pattern for future reference")
			// 学习成功模式
			a.memory.AddContextualInfo(fmt.Sprintf("successful_%s_pattern", action.Name), map[string]any{
				"args":   action.Args,
				"output": observation.Output,
			})
		}

		// 更新记忆中的轨迹指标
		a.memory.UpdateTraceMetrics()

		// 定期进行反思
		if a.config.ReflectionSteps > 0 && len(trace.Steps)%a.config.ReflectionSteps == 0 {
			logger.Infof("")
			logger.Infof("🤖 [REFLECTION_START] Performing reflection after %d steps...", len(trace.Steps))
			logger.Infof("🔍 [ANALYZING] Analyzing execution patterns and progress...")

			// 3. Reflect: 定期反思（每N步）
			reflectionResult, err := a.Reflect(ctx, trace)
			if err != nil {
				logger.Warnf("⚠️  [REFLECTION_ERROR] Reflection failed: %v", err)
			} else {
				// 将反思结果保存到轨迹中
				trace.AddReflection(reflectionResult)

				logger.Infof("🧠 [REFLECTION_COMPLETE] Analysis finished")
				logger.Infof("💭 [REFLECTION_REASON] %s", reflectionResult.Reason)
				logger.Infof("📊 [CONFIDENCE] %.2f", reflectionResult.Confidence)

				// 如果反思建议停止，则停止执行
				if reflectionResult.ShouldStop {
					finalResult = fmt.Sprintf("Execution stopped based on reflection: %s", reflectionResult.Reason)
					trace.Status = state.TraceStatusCompleted
					logger.Infof("🛑 [REFLECTION_STOP] Stopping execution based on reflection")
					logger.Infof("📝 [STOP_REASON] %s", reflectionResult.Reason)
					break
				}

				// 如果反思建议修改计划，记录提示信息
				if reflectionResult.RevisePlan {
					logger.Infof("🔄 [PLAN_REVISION] Plan revision suggested")
					logger.Infof("💡 [REVISION_HINT] %s", reflectionResult.NextActionHint)
				}

				// 如果有下一步提示，记录下来
				if reflectionResult.NextActionHint != "" && !reflectionResult.RevisePlan {
					logger.Infof("💡 [NEXT_ACTION_HINT] %s", reflectionResult.NextActionHint)
				}

				logger.Infof("🤖 [REFLECTION_END] Continuing with execution...")
			}
		}

		// 定期压缩轨迹以节省内存，同时清理过期短期记忆
		if len(trace.Steps) > 20 && len(trace.Steps)%10 == 0 {
			logger.Infof("🗜️  [MEMORY] Compressing trace to maintain efficiency...")
			a.memory.CompressTrace(15)
			if cleaned := a.memory.CleanExpiredShortTerm(); cleaned > 0 {
				logger.Infof("🧹 [MEMORY] Cleaned %d expired short-term entries", cleaned)
			}
		}

		// 检查预算
		if trace.IsExceededBudget() {
			trace.Status = state.TraceStatusFailed // 使用现有的状态
			finalResult = fmt.Sprintf("Execution stopped due to budget limits. Completed %d steps.", len(trace.Steps))
			logger.Warnf("💰 [BUDGET] Execution stopped due to budget limits")
			break
		}
	}

	// 持久化长期记忆
	if err := a.memory.FlushLongTerm(); err != nil {
		logger.Warnf("⚠️  [MEMORY] Failed to flush long-term memory: %v", err)
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

// summarizeObservation 总结观测结果
func (a *BaseAgent) summarizeObservation(obs *state.Observation) string {
	if obs.Output == nil {
		return "No output"
	}

	// 尝试提取关键信息
	if result, ok := obs.Output["result"].(string); ok {
		if len(result) > 100 {
			return result[:100] + "..."
		}
		return result
	}

	if success, ok := obs.Output["success"].(bool); ok {
		if success {
			return "Operation completed successfully"
		} else {
			if errMsg, ok := obs.Output["error"].(string); ok {
				return fmt.Sprintf("Operation failed: %s", errMsg)
			}
			return "Operation failed"
		}
	}

	// 默认总结
	return "Operation completed"
}

// summarizeStep 基于 Action 和 Observation 生成一句话步骤摘要
func (a *BaseAgent) summarizeStep(action state.Action, obs *state.Observation) string {
	if obs == nil {
		return fmt.Sprintf("[%s] No observation", action.Name)
	}
	if obs.ErrMsg != "" {
		errPreview := obs.ErrMsg
		if len(errPreview) > 80 {
			errPreview = errPreview[:80] + "..."
		}
		return fmt.Sprintf("[%s] Failed: %s", action.Name, errPreview)
	}
	return fmt.Sprintf("[%s] Success: %s", action.Name, a.summarizeObservation(obs))
}

// getValueType 获取值的类型描述
func (a *BaseAgent) getValueType(value any) string {
	switch value.(type) {
	case string:
		return "string"
	case int, int32, int64:
		return "integer"
	case float32, float64:
		return "number"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return "unknown"
	}
}
