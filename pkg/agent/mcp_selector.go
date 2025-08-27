package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"openmanus-go/pkg/llm"
	"openmanus-go/pkg/logger"
	"openmanus-go/pkg/state"
)

// MCPToolSelector 负责智能选择和调用 MCP 工具
type MCPToolSelector struct {
	discoveryService *MCPDiscoveryService
	llmClient        llm.Client
}

// NewMCPToolSelector 创建新的 MCP 工具选择器
func NewMCPToolSelector(discoveryService *MCPDiscoveryService, llmClient llm.Client) *MCPToolSelector {
	return &MCPToolSelector{
		discoveryService: discoveryService,
		llmClient:        llmClient,
	}
}

// SelectTool 根据用户请求智能选择最合适的 MCP 工具
func (s *MCPToolSelector) SelectTool(ctx context.Context, userRequest string, context *state.Trace) (*MCPToolInfo, map[string]interface{}, error) {
	// 1. 使用基础搜索获取候选工具
	candidates := s.discoveryService.SearchTools(userRequest, 5)
	if len(candidates) == 0 {
		return nil, nil, fmt.Errorf("no suitable MCP tools found for request: %s", userRequest)
	}

	logger.Get().Sugar().Debugw("Found MCP tool candidates",
		"request", userRequest, "candidates", len(candidates))

	// 2. 如果只有一个候选工具，直接使用
	if len(candidates) == 1 {
		selectedTool := candidates[0]
		parameters, err := s.generateParameters(ctx, userRequest, selectedTool)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate parameters for tool %s: %w", selectedTool.Name, err)
		}
		return selectedTool, parameters, nil
	}

	// 3. 使用 LLM 进行智能选择
	selectedTool, err := s.llmSelectTool(ctx, userRequest, candidates, context)
	if err != nil {
		// 如果 LLM 选择失败，回退到第一个候选工具
		logger.Get().Sugar().Warnw("LLM tool selection failed, falling back to first candidate", "error", err)
		selectedTool = candidates[0]
	}

	// 4. 生成工具参数
	parameters, err := s.generateParameters(ctx, userRequest, selectedTool)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate parameters for tool %s: %w", selectedTool.Name, err)
	}

	logger.Get().Sugar().Infow("Selected MCP tool",
		"tool", selectedTool.Name, "server", selectedTool.ServerName)

	return selectedTool, parameters, nil
}

// llmSelectTool 使用 LLM 从候选工具中选择最合适的工具
func (s *MCPToolSelector) llmSelectTool(ctx context.Context, userRequest string, candidates []*MCPToolInfo, context *state.Trace) (*MCPToolInfo, error) {
	// 构建工具选择提示
	systemPrompt := s.buildToolSelectionPrompt()
	userPrompt := s.buildUserSelectionPrompt(userRequest, candidates, context)

	messages := []llm.Message{
		llm.CreateSystemMessage(systemPrompt),
		llm.CreateUserMessage(userPrompt),
	}

	// 创建工具选择函数
	selectionTool := llm.Tool{
		Type: "function",
		Function: llm.ToolFunction{
			Name:        "select_mcp_tool",
			Description: "Select the most appropriate MCP tool for the user request",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selected_tool": map[string]interface{}{
						"type":        "string",
						"description": "The name of the selected tool",
					},
					"reason": map[string]interface{}{
						"type":        "string",
						"description": "Reason for selecting this tool",
					},
				},
				"required": []string{"selected_tool", "reason"},
			},
		},
	}

	req := &llm.ChatRequest{
		Messages:    messages,
		Tools:       []llm.Tool{selectionTool},
		ToolChoice:  "auto",
		Temperature: 0.1,
	}

	resp, err := s.llmClient.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM tool selection failed: %w", err)
	}

	if len(resp.Choices) == 0 || len(resp.Choices[0].Message.ToolCalls) == 0 {
		return nil, fmt.Errorf("no tool selection response from LLM")
	}

	// 解析选择结果
	toolCall := resp.Choices[0].Message.ToolCalls[0]
	args, err := llm.ParseToolCallArguments(toolCall.Function.Arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tool selection arguments: %w", err)
	}

	selectedToolName, ok := args["selected_tool"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid tool selection response")
	}

	reason, _ := args["reason"].(string)

	// 查找选中的工具
	for _, candidate := range candidates {
		if candidate.Name == selectedToolName {
			logger.Get().Sugar().Infow("LLM selected MCP tool",
				"tool", selectedToolName, "reason", reason)
			return candidate, nil
		}
	}

	return nil, fmt.Errorf("selected tool %s not found in candidates", selectedToolName)
}

