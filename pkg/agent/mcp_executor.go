package agent

import (
	"context"
	"fmt"
	"time"

	"openmanus-go/pkg/config"
	"openmanus-go/pkg/logger"
	"openmanus-go/pkg/mcp/transport"
	"openmanus-go/pkg/state"
)

// MCPExecutor 负责执行 MCP 工具调用
type MCPExecutor struct {
	config           *config.Config
	discoveryService *MCPDiscoveryService
	executionHistory map[string]*ExecutionStats
}

// ExecutionStats 记录工具执行统计信息
type ExecutionStats struct {
	ToolName       string        `json:"toolName"`
	ServerName     string        `json:"serverName"`
	TotalCalls     int           `json:"totalCalls"`
	SuccessCalls   int           `json:"successCalls"`
	FailedCalls    int           `json:"failedCalls"`
	AverageLatency time.Duration `json:"averageLatency"`
	LastExecution  time.Time     `json:"lastExecution"`
	LastError      string        `json:"lastError,omitempty"`
}

// NewMCPExecutor 创建新的 MCP 执行器
func NewMCPExecutor(cfg *config.Config, discoveryService *MCPDiscoveryService) *MCPExecutor {
	return &MCPExecutor{
		config:           cfg,
		discoveryService: discoveryService,
		executionHistory: make(map[string]*ExecutionStats),
	}
}

// ExecuteTool 执行 MCP 工具调用
func (e *MCPExecutor) ExecuteTool(ctx context.Context, action state.Action) (*state.Observation, error) {
	startTime := time.Now()

	// 解析动作参数
	serverName, toolName, toolArgs, err := e.parseActionArgs(action)
	if err != nil {
		return &state.Observation{
			Tool:    action.Name,
			ErrMsg:  fmt.Sprintf("Invalid action arguments: %v", err),
			Latency: time.Since(startTime).Milliseconds(),
		}, nil
	}

	// 获取服务器配置
	serverConfig, exists := e.config.MCP.Servers[serverName]
	if !exists {
		return &state.Observation{
			Tool:    action.Name,
			ErrMsg:  fmt.Sprintf("MCP server '%s' not found in configuration", serverName),
			Latency: time.Since(startTime).Milliseconds(),
		}, nil
	}

	// 验证工具存在
	_, toolExists := e.discoveryService.GetTool(toolName)
	if !toolExists {
		// 尝试使用服务器前缀查找
		prefixedToolName := fmt.Sprintf("%s.%s", serverName, toolName)
		if _, toolExists = e.discoveryService.GetTool(prefixedToolName); !toolExists {
			return &state.Observation{
				Tool:    action.Name,
				ErrMsg:  fmt.Sprintf("Tool '%s' not found on server '%s'", toolName, serverName),
				Latency: time.Since(startTime).Milliseconds(),
			}, nil
		}
		toolName = prefixedToolName
	}

	logger.Infof("🔌 [MCP] Calling %s on server %s", toolName, serverName)

	// 执行工具调用
	result, err := e.callMCPTool(ctx, serverName, serverConfig, toolName, toolArgs)
	latency := time.Since(startTime)

	// 更新执行统计
	e.updateExecutionStats(serverName, toolName, err == nil, latency, err)

	// 构建观测结果
	observation := &state.Observation{
		Tool:    action.Name,
		Latency: latency.Milliseconds(),
	}

	if err != nil {
		observation.ErrMsg = fmt.Sprintf("MCP tool execution failed: %v", err)
		logger.Warnf("❌ [MCP] %s failed: %v (%.0fms)", toolName, err, float64(observation.Latency))
	} else {
		// 直接返回原始结果，让 LLM 处理和决策
		observation.Output = result
		logger.Infof("✅ [MCP] %s completed (%.0fms)", toolName, float64(observation.Latency))
	}

	return observation, nil
}

// parseActionArgs 解析动作参数
func (e *MCPExecutor) parseActionArgs(action state.Action) (serverName, toolName string, toolArgs map[string]interface{}, err error) {
	// 解析服务器名称
	if server, ok := action.Args["server"].(string); ok {
		serverName = server
	} else {
		return "", "", nil, fmt.Errorf("missing or invalid 'server' parameter")
	}

	// 解析工具名称
	if name, ok := action.Args["name"].(string); ok {
		toolName = name
	} else {
		return "", "", nil, fmt.Errorf("missing or invalid 'name' parameter")
	}

	// 解析工具参数
	if args, ok := action.Args["args"].(map[string]interface{}); ok {
		toolArgs = args
	} else {
		toolArgs = make(map[string]interface{})
	}

	return serverName, toolName, toolArgs, nil
}

