package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"openmanus-go/pkg/config"
	"openmanus-go/pkg/logger"
	"openmanus-go/pkg/mcp/transport"
)

// MCPToolInfo 描述从 MCP Server 发现的工具信息
type MCPToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	ServerName  string                 `json:"serverName"`
	ServerURL   string                 `json:"serverUrl"`
	LastSeen    time.Time              `json:"lastSeen"`
}

// MCPDiscoveryService 负责从配置的 MCP 服务器发现和管理工具
type MCPDiscoveryService struct {
	mu              sync.RWMutex
	config          *config.Config
	discoveredTools map[string]*MCPToolInfo   // key: tool_name
	serverTools     map[string][]*MCPToolInfo // key: server_name
	lastUpdate      time.Time
	updateInterval  time.Duration
}

// NewMCPDiscoveryService 创建新的 MCP 发现服务
func NewMCPDiscoveryService(cfg *config.Config) *MCPDiscoveryService {
	return &MCPDiscoveryService{
		config:          cfg,
		discoveredTools: make(map[string]*MCPToolInfo),
		serverTools:     make(map[string][]*MCPToolInfo),
		updateInterval:  5 * time.Minute, // 每5分钟更新一次工具列表
	}
}

// Start 启动工具发现服务
func (s *MCPDiscoveryService) Start(ctx context.Context) error {
	logger.Get().Sugar().Info("Starting MCP tool discovery service")

	// 立即执行一次发现
	if err := s.discoverAllTools(ctx); err != nil {
		logger.Get().Sugar().Warnw("Initial tool discovery failed", "error", err)
		// 不要因为初始发现失败就退出，可能是网络问题
	}

	// 启动定期更新
	ticker := time.NewTicker(s.updateInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.Get().Sugar().Info("MCP discovery service stopped")
				return
			case <-ticker.C:
				if err := s.discoverAllTools(ctx); err != nil {
					logger.Get().Sugar().Warnw("Periodic tool discovery failed", "error", err)
				}
			}
		}
	}()

	return nil
}

// discoverAllTools 从所有配置的 MCP 服务器发现工具
func (s *MCPDiscoveryService) discoverAllTools(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	logger.Get().Sugar().Debugw("Discovering tools from MCP servers", "server_count", len(s.config.MCP.Servers))

	newDiscoveredTools := make(map[string]*MCPToolInfo)
	newServerTools := make(map[string][]*MCPToolInfo)

	for serverName, serverConfig := range s.config.MCP.Servers {
		tools, err := s.discoverToolsFromServer(ctx, serverName, serverConfig)
		if err != nil {
			logger.Get().Sugar().Warnw("Failed to discover tools from server",
				"server", serverName, "error", err)
			continue
		}

		newServerTools[serverName] = tools

		// 添加到全局工具列表
		for _, tool := range tools {
			// 如果工具名冲突，使用 server_name.tool_name 格式
			toolKey := tool.Name
			if existingTool, exists := newDiscoveredTools[toolKey]; exists {
				// 重命名冲突的工具
				toolKey = fmt.Sprintf("%s.%s", tool.ServerName, tool.Name)
				existingToolKey := fmt.Sprintf("%s.%s", existingTool.ServerName, existingTool.Name)

				// 重命名已存在的工具
				delete(newDiscoveredTools, tool.Name)
				newDiscoveredTools[existingToolKey] = existingTool
			}

			newDiscoveredTools[toolKey] = tool
		}
	}

	s.discoveredTools = newDiscoveredTools
	s.serverTools = newServerTools
	s.lastUpdate = time.Now()

	logger.Get().Sugar().Infow("Tool discovery completed",
		"total_tools", len(s.discoveredTools),
		"servers", len(s.serverTools))

	return nil
}

// discoverToolsFromServer 从单个 MCP 服务器发现工具
func (s *MCPDiscoveryService) discoverToolsFromServer(ctx context.Context, serverName string, serverConfig config.MCPServerConfig) ([]*MCPToolInfo, error) {
	// 创建带超时的上下文
	discoveryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 调用 MCP 服务器的 tools/list 方法
	msg, err := transport.ListTools(discoveryCtx, serverName, serverConfig, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools from server %s: %w", serverName, err)
	}

	if msg.IsError() {
		return nil, fmt.Errorf("MCP server %s returned error: %s", serverName, msg.Error.Message)
	}

	// 解析工具列表
	tools := make([]*MCPToolInfo, 0)
	if msg.Result != nil {
		if resultMap, ok := msg.Result.(map[string]interface{}); ok {
			if toolsArray, ok := resultMap["tools"].([]interface{}); ok {
				for _, toolItem := range toolsArray {
					if toolMap, ok := toolItem.(map[string]interface{}); ok {
						tool := &MCPToolInfo{
							ServerName: serverName,
							ServerURL:  serverConfig.URL,
							LastSeen:   time.Now(),
						}

						if name, ok := toolMap["name"].(string); ok {
							tool.Name = name
						}

						if desc, ok := toolMap["description"].(string); ok {
							tool.Description = desc
						}

						if schema, ok := toolMap["inputSchema"].(map[string]interface{}); ok {
							tool.InputSchema = schema
						}

						if tool.Name != "" {
							tools = append(tools, tool)
						}
					}
				}
			}
		}
	}

	logger.Get().Sugar().Debugw("Discovered tools from server",
		"server", serverName, "tool_count", len(tools))

	return tools, nil
}

