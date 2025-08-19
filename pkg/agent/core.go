package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"openmanus-go/pkg/llm"
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
}

// BaseAgent Agent 的基础实现
type BaseAgent struct {
	llmClient    llm.Client
	toolExecutor *tool.Executor
	planner      *Planner
	memory       *Memory
	reflector    *Reflector
	config       *Config
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

	return &BaseAgent{
		llmClient:    llmClient,
		toolExecutor: toolExecutor,
		planner:      planner,
		memory:       memory,
		reflector:    reflector,
		config:       config,
	}
}

// Plan 进行规划
func (a *BaseAgent) Plan(ctx context.Context, goal string, trace *state.Trace) (state.Action, error) {
	return a.planner.Plan(ctx, goal, trace)
}

// Act 执行动作
func (a *BaseAgent) Act(ctx context.Context, action state.Action) (*state.Observation, error) {
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

	// 检查最近的步骤是否表明应该停止
	if len(trace.Steps) > 0 {
		lastStep := trace.Steps[len(trace.Steps)-1]
		if lastStep.Action.Name == "stop" || lastStep.Action.Name == "direct_answer" {
			return true
		}
	}

	return false
}

// Loop 执行完整的控制循环
func (a *BaseAgent) Loop(ctx context.Context, goal string) (string, error) {
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

	var finalResult string

	for !a.ShouldStop(trace) {
		select {
		case <-ctx.Done():
			trace.Status = state.TraceStatusCanceled
			return "", ctx.Err()
		default:
		}

		// 规划下一步
		action, err := a.Plan(ctx, goal, trace)
		if err != nil {
			trace.Status = state.TraceStatusFailed
			return "", fmt.Errorf("planning failed: %w", err)
		}

		// 添加步骤到轨迹
		_ = trace.AddStep(action)

		// 处理直接回答
		if action.Name == "direct_answer" {
			finalResult = getStringFromArgs(action.Args, "answer")
			trace.Status = state.TraceStatusCompleted
			break
		}

		// 处理停止指令
		if action.Name == "stop" {
			finalResult = getStringFromArgs(action.Args, "reason")
			trace.Status = state.TraceStatusCompleted
			break
		}

		// 执行工具调用
		observation, err := a.Act(ctx, action)
		if err != nil {
			// 执行失败，但继续运行让 Agent 处理错误
			observation = &state.Observation{
				Tool:   action.Name,
				ErrMsg: err.Error(),
			}
		}

		// 更新观测结果
		trace.UpdateObservation(observation)

		// 定期进行反思
		if len(trace.Steps)%a.config.ReflectionSteps == 0 {
			reflection, err := a.Reflect(ctx, trace)
			if err == nil && reflection.ShouldStop {
				finalResult = reflection.Reason
				trace.Status = state.TraceStatusCompleted
				break
			}
		}
	}

	// 如果没有明确的结果，生成总结
	if finalResult == "" {
		finalResult = a.generateSummary(trace)
	}

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
