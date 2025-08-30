package tool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"openmanus-go/pkg/logger"
)

// Registry 工具注册表
type Registry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

// NewRegistry 创建新的工具注册表
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// DefaultRegistry 默认的全局工具注册表
var DefaultRegistry = NewRegistry()

// Register 注册工具
func (r *Registry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool '%s' already registered", name)
	}

	r.tools[name] = tool
	logger.Infow("tool.registry.register", "tool", name)
	return nil
}

// Unregister 取消注册工具
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("tool '%s' not found", name)
	}

	delete(r.tools, name)
	logger.Infow("tool.registry.unregister", "tool", name)
	return nil
}

// Get 获取工具
func (r *Registry) Get(name string) (Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}

	return tool, nil
}

// List 列出所有工具
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}

	return tools
}

// ListNames 列出所有工具名称
func (r *Registry) ListNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}

	return names
}

// GetToolsManifest 获取工具清单（用于 LLM 提示）
func (r *Registry) GetToolsManifest() []ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	manifest := make([]ToolInfo, 0, len(r.tools))
	for _, tool := range r.tools {
		toolInfo := ToolInfo{
			Name:         tool.Name(),
			Description:  tool.Description(),
			InputSchema:  tool.InputSchema(),
			OutputSchema: tool.OutputSchema(),
		}

		// 如果工具实现了 ToolWithType 接口，添加类型和服务器信息
		if toolWithType, ok := tool.(ToolWithType); ok {
			toolInfo.Type = toolWithType.Type()
			toolInfo.ServerName = toolWithType.ServerName()
		} else {
			toolInfo.Type = ToolTypeBuiltin
		}

		manifest = append(manifest, toolInfo)
	}

	return manifest
}

// Invoke 调用工具
func (r *Registry) Invoke(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	tool, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	// 验证输入参数（如果需要的话，由具体工具实现）

	// 执行工具并测量延迟
	start := time.Now()
	logger.Debugw("tool.invoke.start", "tool", name, "args", args)
	result, err := tool.Invoke(ctx, args)
	latency := time.Since(start)

	if err != nil {
		logger.Warnw("tool.invoke.error", "tool", name, "error", err, "latency_ms", latency.Milliseconds())
		return map[string]any{
			"error":      err.Error(),
			"latency_ms": latency.Milliseconds(),
		}, err
	}

	// 添加元数据
	if result == nil {
		result = make(map[string]any)
	}
	result["latency_ms"] = latency.Milliseconds()
	logger.Debugw("tool.invoke.ok", "tool", name, "latency_ms", latency.Milliseconds())

	return result, nil
}

// RegisterMCPTools 注册MCP工具
func (r *Registry) RegisterMCPTools(mcpTools []ToolInfo, executor MCPExecutor) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, toolInfo := range mcpTools {
		// 创建MCP工具实例
		mcpTool := NewMCPTool(
			toolInfo.Name,
			toolInfo.Description,
			toolInfo.ServerName,
			toolInfo.InputSchema,
			toolInfo.OutputSchema,
			executor,
		)

		// 如果工具已存在，先删除旧的
		if _, exists := r.tools[toolInfo.Name]; exists {
			logger.Infow("tool.registry.replace_mcp", "tool", toolInfo.Name, "server", toolInfo.ServerName)
			delete(r.tools, toolInfo.Name)
		}

		r.tools[toolInfo.Name] = mcpTool
		logger.Infow("tool.registry.register_mcp", "tool", toolInfo.Name, "server", toolInfo.ServerName)
	}

	return nil
}

// UnregisterMCPTools 取消注册指定服务器的MCP工具
func (r *Registry) UnregisterMCPTools(serverName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var removedTools []string
	for name, tool := range r.tools {
		if toolWithType, ok := tool.(ToolWithType); ok {
			if toolWithType.Type() == ToolTypeMCP && toolWithType.ServerName() == serverName {
				delete(r.tools, name)
				removedTools = append(removedTools, name)
			}
		}
	}

	if len(removedTools) > 0 {
		logger.Infow("tool.registry.unregister_mcp_tools", "server", serverName, "tools", removedTools)
	}

	return nil
}

// RegisterDefaults 注册默认工具
func (r *Registry) RegisterDefaults() error {
	// 这里会注册所有内置工具
	// 具体实现在各个工具包中
	return nil
}

// 全局函数，操作默认注册表

// Register 注册工具到默认注册表
func Register(tool Tool) error {
	return DefaultRegistry.Register(tool)
}

// Get 从默认注册表获取工具
func Get(name string) (Tool, error) {
	return DefaultRegistry.Get(name)
}

// List 列出默认注册表中的所有工具
func List() []Tool {
	return DefaultRegistry.List()
}

// GetToolsManifest 获取默认注册表的工具清单
func GetToolsManifest() []ToolInfo {
	return DefaultRegistry.GetToolsManifest()
}

// Invoke 调用默认注册表中的工具
func Invoke(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	return DefaultRegistry.Invoke(ctx, name, args)
}
