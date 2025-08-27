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

// TaskCompletionAnalyzer 任务完成度分析器
type TaskCompletionAnalyzer struct {
	llmClient llm.Client
}

// NewTaskCompletionAnalyzer 创建任务完成度分析器
func NewTaskCompletionAnalyzer(llmClient llm.Client) *TaskCompletionAnalyzer {
	return &TaskCompletionAnalyzer{
		llmClient: llmClient,
	}
}

// TaskCompletionResult 任务完成分析结果
type TaskCompletionResult struct {
	IsComplete      bool     `json:"is_complete"`      // 任务是否完成
	CompletedTasks  []string `json:"completed_tasks"`  // 已完成的子任务
	PendingTasks    []string `json:"pending_tasks"`    // 待完成的子任务
	Confidence      float64  `json:"confidence"`       // 完成度置信度 (0-1)
	Reason          string   `json:"reason"`           // 分析原因
	SuggestedAction string   `json:"suggested_action"` // 建议的下一步行动
}

// AnalyzeTaskCompletion 分析任务完成情况
func (t *TaskCompletionAnalyzer) AnalyzeTaskCompletion(ctx context.Context, goal string, trace *state.Trace) (*TaskCompletionResult, error) {
	// 构建分析提示
	systemPrompt := t.buildAnalysisSystemPrompt()
	userPrompt := t.buildAnalysisUserPrompt(goal, trace)

	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	logger.Get().Sugar().Debugw("task.completion.analysis.request", "goal", goal, "steps", len(trace.Steps))

	// 调用LLM进行分析
	req := &llm.ChatRequest{
		Messages:    messages,
		Temperature: 0.1,
		MaxTokens:   2000,
	}
	response, err := t.llmClient.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze task completion: %w", err)
	}

	// 获取响应内容
	var responseContent string
	if len(response.Choices) > 0 {
		responseContent = response.Choices[0].Message.Content
	} else {
		return nil, fmt.Errorf("no response choices received from LLM")
	}

	logger.Get().Sugar().Debugw("task.completion.analysis.response", "content", responseContent)

	// 尝试提取JSON内容（可能被包裹在代码块中）
	jsonContent := t.extractJSONFromResponse(responseContent)

	// 解析响应
	result := &TaskCompletionResult{}
	if err := json.Unmarshal([]byte(jsonContent), result); err != nil {
		// 如果JSON解析失败，尝试提取关键信息
		logger.Get().Sugar().Warnw("Failed to parse completion analysis JSON, using fallback", "error", err, "content", jsonContent)
		return t.fallbackAnalysis(goal, trace, responseContent), nil
	}

	logger.Get().Sugar().Infow("task.completion.analysis.result",
		"is_complete", result.IsComplete,
		"completed_count", len(result.CompletedTasks),
		"pending_count", len(result.PendingTasks),
		"confidence", result.Confidence)

	return result, nil
}

// extractJSONFromResponse 从响应中提取JSON内容，处理可能的代码块包装
func (t *TaskCompletionAnalyzer) extractJSONFromResponse(response string) string {
	// 尝试匹配 ```json...``` 代码块
	if strings.Contains(response, "```json") {
		start := strings.Index(response, "```json")
		if start != -1 {
			start += 7 // 跳过 "```json"
			end := strings.Index(response[start:], "```")
			if end != -1 {
				return strings.TrimSpace(response[start : start+end])
			}
		}
	}

	// 尝试匹配 ```...``` 代码块（无json标识）
	if strings.Count(response, "```") >= 2 {
		start := strings.Index(response, "```")
		if start != -1 {
			start += 3
			// 跳过可能的语言标识符
			if newlineIdx := strings.Index(response[start:], "\n"); newlineIdx != -1 {
				start += newlineIdx + 1
			}
			end := strings.Index(response[start:], "```")
			if end != -1 {
				content := strings.TrimSpace(response[start : start+end])
				// 检查内容是否看起来像JSON
				if strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}") {
					return content
				}
			}
		}
	}

	// 如果没有代码块，直接返回原内容
	return strings.TrimSpace(response)
}

// buildAnalysisSystemPrompt 构建分析系统提示
func (t *TaskCompletionAnalyzer) buildAnalysisSystemPrompt() string {
	return `You are a Task Completion Analyzer. Your job is to carefully analyze whether a user's goal has been fully achieved based on the actions taken.

Your analysis should consider:
1. **Multi-step Tasks**: If the goal contains multiple requirements (e.g., "analyze data AND save to file"), ALL parts must be completed
2. **Action Verification**: Check if the intended actions were actually executed successfully
3. **Output Requirements**: If the goal asks for specific outputs (files, reports, etc.), verify they were created
4. **Logical Completeness**: Ensure the sequence of actions logically fulfills the entire goal

Return your analysis as a JSON object with this exact format:
{
  "is_complete": boolean,
  "completed_tasks": ["list", "of", "completed", "subtasks"],
  "pending_tasks": ["list", "of", "remaining", "subtasks"],
  "confidence": 0.0-1.0,
  "reason": "detailed explanation of your analysis",
  "suggested_action": "what should be done next (if anything)"
}

CRITICAL: Only mark is_complete=true if EVERY requirement in the goal has been satisfied. Be strict about this evaluation.`
}