// GetAllTools 获取所有发现的工具
func (s *MCPDiscoveryService) GetAllTools() map[string]*MCPToolInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*MCPToolInfo)
	for k, v := range s.discoveredTools {
		result[k] = v
	}
	return result
}

// GetToolsByServer 获取指定服务器的工具
func (s *MCPDiscoveryService) GetToolsByServer(serverName string) []*MCPToolInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if tools, exists := s.serverTools[serverName]; exists {
		result := make([]*MCPToolInfo, len(tools))
		copy(result, tools)
		return result
	}
	return nil
}

// SearchTools 根据查询搜索最匹配的工具
func (s *MCPDiscoveryService) SearchTools(query string, maxResults int) []*MCPToolInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.discoveredTools) == 0 {
		return nil
	}

	type toolScore struct {
		tool  *MCPToolInfo
		score float64
	}

	scores := make([]toolScore, 0, len(s.discoveredTools))
	queryLower := strings.ToLower(query)

	for _, tool := range s.discoveredTools {
		score := s.calculateToolScore(queryLower, tool)
		if score > 0 {
			scores = append(scores, toolScore{tool: tool, score: score})
		}
	}

	// 按分数排序
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// 返回前 maxResults 个结果
	if maxResults > 0 && len(scores) > maxResults {
		scores = scores[:maxResults]
	}

	result := make([]*MCPToolInfo, len(scores))
	for i, s := range scores {
		result[i] = s.tool
	}

	return result
}

// calculateToolScore 计算工具与查询的匹配分数
func (s *MCPDiscoveryService) calculateToolScore(queryLower string, tool *MCPToolInfo) float64 {
	score := 0.0

	toolNameLower := strings.ToLower(tool.Name)
	toolDescLower := strings.ToLower(tool.Description)

	// 精确匹配工具名 - 最高分
	if toolNameLower == queryLower {
		score += 10.0
	} else if strings.Contains(toolNameLower, queryLower) {
		score += 5.0
	}

	// 描述中包含查询词
	if strings.Contains(toolDescLower, queryLower) {
		score += 3.0
	}

	// 查询词的各个部分匹配
	queryWords := strings.Fields(queryLower)
	for _, word := range queryWords {
		if len(word) < 1 { // 降低最小长度要求
			continue
		}

		if strings.Contains(toolNameLower, word) {
			score += 2.0
		}
		if strings.Contains(toolDescLower, word) {
			score += 1.0
		}
	}

	// 特殊关键词映射 - 增强中文支持
	keywordMappings := map[string][]string{
		"股价":   {"stock", "price"},
		"股票":   {"stock"},
		"查询":   {"query", "search", "get"},
		"苹果":   {"apple", "aapl", "america"}, // 苹果公司是美股，加权重到美股工具
		"价格":   {"price"},
		"排行":   {"ranking", "rank"},
		"指数":   {"index"},
		"期货":   {"futures"},
		"外汇":   {"forex"},
		"美股":   {"america", "american"},
		"港股":   {"hongkong", "hong"},
		"A股":   {"china", "chinese"},
		"AAPL": {"america", "apple", "aapl"}, // AAPL 明确指向美股工具
		"aapl": {"america", "apple", "aapl"},
	}

	// 检查关键词映射
	for _, word := range queryWords {
		if mappings, exists := keywordMappings[word]; exists {
			for _, mapping := range mappings {
				if strings.Contains(toolNameLower, mapping) {
					score += 3.0
				}
				if strings.Contains(toolDescLower, mapping) {
					score += 2.0
				}
			}
		}
	}

	// 如果分数为0，给一个基础分数以便至少返回一些结果
	if score == 0 {
		// 检查是否有任何相关性
		relevantWords := []string{"stock", "price", "query", "search", "get", "america", "china"}
		for _, word := range relevantWords {
			if strings.Contains(toolNameLower, word) || strings.Contains(toolDescLower, word) {
				score += 0.5
				break
			}
		}
	}

	return score
}

// GetTool 根据工具名获取工具信息
func (s *MCPDiscoveryService) GetTool(toolName string) (*MCPToolInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tool, exists := s.discoveredTools[toolName]
	return tool, exists
}

// GetLastUpdateTime 获取最后更新时间
func (s *MCPDiscoveryService) GetLastUpdateTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastUpdate
}

// RefreshTools 强制刷新工具列表
func (s *MCPDiscoveryService) RefreshTools(ctx context.Context) error {
	return s.discoverAllTools(ctx)
}

// GetServerStatus 获取各服务器状态
func (s *MCPDiscoveryService) GetServerStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := make(map[string]interface{})
	for serverName, tools := range s.serverTools {
		status[serverName] = map[string]interface{}{
			"tool_count": len(tools),
			"last_seen":  s.lastUpdate,
			"tools":      tools,
		}
	}

	return status
}
