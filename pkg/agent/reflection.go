package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"openmanus-go/pkg/llm"
	"openmanus-go/pkg/logger"
	"openmanus-go/pkg/state"
)

// Reflector 反思器
type Reflector struct {
	llmClient llm.Client
	memory    *Memory // 添加内存引用
}

// NewReflector 创建反思器
func NewReflector(llmClient llm.Client, memory *Memory) *Reflector {
	return &Reflector{
		llmClient: llmClient,
		memory:    memory,
	}
}

// Reflect 进行反思分析
func (r *Reflector) Reflect(ctx context.Context, trace *state.Trace) (*state.ReflectionResult, error) {
	// 构建反思提示
	prompt := r.buildReflectionPrompt(trace)
	logger.Debugw("agent.reflect.request", "steps", len(trace.Steps), "status", trace.Status)

	// 准备消息
	messages := []llm.Message{
		llm.CreateSystemMessage(r.getSystemPrompt()),
		llm.CreateUserMessage(prompt),
	}

	// 创建请求
	req := &llm.ChatRequest{
		Messages:    messages,
		Temperature: 0.1, // 低温度确保一致性
	}

	// 发送请求
	resp, err := r.llmClient.Chat(ctx, req)
	if err != nil {
		logger.Errorw("agent.reflect.llm_error", "error", err)
		return nil, fmt.Errorf("reflection LLM request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices from reflection")
	}

	// 解析响应
	content := resp.Choices[0].Message.Content
	var result state.ReflectionResult

	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// 如果解析失败，创建默认结果
		logger.Warnw("agent.reflect.parse_failed", "content_preview", preview(content, 200))
		return &state.ReflectionResult{
			RevisePlan:     false,
			NextActionHint: content,
			ShouldStop:     false,
			Reason:         "Failed to parse reflection response",
			Confidence:     0.5,
		}, nil
	}

	logger.Debugw("agent.reflect.ok", "revise_plan", result.RevisePlan, "should_stop", result.ShouldStop, "confidence", result.Confidence)

	return &result, nil
}

// getSystemPrompt 获取系统提示
func (r *Reflector) getSystemPrompt() string {
	return `You are a reflection module for an AI agent. Your job is to analyze the agent's execution trace and provide insights about progress, potential issues, and next steps.

Analyze the given execution trace and respond with a JSON object containing:
{
  "revise_plan": boolean,     // Whether the current plan should be revised
  "next_action_hint": string, // Suggestion for the next action
  "should_stop": boolean,     // Whether execution should stop
  "reason": string,          // Explanation for the recommendation
  "confidence": number       // Confidence level (0.0 to 1.0)
}

Consider:
1. Are we making progress toward the goal?
2. Are there repeated failures or loops?
3. Is there missing information or context?
4. Are we using the right tools and approach?
5. Should we try a different strategy?

Be concise but thorough in your analysis.`
}

// buildReflectionPrompt 构建反思提示（增强版，使用 Memory 分析）
func (r *Reflector) buildReflectionPrompt(trace *state.Trace) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("GOAL: %s\n\n", trace.Goal))

	// 添加执行统计
	prompt.WriteString("EXECUTION SUMMARY:\n")
	prompt.WriteString(fmt.Sprintf("- Total steps: %d/%d\n", len(trace.Steps), trace.Budget.MaxSteps))
	prompt.WriteString(fmt.Sprintf("- Status: %s\n", trace.Status))

	// 使用 Memory 获取更精确的分析
	var successCount, failureCount int
	if r.memory != nil {
		r.memory.UpdateTraceMetrics() // 更新指标
		successfulSteps := r.memory.GetSuccessfulSteps()
		failedSteps := r.memory.GetFailedSteps()
		successCount = len(successfulSteps)
		failureCount = len(failedSteps)
	} else {
		// fallback 到原有逻辑
		successCount, failureCount = r.analyzeStepOutcomes(trace.Steps)
	}

	totalSteps := len(trace.Steps)
	if totalSteps > 0 {
		successRate := float64(successCount) / float64(totalSteps) * 100
		prompt.WriteString(fmt.Sprintf("- Success rate: %.1f%% (%d/%d successful, %d failed)\n", successRate, successCount, totalSteps, failureCount))
	}

	// 添加 Memory 提供的智能分析
	if r.memory != nil {
		memorySummary := r.memory.GetSummary()
		if metrics, ok := memorySummary["metrics"].(map[string]any); ok {
			if updatedAt, ok := metrics["updated_at"]; ok {
				prompt.WriteString(fmt.Sprintf("- Last metrics update: %v\n", updatedAt))
			}
		}

		// 添加历史压缩信息
		if trace.Scratch != nil {
			if compressedHistory, ok := trace.Scratch["compressed_history"].(map[string]any); ok {
				if keyOutcomes, ok := compressedHistory["key_outcomes"].([]string); ok && len(keyOutcomes) > 0 {
					prompt.WriteString("- Key outcomes from compressed history:\n")
					for _, outcome := range keyOutcomes {
						prompt.WriteString(fmt.Sprintf("  • %s\n", outcome))
					}
				}
			}
		}
	}

	prompt.WriteString("\n")

	// 添加最近的步骤
	recentSteps := r.getRecentSteps(trace.Steps, 5)
	if len(recentSteps) > 0 {
		prompt.WriteString("RECENT STEPS:\n")
		for i, step := range recentSteps {
			prompt.WriteString(fmt.Sprintf("%d. Action: %s", len(trace.Steps)-len(recentSteps)+i+1, step.Action.Name))

			if step.Action.Reason != "" {
				prompt.WriteString(fmt.Sprintf(" (Reason: %s)", step.Action.Reason))
			}
			prompt.WriteString("\n")

			if step.Observation != nil {
				if step.Observation.ErrMsg != "" {
					prompt.WriteString(fmt.Sprintf("   Result: FAILED - %s\n", step.Observation.ErrMsg))
				} else {
					summary := r.summarizeObservation(step.Observation)
					prompt.WriteString(fmt.Sprintf("   Result: SUCCESS - %s\n", summary))
				}
			}
		}
		prompt.WriteString("\n")
	}

	// 分析模式和问题
	patterns := r.analyzePatterns(trace.Steps)
	if len(patterns) > 0 {
		prompt.WriteString("PATTERNS DETECTED:\n")
		for _, pattern := range patterns {
			prompt.WriteString(fmt.Sprintf("- %s\n", pattern))
		}
		prompt.WriteString("\n")
	}

	// 添加预算状态
	if trace.IsExceededBudget() {
		prompt.WriteString("WARNING: Budget limits have been exceeded!\n\n")
	}

	prompt.WriteString("Please analyze this execution trace and provide your reflection.")

	return prompt.String()
}