// callMCPTool 调用 MCP 工具
func (e *MCPExecutor) callMCPTool(ctx context.Context, serverName string, serverConfig config.MCPServerConfig, toolName string, args map[string]interface{}) (map[string]interface{}, error) {
	// 创建带超时的上下文
	callCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// 调用 MCP 服务器
	msg, err := transport.CallTool(callCtx, serverName, serverConfig, toolName, args, nil)
	if err != nil {
		return nil, fmt.Errorf("MCP transport error: %w", err)
	}

	// 检查响应是否为错误
	if msg.IsError() {
		return nil, fmt.Errorf("MCP server error: %s", msg.Error.Message)
	}

	// 解析结果
	result := make(map[string]interface{})
	if msg.Result != nil {
		if resultMap, ok := msg.Result.(map[string]interface{}); ok {
			// 处理标准 MCP 响应格式
			if content, hasContent := resultMap["content"]; hasContent {
				if contentArray, isArray := content.([]interface{}); isArray && len(contentArray) > 0 {
					// 提取第一个内容项的文本
					if contentItem, isMap := contentArray[0].(map[string]interface{}); isMap {
						if text, hasText := contentItem["text"].(string); hasText {
							result["result"] = text
						} else {
							result["content"] = contentArray
						}
					}
				} else {
					result["content"] = content
				}
			} else {
				// 直接使用整个结果
				result = resultMap
			}
		} else {
			// 非标准格式，直接包装
			result["result"] = msg.Result
		}
	}

	// 添加元数据
	result["_meta"] = map[string]interface{}{
		"server":    serverName,
		"tool":      toolName,
		"timestamp": time.Now().UTC(),
	}

	return result, nil
}

// updateExecutionStats 更新执行统计信息
func (e *MCPExecutor) updateExecutionStats(serverName, toolName string, success bool, latency time.Duration, err error) {
	statsKey := fmt.Sprintf("%s.%s", serverName, toolName)

	stats, exists := e.executionHistory[statsKey]
	if !exists {
		stats = &ExecutionStats{
			ToolName:   toolName,
			ServerName: serverName,
		}
		e.executionHistory[statsKey] = stats
	}

	stats.TotalCalls++
	stats.LastExecution = time.Now()

	if success {
		stats.SuccessCalls++
		stats.LastError = ""
	} else {
		stats.FailedCalls++
		if err != nil {
			stats.LastError = err.Error()
		}
	}

	// 更新平均延迟（简单移动平均）
	if stats.AverageLatency == 0 {
		stats.AverageLatency = latency
	} else {
		stats.AverageLatency = (stats.AverageLatency + latency) / 2
	}
}

// GetExecutionStats 获取执行统计信息
func (e *MCPExecutor) GetExecutionStats() map[string]*ExecutionStats {
	result := make(map[string]*ExecutionStats)
	for k, v := range e.executionHistory {
		result[k] = v
	}
	return result
}

// GetToolSuccessRate 获取工具成功率
func (e *MCPExecutor) GetToolSuccessRate(serverName, toolName string) float64 {
	statsKey := fmt.Sprintf("%s.%s", serverName, toolName)
	stats, exists := e.executionHistory[statsKey]
	if !exists || stats.TotalCalls == 0 {
		return 0.0
	}
	return float64(stats.SuccessCalls) / float64(stats.TotalCalls)
}

// ExecuteWithRetry 带重试的工具执行
func (e *MCPExecutor) ExecuteWithRetry(ctx context.Context, action state.Action, maxRetries int) (*state.Observation, error) {
	var lastObservation *state.Observation

	for attempt := 0; attempt <= maxRetries; attempt++ {
		observation, err := e.ExecuteTool(ctx, action)
		if err != nil {
			return observation, err
		}

		// 如果执行成功（没有错误消息），返回结果
		if observation.ErrMsg == "" {
			return observation, nil
		}

		lastObservation = observation

		// 如果不是最后一次尝试，等待后重试
		if attempt < maxRetries {
			logger.Get().Sugar().Infow("MCP tool execution failed, retrying",
				"attempt", attempt+1, "max_retries", maxRetries, "error", observation.ErrMsg)

			// 指数退避
			backoffDuration := time.Duration(1<<uint(attempt)) * time.Second
			select {
			case <-ctx.Done():
				return observation, ctx.Err()
			case <-time.After(backoffDuration):
			}
		}
	}

	return lastObservation, nil
}

