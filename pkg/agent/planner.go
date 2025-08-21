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

// Planner 规划器
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

// Plan 进行规划，返回下一步动作
func (p *Planner) Plan(ctx context.Context, goal string, trace *state.Trace) (state.Action, error) {
	// 构建系统提示
	systemPrompt := p.buildSystemPrompt()

	// 构建上下文
	contextPrompt := p.buildContextPrompt(goal, trace)

	// 构建工具清单
	toolsPrompt := p.buildToolsPrompt()

	// 构建完整提示（用于调试）
	_ = fmt.Sprintf("%s\n\n%s\n\n%s\n\nPlease decide the next action:",
		systemPrompt, contextPrompt, toolsPrompt)

	// 准备消息
	messages := []llm.Message{
		llm.CreateSystemMessage(systemPrompt),
		llm.CreateUserMessage(contextPrompt + "\n\n" + toolsPrompt + "\n\nPlease decide the next action:"),
	}

	// 准备工具定义
	tools := p.buildLLMTools()

	// 调试信息
	logger.Get().Sugar().Debugf("Tools count: %d", len(tools))
	for i, tool := range tools {
		logger.Get().Sugar().Debugf("Tool %d: %s - %s", i, tool.Function.Name, tool.Function.Description)
	}

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

	// 调试信息
	logger.Get().Sugar().Debugf("LLM Response - ToolCalls: %d, Content: %q, FinishReason: %s",
		len(choice.Message.ToolCalls), choice.Message.Content, choice.FinishReason)

	// 打印完整的响应结构
	if len(choice.Message.ToolCalls) > 0 {
		logger.Get().Sugar().Debugf("ToolCall details: %+v", choice.Message.ToolCalls[0])
	}

	// 处理工具调用
	if len(choice.Message.ToolCalls) > 0 {
		toolCall := choice.Message.ToolCalls[0]
		args, err := llm.ParseToolCallArguments(toolCall.Function.Arguments)
		if err != nil {
			return state.Action{}, fmt.Errorf("failed to parse tool arguments: %w", err)
		}

		return state.Action{
			Name: toolCall.Function.Name,
			Args: args,
		}, nil
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

// buildSystemPrompt 构建系统提示
func (p *Planner) buildSystemPrompt() string {
	return `You are OpenManus-Go, a generalist agent that helps users accomplish their goals.

Your task is to maintain a loop of: Plan -> (Direct Answer | Tool Use) -> Observe -> Reflect -> Decide Next.

Guidelines:
1. When given a goal, break it down into actionable steps
2. Use available tools when necessary to gather information or perform actions
3. Provide direct answers when you have sufficient information
4. Always follow the tool registry strictly and return valid JSON arguments
5. Stop when the user goal is satisfied or no more useful action can be taken

Decision Types:
- DIRECT_ANSWER: Provide a direct response to the user
- USE_TOOL: Call a tool with appropriate arguments
- ASK_CLARIFICATION: Ask for more information from the user
- STOP: Stop execution with a reason

Always respond with either a tool call or a JSON decision in the format:
{"type": "DECISION_TYPE", "content": "response", "reason": "explanation"}`
}

// buildContextPrompt 构建上下文提示
func (p *Planner) buildContextPrompt(goal string, trace *state.Trace) string {
	var context strings.Builder

	context.WriteString(fmt.Sprintf("GOAL: %s\n\n", goal))

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
					// 截断长输出
					output := p.summarizeOutput(step.Observation.Output)
					context.WriteString(fmt.Sprintf("  Result: %s\n", output))
				}
			}
		}
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
