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
	mcpDiscovery *MCPDiscoveryService
	mcpSelector  *MCPToolSelector
	mcpExecutor  *MCPExecutor
}

// NewPlanner 创建规划器
func NewPlanner(llmClient llm.Client, toolRegistry *tool.Registry) *Planner {
	return &Planner{
		llmClient:    llmClient,
		toolRegistry: toolRegistry,
	}
}

// NewPlannerWithMCP 创建带 MCP 功能的规划器
func NewPlannerWithMCP(llmClient llm.Client, toolRegistry *tool.Registry, mcpDiscovery *MCPDiscoveryService, mcpSelector *MCPToolSelector, mcpExecutor *MCPExecutor) *Planner {
	return &Planner{
		llmClient:    llmClient,
		toolRegistry: toolRegistry,
		mcpDiscovery: mcpDiscovery,
		mcpSelector:  mcpSelector,
		mcpExecutor:  mcpExecutor,
	}
}

// Plan 进行规划，返回下一步动作
func (p *Planner) Plan(ctx context.Context, goal string, trace *state.Trace) (state.Action, error) {
	// 首先尝试使用智能 MCP 工具选择
	if p.mcpSelector != nil && p.shouldUseMCPTools(goal, trace) {
		action, err := p.tryMCPToolSelection(ctx, goal, trace)
		if err == nil {
			logger.Infof("🤖 [MCP] Auto-selected tool via intelligent selection")
			return action, nil
		}
		logger.Debugf("🔄 [MCP] Selection failed, using standard planning: %v", err)
	}

	// 回退到标准规划流程
	return p.standardPlan(ctx, goal, trace)
}

// shouldUseMCPTools 判断是否应该优先使用 MCP 工具
func (p *Planner) shouldUseMCPTools(goal string, trace *state.Trace) bool {
	// 如果没有配置 MCP 服务，直接返回 false
	if p.mcpDiscovery == nil {
		logger.Get().Sugar().Debugw("No MCP discovery service available")
		return false
	}

	// 检查是否有可用的 MCP 工具
	allTools := p.mcpDiscovery.GetAllTools()
	logger.Get().Sugar().Debugw("MCP tools check", "available_tools", len(allTools))
	if len(allTools) == 0 {
		return false
	}

	// 🔧 重要修复：如果最近已经有成功的 MCP 工具调用，不要继续使用 MCP 工具
	// 让 LLM 分析现有结果并决定是否给出答案
	if trace != nil && len(trace.Steps) > 0 {
		// 检查最近3步中是否有成功的 MCP 调用
		recentSteps := 3
		if len(trace.Steps) < recentSteps {
			recentSteps = len(trace.Steps)
		}

		for i := len(trace.Steps) - recentSteps; i < len(trace.Steps); i++ {
			step := trace.Steps[i]
			// 如果有成功的 MCP 调用，让标准规划器处理结果
			if step.Action.Name == "mcp_call" && step.Observation != nil && step.Observation.ErrMsg == "" {
				logger.Debugf("📊 [PLAN] Analyzing recent MCP results")
				return false
			}
		}
	}

	goalLower := strings.ToLower(goal)

	// 一些关键词表明需要外部服务或实时数据
	externalKeywords := []string{
		"search", "query", "find", "get", "fetch", "retrieve", "lookup",
		"weather", "news", "stock", "price", "current", "latest", "real-time",
		"api", "service", "external", "online", "web", "internet",
		"database", "data", "information", "check", "verify", "validate",
		"查询", "搜索", "获取", "股价", "股票", "天气", "新闻", "数据", "信息",
	}

	for _, keyword := range externalKeywords {
		if strings.Contains(goalLower, keyword) {
			logger.Debugf("🔍 [MCP] Triggered by keyword: %s", keyword)
			return true
		}
	}

	// 检查最近是否有失败的内置工具调用
	if trace != nil && len(trace.Steps) > 0 {
		for i := len(trace.Steps) - 1; i >= 0 && i >= len(trace.Steps)-3; i-- {
			step := trace.Steps[i]
			if step.Observation != nil && step.Observation.ErrMsg != "" {
				// 如果最近的内置工具失败了，尝试 MCP 工具
				logger.Debugf("🔄 [MCP] Triggered by previous tool failure")
				return true
			}
		}
	}

	logger.Debugf("🚫 [MCP] Not triggered for: %s", goal)
	return false
}

// tryMCPToolSelection 尝试使用 MCP 工具选择
func (p *Planner) tryMCPToolSelection(ctx context.Context, goal string, trace *state.Trace) (state.Action, error) {
	// 使用智能选择器自动选择和调用 MCP 工具
	return p.mcpSelector.AutoSelectAndCall(ctx, goal, trace)
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

	// 调试信息
	logger.Get().Sugar().Debugf("LLM Response - ToolCalls: %d, Content: %q, FinishReason: %s",
		len(choice.Message.ToolCalls), choice.Message.Content, choice.FinishReason)

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

CRITICAL PRIORITY: If you have data from previous tool calls (especially MCP tool results), FIRST analyze whether this data is sufficient to answer the user's question. If it is sufficient, immediately use direct_answer to provide the answer based on the available data.

Guidelines:
1. **HIGHEST PRIORITY**: When you have data from MCP tools or other sources, analyze it first to see if it answers the user's question
2. If the data is sufficient, provide a direct_answer immediately - don't call more tools
3. Only call additional tools if the existing data is insufficient or incomplete
4. PRIORITIZE MCP tools for external data/services when new data is needed
5. Use the intelligent MCP system to automatically discover, select and call the most suitable MCP tools
6. Fall back to built-in tools only when MCP tools are not available or suitable
7. Always follow the tool registry strictly and return valid JSON arguments
8. Stop when the user goal is satisfied or no more useful action can be taken

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

	// 检查是否有成功的 MCP 数据
	hasSuccessfulMCPData := false
	var latestMCPData string

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
					// 检查是否是成功的 MCP 调用
					if step.Action.Name == "mcp_call" {
						hasSuccessfulMCPData = true
						// 对于 MCP 数据，显示完整内容而不是截断
						if rawOutput, ok := step.Observation.Output["content"]; ok {
							latestMCPData = fmt.Sprintf("%v", rawOutput)
						} else if jsonBytes, err := json.Marshal(step.Observation.Output); err == nil {
							latestMCPData = string(jsonBytes)
						}
					}

					// 截断长输出（但 MCP 数据会在下面专门处理）
					output := p.summarizeOutput(step.Observation.Output)
					context.WriteString(fmt.Sprintf("  Result: %s\n", output))
				}
			}
		}
		context.WriteString("\n")
	}

	// 如果有成功的 MCP 数据，添加特别的分析指导
	if hasSuccessfulMCPData {
		context.WriteString("🎯 IMPORTANT - DATA ANALYSIS PRIORITY:\n")
		context.WriteString("You have successfully obtained data from MCP tools. Your FIRST task is to analyze this data and determine if it's sufficient to answer the user's question.\n")
		context.WriteString("If the data answers the user's question, immediately provide a direct_answer based on this data.\n")
		context.WriteString("Only call additional tools if the existing data is insufficient.\n\n")

		if latestMCPData != "" {
			context.WriteString("LATEST MCP DATA TO ANALYZE:\n")
			context.WriteString(latestMCPData)
			context.WriteString("\n\n")
		}
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
