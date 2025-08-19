package tool

import (
	"context"
	"fmt"
	"sync"
	"time"
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

// GetToolsManifest 获取工具清单（用于 LLM 提示）
func (r *Registry) GetToolsManifest() []ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	manifest := make([]ToolInfo, 0, len(r.tools))
	for _, tool := range r.tools {
		manifest = append(manifest, ToolInfo{
			Name:         tool.Name(),
			Description:  tool.Description(),
			InputSchema:  tool.InputSchema(),
			OutputSchema: tool.OutputSchema(),
		})
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
	result, err := tool.Invoke(ctx, args)
	latency := time.Since(start)

	if err != nil {
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

	return result, nil
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
