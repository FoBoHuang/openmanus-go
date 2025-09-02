package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"openmanus-go/pkg/llm"
	"openmanus-go/pkg/logger"
	"openmanus-go/pkg/state"
	"openmanus-go/pkg/tool"
)

// Planner 规划器（统一工具选择策略）
type Planner struct {
	llmClient    llm.Client
	toolRegistry *tool.Registry
}

// NewPlanner 创建规划器
func NewPlanner(llmClient llm.Client, toolRegistry *tool.Registry) *Planner {
	return &Planner{
		llmClient:    llmClient,
		toolRegistry: toolRegistry,
	}
}

// Plan 进行规划，返回下一步动作（统一工具选择策略）
func (p *Planner) Plan(ctx context.Context, goal string, trace *state.Trace) (state.Action, error) {
	// 使用统一的规划流程，LLM从所有可用工具中选择
	// 包括内置工具和MCP工具，无需特殊的优先级逻辑
	return p.standardPlan(ctx, goal, trace)
}

// standardPlan 标准规划流程（原有逻辑）
func (p *Planner) standardPlan(ctx context.Context, goal string, trace *state.Trace) (state.Action, error) {
	// 构建系统提示
	systemPrompt := p.buildSystemPrompt()

	// 构建上下文
	contextPrompt := p.buildContextPrompt(goal, trace)

	// 构建工具清单
	toolsPrompt := p.buildToolsPrompt()

	// 准备消息
	messages := []llm.Message{
		llm.CreateSystemMessage(systemPrompt),
		llm.CreateUserMessage(contextPrompt + "\n\n" + toolsPrompt + "\n\nPlease decide the next action:"),
	}

	// 准备工具定义
	tools := p.buildLLMTools()

	// 创建请求
	req := &llm.ChatRequest{
		Messages:    messages,
		Tools:       tools,
		ToolChoice:  "auto",
		Temperature: 0.1,
	}

	// 发送请求
	resp, err := p.llmClient.Chat(ctx, req)
	if err != nil {
		return state.Action{}, fmt.Errorf("LLM request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return state.Action{}, fmt.Errorf("no response choices")
	}

	choice := resp.Choices[0]

	// 详细的LLM响应日志
	if len(choice.Message.ToolCalls) > 0 {
		logger.Infof("🛠️  [LLM] LLM decided to use a tool")
		toolCall := choice.Message.ToolCalls[0]

		// 获取工具信息以判断类型
		toolInfo := p.getToolInfo(toolCall.Function.Name)
		toolTypeSymbol := "🔧" // 默认内置工具
		toolTypeText := "Built-in"
		if toolInfo != nil && toolInfo.Type == tool.ToolTypeMCP {
			toolTypeSymbol = "🌐"
			toolTypeText = "MCP"
		}

		logger.Infof("🎯 [TOOL] Selected: %s %s (%s tool)", toolTypeSymbol, toolCall.Function.Name, toolTypeText)
		if toolInfo != nil && toolInfo.ServerName != "" {
			logger.Infof("📡 [SERVER] From MCP server: %s", toolInfo.ServerName)
		}

		args, err := llm.ParseToolCallArguments(toolCall.Function.Arguments)
		if err != nil {
			return state.Action{}, fmt.Errorf("failed to parse tool arguments: %w", err)
		}

		// 显示工具参数（如果不太长的话）
		if argsStr := fmt.Sprintf("%v", args); len(argsStr) < 200 {
			logger.Infof("⚙️  [ARGS] Tool arguments: %v", args)
		} else {
			logger.Infof("⚙️  [ARGS] Tool arguments: <long arguments, %d chars>", len(argsStr))
		}

		return state.Action{
			Name: toolCall.Function.Name,
			Args: args,
		}, nil
	} else {
		logger.Infof("💭 [LLM] LLM decided not to use any tools")
		if choice.Message.Content != "" {
			logger.Infof("📝 [RESPONSE] LLM response: %s", truncateString(choice.Message.Content, 150))
		}
	}

	// 处理直接回答
	if choice.Message.Content != "" {
		// 尝试解析为 JSON 决策
		var decision state.Decision
		if err := json.Unmarshal([]byte(choice.Message.Content), &decision); err == nil {
			return p.convertDecisionToAction(decision), nil
		}

		// 否则作为直接回答处理
		return state.Action{
			Name: "direct_answer",
			Args: map[string]any{
				"answer": choice.Message.Content,
			},
		}, nil
	}

	return state.Action{}, fmt.Errorf("no valid response from LLM")
}

// buildSystemPrompt 构建系统提示（统一工具选择策略）
func (p *Planner) buildSystemPrompt() string {
	return `You are OpenManus-Go, a generalist agent that helps users accomplish their goals.

Your task is to maintain a loop of: Plan -> (Direct Answer | Tool Use) -> Observe -> Reflect -> Decide Next.

CRITICAL PRIORITY: If you have data from previous tool calls, FIRST analyze whether this data is sufficient to answer the user's question. If it is sufficient, immediately use direct_answer to provide the answer based on the available data.

Guidelines:
1. **HIGHEST PRIORITY**: When you have data from previous tool calls, analyze it first to see if it answers the user's question
2. If the data is sufficient, provide a direct_answer immediately - don't call more tools
3. Only call additional tools if the existing data is insufficient or incomplete
4. Choose the most appropriate tool from all available tools (both built-in and external tools)
5. All tools are treated equally - select based on functionality, not tool type
6. Always follow the tool registry strictly and return valid JSON arguments
7. Stop when the user goal is satisfied or no more useful action can be taken

Available Tool Types:
- Built-in tools: For local operations (file system, calculations, etc.)
- External tools: For remote data/services (APIs, databases, web services, etc.)

Decision Types:
- DIRECT_ANSWER: Provide a direct response to the user (USE THIS when you have sufficient data)
- USE_TOOL: Call a tool with appropriate arguments (only if more data is needed)
- ASK_CLARIFICATION: Ask for more information from the user
- STOP: Stop execution with a reason

Always respond with either a tool call or a JSON decision in the format:
{"type": "DECISION_TYPE", "content": "response", "reason": "explanation"}`
}

// buildContextPrompt 构建上下文提示
func (p *Planner) buildContextPrompt(goal string, trace *state.Trace) string {
	var context strings.Builder

	context.WriteString(fmt.Sprintf("GOAL: %s\n\n", goal))

	// 检查是否有成功的工具调用数据
	hasSuccessfulToolData := false
	var latestToolData string

	if len(trace.Steps) == 0 {
		context.WriteString("CONTEXT: This is the first step. No previous actions have been taken.\n")
	} else {
		context.WriteString("PREVIOUS STEPS:\n")
		for i, step := range trace.Steps {
			context.WriteString(fmt.Sprintf("Step %d: %s", i+1, step.Action.Name))
			if step.Action.Reason != "" {
				context.WriteString(fmt.Sprintf(" (%s)", step.Action.Reason))
			}
			context.WriteString("\n")

			if step.Observation != nil {
				if step.Observation.ErrMsg != "" {
					context.WriteString(fmt.Sprintf("  Result: ERROR - %s\n", step.Observation.ErrMsg))
				} else {
					// 检查是否是成功的工具调用
					if len(step.Observation.Output) > 0 {
						hasSuccessfulToolData = true
						// 保存最新的成功工具数据
						if rawOutput, ok := step.Observation.Output["content"]; ok {
							latestToolData = fmt.Sprintf("%v", rawOutput)
						} else if rawOutput, ok := step.Observation.Output["result"]; ok {
							latestToolData = fmt.Sprintf("%v", rawOutput)
						} else if jsonBytes, err := json.Marshal(step.Observation.Output); err == nil {
							latestToolData = string(jsonBytes)
						}
					}

					// 截断长输出
					output := p.summarizeOutput(step.Observation.Output)
					context.WriteString(fmt.Sprintf("  Result: %s\n", output))
				}
			}
		}
		context.WriteString("\n")
	}

	// 如果有成功的工具数据，添加分析指导
	if hasSuccessfulToolData {
		context.WriteString("🎯 IMPORTANT - DATA ANALYSIS PRIORITY:\n")
		context.WriteString("You have successfully obtained data from previous tool calls. Your FIRST task is to analyze this data and determine if it's sufficient to answer the user's question.\n")
		context.WriteString("If the data answers the user's question, immediately provide a direct_answer based on this data.\n")
		context.WriteString("Only call additional tools if the existing data is insufficient.\n\n")

		if latestToolData != "" {
			context.WriteString("LATEST TOOL DATA TO ANALYZE:\n")
			context.WriteString(latestToolData)
			context.WriteString("\n\n")
		}
	}

	// 添加最新反思信息
	latestReflection := trace.GetLatestReflection()
	if latestReflection != nil {
		context.WriteString("🤖 LATEST REFLECTION:\n")
		context.WriteString(fmt.Sprintf("- Reason: %s\n", latestReflection.Result.Reason))
		if latestReflection.Result.RevisePlan {
			context.WriteString("- ⚠️ Plan revision suggested\n")
		}
		if latestReflection.Result.NextActionHint != "" {
			context.WriteString(fmt.Sprintf("- 💡 Next action hint: %s\n", latestReflection.Result.NextActionHint))
		}
		context.WriteString(fmt.Sprintf("- Confidence: %.2f\n", latestReflection.Result.Confidence))
		context.WriteString("\n")
	}

	// 添加预算信息
	context.WriteString(fmt.Sprintf("BUDGET: %d/%d steps used\n",
		trace.Budget.UsedSteps, trace.Budget.MaxSteps))

	return context.String()
}

// buildToolsPrompt 构建工具提示
func (p *Planner) buildToolsPrompt() string {
	tools := p.toolRegistry.GetToolsManifest()
	if len(tools) == 0 {
		return "TOOLS: No tools available."
	}

	var prompt strings.Builder
	prompt.WriteString("AVAILABLE TOOLS:\n")

	for _, tool := range tools {
		prompt.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
	}

	return prompt.String()
}

// buildLLMTools 构建 LLM 工具定义
func (p *Planner) buildLLMTools() []llm.Tool {
	toolsManifest := p.toolRegistry.GetToolsManifest()
	llmTools := make([]llm.Tool, 0, len(toolsManifest)+2)

	// 添加注册的工具
	for _, toolInfo := range toolsManifest {
		llmTools = append(llmTools, llm.CreateToolFromToolInfo(
			toolInfo.Name,
			toolInfo.Description,
			toolInfo.InputSchema,
		))
	}

	// 添加特殊工具
	llmTools = append(llmTools, llm.Tool{
		Type: "function",
		Function: llm.ToolFunction{
			Name:        "direct_answer",
			Description: "Provide a direct answer to the user's question",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"answer": map[string]any{
						"type":        "string",
						"description": "The direct answer to provide to the user",
					},
				},
				"required": []string{"answer"},
			},
		},
	})

	llmTools = append(llmTools, llm.Tool{
		Type: "function",
		Function: llm.ToolFunction{
			Name:        "stop",
			Description: "Stop execution with a reason",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"reason": map[string]any{
						"type":        "string",
						"description": "The reason for stopping",
					},
				},
				"required": []string{"reason"},
			},
		},
	})

	return llmTools
}