// ValidateToolCall 验证工具调用参数
func (e *MCPExecutor) ValidateToolCall(action state.Action) error {
	serverName, toolName, toolArgs, err := e.parseActionArgs(action)
	if err != nil {
		return err
	}

	// 检查服务器配置
	if _, exists := e.config.MCP.Servers[serverName]; !exists {
		return fmt.Errorf("MCP server '%s' not configured", serverName)
	}

	// 检查工具是否存在
	toolInfo, exists := e.discoveryService.GetTool(toolName)
	if !exists {
		// 尝试使用服务器前缀
		prefixedToolName := fmt.Sprintf("%s.%s", serverName, toolName)
		if toolInfo, exists = e.discoveryService.GetTool(prefixedToolName); !exists {
			return fmt.Errorf("tool '%s' not found on server '%s'", toolName, serverName)
		}
	}

	// 验证参数（基础验证）
	if toolInfo.InputSchema != nil {
		if required, ok := toolInfo.InputSchema["required"].([]interface{}); ok {
			for _, reqField := range required {
				if fieldName, ok := reqField.(string); ok {
					if _, hasField := toolArgs[fieldName]; !hasField {
						return fmt.Errorf("required parameter '%s' missing for tool '%s'", fieldName, toolName)
					}
				}
			}
		}
	}

	return nil
}

// GetRecommendedTools 根据执行历史推荐工具
func (e *MCPExecutor) GetRecommendedTools(query string, maxResults int) []*MCPToolInfo {
	// 获取所有工具
	allTools := e.discoveryService.SearchTools(query, maxResults*2)
	if len(allTools) == 0 {
		return nil
	}

	// 根据成功率和使用频率排序
	type toolWithScore struct {
		tool  *MCPToolInfo
		score float64
	}

	toolScores := make([]toolWithScore, 0, len(allTools))
	for _, tool := range allTools {
		score := e.calculateToolRecommendationScore(tool)
		toolScores = append(toolScores, toolWithScore{tool: tool, score: score})
	}

	// 按分数排序
	for i := 0; i < len(toolScores)-1; i++ {
		for j := i + 1; j < len(toolScores); j++ {
			if toolScores[i].score < toolScores[j].score {
				toolScores[i], toolScores[j] = toolScores[j], toolScores[i]
			}
		}
	}

	// 返回前 maxResults 个结果
	if maxResults > 0 && len(toolScores) > maxResults {
		toolScores = toolScores[:maxResults]
	}

	result := make([]*MCPToolInfo, len(toolScores))
	for i, ts := range toolScores {
		result[i] = ts.tool
	}

	return result
}

// calculateToolRecommendationScore 计算工具推荐分数
func (e *MCPExecutor) calculateToolRecommendationScore(tool *MCPToolInfo) float64 {
	statsKey := fmt.Sprintf("%s.%s", tool.ServerName, tool.Name)
	stats, exists := e.executionHistory[statsKey]

	baseScore := 1.0 // 基础分数

	if exists {
		// 成功率影响 (0-3分)
		successRate := float64(stats.SuccessCalls) / float64(stats.TotalCalls)
		baseScore += successRate * 3.0

		// 使用频率影响 (0-2分)
		if stats.TotalCalls > 10 {
			baseScore += 2.0
		} else if stats.TotalCalls > 5 {
			baseScore += 1.0
		}

		// 最近使用影响 (0-1分)
		timeSinceLastUse := time.Since(stats.LastExecution)
		if timeSinceLastUse < 24*time.Hour {
			baseScore += 1.0
		} else if timeSinceLastUse < 7*24*time.Hour {
			baseScore += 0.5
		}

		// 平均延迟影响 (惩罚慢工具)
		if stats.AverageLatency > 10*time.Second {
			baseScore -= 1.0
		}
	}

	return baseScore
}

// ExecuteMCPTool 实现 tool.MCPExecutor 接口
func (e *MCPExecutor) ExecuteMCPTool(ctx context.Context, serverName, toolName string, args map[string]any) (map[string]any, error) {
	// 获取服务器配置
	serverConfig, exists := e.config.MCP.Servers[serverName]
	if !exists {
		return nil, fmt.Errorf("MCP server '%s' not found in configuration", serverName)
	}

	// 验证工具存在
	_, toolExists := e.discoveryService.GetTool(toolName)
	if !toolExists {
		// 尝试使用服务器前缀查找
		prefixedToolName := fmt.Sprintf("%s.%s", serverName, toolName)
		if _, toolExists = e.discoveryService.GetTool(prefixedToolName); !toolExists {
			return nil, fmt.Errorf("tool '%s' not found on server '%s'", toolName, serverName)
		}
		toolName = prefixedToolName
	}

	// 转换参数类型
	toolArgs := make(map[string]interface{})
	for k, v := range args {
		toolArgs[k] = v
	}

	// 执行工具调用
	result, err := e.callMCPTool(ctx, serverName, serverConfig, toolName, toolArgs)
	if err != nil {
		// 更新执行统计
		e.updateExecutionStats(serverName, toolName, false, 0, err)
		return nil, err
	}

	// 更新执行统计
	e.updateExecutionStats(serverName, toolName, true, 0, nil)

	return result, nil
}