// buildAnalysisUserPrompt 构建分析用户提示
func (t *TaskCompletionAnalyzer) buildAnalysisUserPrompt(goal string, trace *state.Trace) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("**GOAL TO ANALYZE**: %s\n\n", goal))

	if len(trace.Steps) == 0 {
		prompt.WriteString("**ACTIONS TAKEN**: None - this is the initial state.\n\n")
	} else {
		prompt.WriteString("**ACTIONS TAKEN**:\n")
		for i, step := range trace.Steps {
			prompt.WriteString(fmt.Sprintf("%d. Action: %s", i+1, step.Action.Name))

			if step.Action.Reason != "" {
				prompt.WriteString(fmt.Sprintf(" (Reason: %s)", step.Action.Reason))
			}

			if step.Observation != nil {
				if step.Observation.ErrMsg != "" {
					prompt.WriteString(fmt.Sprintf(" → ❌ FAILED: %s", step.Observation.ErrMsg))
				} else {
					// 显示简短的输出预览
					outputStr := fmt.Sprintf("%v", step.Observation.Output)
					if len(outputStr) > 200 {
						outputStr = outputStr[:200] + "..."
					}
					prompt.WriteString(fmt.Sprintf(" → ✅ SUCCESS: %s", outputStr))
				}
			}
			prompt.WriteString("\n")
		}
		prompt.WriteString("\n")
	}

	// 添加特殊指导
	prompt.WriteString("**ANALYSIS INSTRUCTIONS**:\n")
	prompt.WriteString("1. Break down the goal into individual requirements\n")
	prompt.WriteString("2. For each requirement, check if it has been successfully completed\n")
	prompt.WriteString("3. Pay special attention to file creation, data processing, and output generation tasks\n")
	prompt.WriteString("4. If ANY requirement is incomplete, the overall task is incomplete\n")
	prompt.WriteString("5. Provide your analysis in the specified JSON format\n")

	return prompt.String()
}

// fallbackAnalysis 备用分析方法，当JSON解析失败时使用
func (t *TaskCompletionAnalyzer) fallbackAnalysis(goal string, trace *state.Trace, responseContent string) *TaskCompletionResult {
	// 简单的启发式分析
	result := &TaskCompletionResult{
		IsComplete:      false,
		CompletedTasks:  []string{},
		PendingTasks:    []string{goal},
		Confidence:      0.2,
		Reason:          "Fallback analysis due to LLM response parsing error",
		SuggestedAction: "continue_execution",
	}

	// 分析目标中的关键动作
	goalLower := strings.ToLower(goal)
	isMultiStepTask := false
	requiredActions := []string{}

	// 检测常见的多步任务模式
	if strings.Contains(goalLower, "并") || strings.Contains(goalLower, "然后") || strings.Contains(goalLower, "and") {
		isMultiStepTask = true
	}

	if strings.Contains(goalLower, "保存") || strings.Contains(goalLower, "写入") || strings.Contains(goalLower, "文件") {
		requiredActions = append(requiredActions, "file_creation")
	}

	if strings.Contains(goalLower, "总结") || strings.Contains(goalLower, "分析") || strings.Contains(goalLower, "生成") {
		requiredActions = append(requiredActions, "content_generation")
	}

	// 检查是否有成功的步骤
	if len(trace.Steps) > 0 {
		var successfulActions []string
		hasFileCreation := false
		hasContentGeneration := false
		hasFailures := false

		for _, step := range trace.Steps {
			if step.Observation != nil && step.Observation.ErrMsg == "" {
				successfulActions = append(successfulActions, step.Action.Name)

				// 检查具体的动作类型
				if step.Action.Name == "fs" || step.Action.Name == "file_copy" {
					hasFileCreation = true
				}
				if step.Action.Name == "direct_answer" || step.Action.Name == "crawler" {
					hasContentGeneration = true
				}
			} else if step.Observation != nil && step.Observation.ErrMsg != "" {
				hasFailures = true
			}
		}

		result.CompletedTasks = successfulActions

		// 对于多步任务，需要所有关键动作都完成
		if isMultiStepTask {
			missingActions := []string{}

			for _, required := range requiredActions {
				switch required {
				case "file_creation":
					if !hasFileCreation {
						missingActions = append(missingActions, "文件创建/保存")
					}
				case "content_generation":
					if !hasContentGeneration {
						missingActions = append(missingActions, "内容生成/总结")
					}
				}
			}

			if len(missingActions) == 0 && len(successfulActions) > 0 {
				result.IsComplete = true
				result.Confidence = 0.6
				result.SuggestedAction = "direct_answer"
				result.PendingTasks = []string{}
			} else {
				result.PendingTasks = missingActions
				result.Confidence = 0.3
			}
		} else {
			// 单步任务：有成功操作且没有失败
			if len(successfulActions) > 0 && !hasFailures {
				result.Confidence = 0.5

				// 检查响应内容中的完成指示
				lowerResponse := strings.ToLower(responseContent)
				if strings.Contains(lowerResponse, "complete") ||
					strings.Contains(lowerResponse, "finished") ||
					strings.Contains(lowerResponse, "done") ||
					strings.Contains(lowerResponse, "完成") {
					result.IsComplete = true
					result.PendingTasks = []string{}
					result.SuggestedAction = "direct_answer"
				}
			}
		}
	}

	logger.Get().Sugar().Debugw("fallback.analysis.result",
		"is_multi_step", isMultiStepTask,
		"required_actions", requiredActions,
		"is_complete", result.IsComplete,
		"confidence", result.Confidence)

	return result
}