// generateParameters 为选定的工具生成参数
func (s *MCPToolSelector) generateParameters(ctx context.Context, userRequest string, tool *MCPToolInfo) (map[string]interface{}, error) {
	if tool.InputSchema == nil {
		return map[string]interface{}{}, nil
	}

	// 构建参数生成提示
	systemPrompt := s.buildParameterGenerationPrompt()
	userPrompt := s.buildUserParameterPrompt(userRequest, tool)

	messages := []llm.Message{
		llm.CreateSystemMessage(systemPrompt),
		llm.CreateUserMessage(userPrompt),
	}

	// 创建参数生成函数
	parameterTool := llm.Tool{
		Type: "function",
		Function: llm.ToolFunction{
			Name:        "generate_tool_parameters",
			Description: "Generate parameters for the selected MCP tool based on user request",
			Parameters:  tool.InputSchema,
		},
	}

	req := &llm.ChatRequest{
		Messages:    messages,
		Tools:       []llm.Tool{parameterTool},
		ToolChoice:  "auto",
		Temperature: 0.1,
	}

	resp, err := s.llmClient.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("parameter generation failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	choice := resp.Choices[0]

	// 优先处理工具调用
	if len(choice.Message.ToolCalls) > 0 {
		toolCall := choice.Message.ToolCalls[0]
		parameters, err := llm.ParseToolCallArguments(toolCall.Function.Arguments)
		if err != nil {
			return nil, fmt.Errorf("failed to parse generated parameters: %w", err)
		}
		logger.Get().Sugar().Debugw("Generated parameters via tool call",
			"tool", tool.Name, "parameters", parameters)
		return parameters, nil
	}

	// 如果没有工具调用，尝试解析文本响应为 JSON
	if choice.Message.Content != "" {
		var parameters map[string]interface{}
		if err := json.Unmarshal([]byte(choice.Message.Content), &parameters); err == nil {
			logger.Get().Sugar().Debugw("Generated parameters via text response",
				"tool", tool.Name, "parameters", parameters)
			return parameters, nil
		}
	}

	// 如果都失败了，尝试生成简单的默认参数
	parameters := s.generateDefaultParameters(userRequest, tool)

	logger.Get().Sugar().Debugw("Generated parameters for MCP tool",
		"tool", tool.Name, "parameters", parameters)

	return parameters, nil
}

// buildToolSelectionPrompt 构建工具选择的系统提示
func (s *MCPToolSelector) buildToolSelectionPrompt() string {
	return `You are an intelligent tool selector for MCP (Model Context Protocol) tools. Your task is to analyze the user's request and select the most appropriate tool from the available candidates.

Consider the following factors when selecting a tool:
1. Functional relevance: How well does the tool's purpose match the user's request?
2. Parameter compatibility: Can the required parameters be extracted from the user's request?
3. Context appropriateness: Is this tool suitable for the current context?
4. Reliability: Prefer tools that are more likely to succeed

Always select exactly one tool and provide a clear reason for your choice.`
}

// buildUserSelectionPrompt 构建用户工具选择提示
func (s *MCPToolSelector) buildUserSelectionPrompt(userRequest string, candidates []*MCPToolInfo, context *state.Trace) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("User Request: %s\n\n", userRequest))

	// 添加上下文信息
	if context != nil && len(context.Steps) > 0 {
		prompt.WriteString("Context from previous steps:\n")
		for i, step := range context.Steps {
			if i >= 3 { // 只显示最近3步
				break
			}
			prompt.WriteString(fmt.Sprintf("- Step %d: %s\n", len(context.Steps)-i, step.Action.Name))
			if step.Observation != nil && step.Observation.ErrMsg != "" {
				prompt.WriteString(fmt.Sprintf("  Error: %s\n", step.Observation.ErrMsg))
			}
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString("Available MCP Tools:\n")
	for i, tool := range candidates {
		prompt.WriteString(fmt.Sprintf("%d. Tool: %s\n", i+1, tool.Name))
		prompt.WriteString(fmt.Sprintf("   Server: %s\n", tool.ServerName))
		prompt.WriteString(fmt.Sprintf("   Description: %s\n", tool.Description))

		// 显示参数信息
		if tool.InputSchema != nil {
			if props, ok := tool.InputSchema["properties"].(map[string]interface{}); ok {
				prompt.WriteString("   Parameters: ")
				paramNames := make([]string, 0, len(props))
				for paramName := range props {
					paramNames = append(paramNames, paramName)
				}
				prompt.WriteString(strings.Join(paramNames, ", "))
				prompt.WriteString("\n")
			}
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString("Please select the most appropriate tool for this request.")

	return prompt.String()
}

// buildParameterGenerationPrompt 构建参数生成的系统提示
func (s *MCPToolSelector) buildParameterGenerationPrompt() string {
	return `You are a parameter generator for MCP tools. Your task is to extract and generate appropriate parameters from the user's request based on the tool's input schema.

Guidelines:
1. Extract parameter values directly from the user's request when possible
2. Use reasonable defaults for optional parameters
3. Ensure all required parameters are provided
4. Convert data types appropriately (strings, numbers, booleans, arrays, objects)
5. If a required parameter cannot be determined from the request, use a sensible default or ask for clarification

Generate parameters that will allow the tool to fulfill the user's request effectively.`
}

// buildUserParameterPrompt 构建用户参数生成提示
func (s *MCPToolSelector) buildUserParameterPrompt(userRequest string, tool *MCPToolInfo) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("User Request: %s\n\n", userRequest))
	prompt.WriteString(fmt.Sprintf("Selected Tool: %s\n", tool.Name))
	prompt.WriteString(fmt.Sprintf("Tool Description: %s\n\n", tool.Description))

	// 显示参数模式
	if tool.InputSchema != nil {
		schemaBytes, _ := json.MarshalIndent(tool.InputSchema, "", "  ")
		prompt.WriteString(fmt.Sprintf("Tool Input Schema:\n%s\n\n", string(schemaBytes)))
	}

	prompt.WriteString("Please generate appropriate parameters for this tool based on the user request and the input schema.")

	return prompt.String()
}

// generateDefaultParameters 生成默认参数（当 LLM 参数生成失败时的回退方案）
func (s *MCPToolSelector) generateDefaultParameters(userRequest string, tool *MCPToolInfo) map[string]interface{} {
	parameters := make(map[string]interface{})

	if tool.InputSchema == nil {
		return parameters
	}

	// 获取参数定义
	properties, ok := tool.InputSchema["properties"].(map[string]interface{})
	if !ok {
		return parameters
	}

	required, _ := tool.InputSchema["required"].([]interface{})
	requiredFields := make(map[string]bool)
	for _, req := range required {
		if reqStr, ok := req.(string); ok {
			requiredFields[reqStr] = true
		}
	}

	// 简单的参数推断
	requestLower := strings.ToLower(userRequest)

	for paramName, paramInfo := range properties {
		paramMap, ok := paramInfo.(map[string]interface{})
		if !ok {
			continue
		}

		paramType, _ := paramMap["type"].(string)
		description, _ := paramMap["description"].(string)
		descLower := strings.ToLower(description)

		// 根据参数名和描述推断值
		switch {
		case strings.Contains(paramName, "symbol") || strings.Contains(descLower, "symbol") || strings.Contains(descLower, "股票代码"):
			// 尝试提取股票代码
			if symbol := s.extractStockSymbol(requestLower); symbol != "" {
				parameters[paramName] = symbol
			} else if requiredFields[paramName] {
				parameters[paramName] = "AAPL" // 默认苹果股票
			}

		case strings.Contains(paramName, "market") || strings.Contains(descLower, "market"):
			// 市场参数
			if strings.Contains(requestLower, "美股") || strings.Contains(requestLower, "苹果") {
				parameters[paramName] = "US"
			} else if strings.Contains(requestLower, "港股") {
				parameters[paramName] = "HK"
			} else if strings.Contains(requestLower, "a股") {
				parameters[paramName] = "CN"
			} else if requiredFields[paramName] {
				parameters[paramName] = "US"
			}

		case strings.Contains(paramName, "code") && paramType == "string":
			// 代码参数
			if code := s.extractStockSymbol(requestLower); code != "" {
				parameters[paramName] = code
			} else if requiredFields[paramName] {
				parameters[paramName] = "AAPL"
			}

		case paramType == "string" && requiredFields[paramName]:
			// 必需的字符串参数，给个默认值
			parameters[paramName] = "AAPL"

		case paramType == "number" && requiredFields[paramName]:
			// 必需的数字参数
			parameters[paramName] = 1

		case paramType == "boolean" && requiredFields[paramName]:
			// 必需的布尔参数
			parameters[paramName] = true
		}
	}

	return parameters
}

// extractStockSymbol 从用户请求中提取股票代码
func (s *MCPToolSelector) extractStockSymbol(requestLower string) string {
	// 常见股票代码映射
	stockMappings := map[string]string{
		"苹果":        "AAPL",
		"apple":     "AAPL",
		"微软":        "MSFT",
		"microsoft": "MSFT",
		"谷歌":        "GOOGL",
		"google":    "GOOGL",
		"特斯拉":       "TSLA",
		"tesla":     "TSLA",
		"亚马逊":       "AMZN",
		"amazon":    "AMZN",
	}

	// 检查映射
	for keyword, symbol := range stockMappings {
		if strings.Contains(requestLower, keyword) {
			return symbol
		}
	}

	// 尝试提取已存在的股票代码格式
	re := regexp.MustCompile(`[A-Z]{1,5}`)
	matches := re.FindAllString(strings.ToUpper(requestLower), -1)
	if len(matches) > 0 {
		return matches[0]
	}

	return ""
}

// AutoSelectAndCall 自动选择并准备调用 MCP 工具
func (s *MCPToolSelector) AutoSelectAndCall(ctx context.Context, userRequest string, context *state.Trace) (state.Action, error) {
	// 选择工具和生成参数
	selectedTool, parameters, err := s.SelectTool(ctx, userRequest, context)
	if err != nil {
		return state.Action{}, err
	}

	// 创建 MCP 调用动作
	action := state.Action{
		Name: "mcp_call",
		Args: map[string]interface{}{
			"server": selectedTool.ServerName,
			"name":   selectedTool.Name,
			"args":   parameters,
		},
		Reason: fmt.Sprintf("Selected MCP tool %s from server %s to handle: %s",
			selectedTool.Name, selectedTool.ServerName, userRequest),
	}

	return action, nil
}