func preview(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// analyzeStepOutcomes 分析步骤结果
func (r *Reflector) analyzeStepOutcomes(steps []state.Step) (successCount, failureCount int) {
	for _, step := range steps {
		if step.Observation != nil {
			if step.Observation.ErrMsg != "" {
				failureCount++
			} else {
				successCount++
			}
		}
	}
	return successCount, failureCount
}

// getRecentSteps 获取最近的步骤
func (r *Reflector) getRecentSteps(steps []state.Step, count int) []state.Step {
	if len(steps) <= count {
		return steps
	}
	return steps[len(steps)-count:]
}

// summarizeObservation 总结观测结果
func (r *Reflector) summarizeObservation(obs *state.Observation) string {
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

// analyzePatterns 分析执行模式
func (r *Reflector) analyzePatterns(steps []state.Step) []string {
	var patterns []string

	if len(steps) < 2 {
		return patterns
	}

	// 检查重复动作
	actionCounts := make(map[string]int)
	recentActions := make([]string, 0, 5)

	for _, step := range steps {
		actionCounts[step.Action.Name]++

		// 记录最近的动作
		if len(recentActions) >= 5 {
			recentActions = recentActions[1:]
		}
		recentActions = append(recentActions, step.Action.Name)
	}

	// 检查重复失败
	for action, count := range actionCounts {
		if count > 3 {
			patterns = append(patterns, fmt.Sprintf("Repeated use of '%s' (%d times)", action, count))
		}
	}

	// 检查循环模式
	if len(recentActions) >= 4 {
		if r.detectLoop(recentActions) {
			patterns = append(patterns, "Potential loop detected in recent actions")
		}
	}

	// 检查连续失败
	consecutiveFailures := 0
	for i := len(steps) - 1; i >= 0 && i >= len(steps)-5; i-- {
		if steps[i].Observation != nil && steps[i].Observation.ErrMsg != "" {
			consecutiveFailures++
		} else {
			break
		}
	}

	if consecutiveFailures >= 3 {
		patterns = append(patterns, fmt.Sprintf("Consecutive failures detected (%d in a row)", consecutiveFailures))
	}

	return patterns
}

// detectLoop 检测循环模式
func (r *Reflector) detectLoop(actions []string) bool {
	if len(actions) < 4 {
		return false
	}

	// 检查简单的 2-步循环
	if len(actions) >= 4 {
		if actions[len(actions)-1] == actions[len(actions)-3] &&
			actions[len(actions)-2] == actions[len(actions)-4] {
			return true
		}
	}

	return false
}

// QuickReflect 快速反思（不使用 LLM）
func (r *Reflector) QuickReflect(trace *state.Trace) *state.ReflectionResult {
	if len(trace.Steps) == 0 {
		return &state.ReflectionResult{
			RevisePlan:     false,
			NextActionHint: "Start executing the plan",
			ShouldStop:     false,
			Reason:         "No steps executed yet",
			Confidence:     1.0,
		}
	}

	// 检查预算
	if trace.IsExceededBudget() {
		return &state.ReflectionResult{
			RevisePlan:     false,
			NextActionHint: "Stop due to budget exceeded",
			ShouldStop:     true,
			Reason:         "Budget limits exceeded",
			Confidence:     1.0,
		}
	}

	// 检查连续失败
	recentSteps := r.getRecentSteps(trace.Steps, 3)
	allFailed := true
	for _, step := range recentSteps {
		if step.Observation == nil || step.Observation.ErrMsg == "" {
			allFailed = false
			break
		}
	}

	if allFailed && len(recentSteps) >= 3 {
		return &state.ReflectionResult{
			RevisePlan:     true,
			NextActionHint: "Try a different approach or tool",
			ShouldStop:     false,
			Reason:         "Multiple consecutive failures detected",
			Confidence:     0.8,
		}
	}

	// 默认继续执行
	return &state.ReflectionResult{
		RevisePlan:     false,
		NextActionHint: "Continue with the current plan",
		ShouldStop:     false,
		Reason:         "Execution is progressing normally",
		Confidence:     0.7,
	}
}