// summarizeOutput 总结输出内容
func (p *Planner) summarizeOutput(output map[string]any) string {
	if output == nil {
		return "No output"
	}

	// 转换为 JSON 字符串
	jsonBytes, err := json.Marshal(output)
	if err != nil {
		return "Invalid output"
	}

	jsonStr := string(jsonBytes)

	// 如果太长，进行截断
	maxLen := 200
	if len(jsonStr) > maxLen {
		return jsonStr[:maxLen] + "..."
	}

	return jsonStr
}

// convertDecisionToAction 将决策转换为动作
func (p *Planner) convertDecisionToAction(decision state.Decision) state.Action {
	switch decision.Type {
	case state.DecisionDirectAnswer:
		return state.Action{
			Name: "direct_answer",
			Args: map[string]any{
				"answer": decision.Content,
			},
			Reason: decision.Reason,
		}
	case state.DecisionStop:
		return state.Action{
			Name: "stop",
			Args: map[string]any{
				"reason": decision.Content,
			},
			Reason: decision.Reason,
		}
	case state.DecisionAskClarification:
		return state.Action{
			Name: "direct_answer",
			Args: map[string]any{
				"answer": "I need more information: " + decision.Content,
			},
			Reason: decision.Reason,
		}
	case state.DecisionUseTool:
		if decision.Action != nil {
			return *decision.Action
		}
		// 如果没有具体动作，返回停止
		return state.Action{
			Name: "stop",
			Args: map[string]any{
				"reason": "No tool action specified",
			},
		}
	default:
		return state.Action{
			Name: "stop",
			Args: map[string]any{
				"reason": "Unknown decision type",
			},
		}
	}
}

// getToolInfo 获取工具信息
func (p *Planner) getToolInfo(toolName string) *tool.ToolInfo {
	manifest := p.toolRegistry.GetToolsManifest()
	for _, toolInfo := range manifest {
		if toolInfo.Name == toolName {
			return &toolInfo
		}
	}
	return nil
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
