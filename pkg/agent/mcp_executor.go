package agent

import (
	"context"
	"fmt"
	"time"

	"openmanus-go/pkg/config"
	"openmanus-go/pkg/mcp/transport"
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
