package tool

import (
	"context"
	"fmt"
	"time"

	"openmanus-go/pkg/logger"
	"openmanus-go/pkg/state"
)

// Executor 工具执行器
type Executor struct {
	registry *Registry
	timeout  time.Duration
}

// NewExecutor 创建工具执行器
func NewExecutor(registry *Registry, timeout time.Duration) *Executor {
	if registry == nil {
		registry = DefaultRegistry
	}
	if timeout == 0 {
		timeout = 30 * time.Second // 默认超时时间
	}

	return &Executor{
		registry: registry,
		timeout:  timeout,
	}
}

// Execute 执行工具调用并返回观测结果
func (e *Executor) Execute(ctx context.Context, action state.Action) (*state.Observation, error) {
	// 创建带超时的上下文
	execCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	start := time.Now()

	// 获取工具信息以显示类型
	toolInfo := e.getToolInfo(action.Name)
	toolTypeSymbol := "🔧" // 默认内置工具
	toolTypeText := "Built-in"
	if toolInfo != nil && toolInfo.Type == ToolTypeMCP {
		toolTypeSymbol = "🌐"
		toolTypeText = "MCP"
	}

	logger.Infof("🔧 [TOOL] Executing %s %s (%s tool)", toolTypeSymbol, action.Name, toolTypeText)
	if toolInfo != nil && toolInfo.ServerName != "" {
		logger.Infof("📡 [SERVER] Calling MCP server: %s", toolInfo.ServerName)
	}

	// 调用工具
	result, err := e.registry.Invoke(execCtx, action.Name, action.Args)
	latency := time.Since(start)

	// 构建观测结果
	observation := &state.Observation{
		Tool:    action.Name,
		Output:  result,
		Latency: latency.Milliseconds(),
	}

	if err != nil {
		observation.ErrMsg = err.Error()
		logger.Warnw("tool.exec.error", "tool", action.Name, "error", err, "latency_ms", observation.Latency)
		// 即使出错，也返回观测结果，让 Agent 能够处理错误
		return observation, nil
	}

	logger.Infow("tool.exec.ok", "tool", action.Name, "latency_ms", observation.Latency, "output_preview", previewResult(result))

	return observation, nil
}

// ExecuteWithRetry 带重试的工具执行
func (e *Executor) ExecuteWithRetry(ctx context.Context, action state.Action, maxRetries int, backoff time.Duration) (*state.Observation, error) {
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		obs, err := e.Execute(ctx, action)
		if err == nil && obs.ErrMsg == "" {
			return obs, nil
		}

		lastErr = err
		if err == nil && obs.ErrMsg != "" {
			lastErr = fmt.Errorf(obs.ErrMsg)
		}

		// 如果不是最后一次重试，等待一段时间
		if i < maxRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff * time.Duration(i+1)): // 指数退避
			}
		}
	}

	return nil, fmt.Errorf("tool execution failed after %d retries: %w", maxRetries, lastErr)
}

// BatchExecute 批量执行工具（并发）
func (e *Executor) BatchExecute(ctx context.Context, actions []state.Action) ([]*state.Observation, error) {
	results := make([]*state.Observation, len(actions))
	errors := make([]error, len(actions))

	// 使用 channel 来收集结果
	type result struct {
		index int
		obs   *state.Observation
		err   error
	}

	resultChan := make(chan result, len(actions))

	// 启动并发执行
	for i, action := range actions {
		go func(idx int, act state.Action) {
			obs, err := e.Execute(ctx, act)
			resultChan <- result{index: idx, obs: obs, err: err}
		}(i, action)
	}

	// 收集结果
	for i := 0; i < len(actions); i++ {
		res := <-resultChan
		results[res.index] = res.obs
		errors[res.index] = res.err
	}

	// 检查是否有错误
	var hasError bool
	for _, err := range errors {
		if err != nil {
			hasError = true
			break
		}
	}

	if hasError {
		return results, fmt.Errorf("some tool executions failed")
	}

	return results, nil
}

// ValidateAction 验证动作是否有效
func (e *Executor) ValidateAction(action state.Action) error {
	_, err := e.registry.Get(action.Name)
	if err != nil {
		return fmt.Errorf("tool not found: %w", err)
	}

	// 验证输入参数（如果工具支持的话）
	// 注意：这里需要具体的工具类型来进行验证

	return nil
}

// GetAvailableTools 获取可用工具列表
func (e *Executor) GetAvailableTools() []ToolInfo {
	return e.registry.GetToolsManifest()
}

// SetTimeout 设置执行超时时间
func (e *Executor) SetTimeout(timeout time.Duration) {
	e.timeout = timeout
}

// getToolInfo 获取工具信息
func (e *Executor) getToolInfo(toolName string) *ToolInfo {
	manifest := e.registry.GetToolsManifest()
	for _, toolInfo := range manifest {
		if toolInfo.Name == toolName {
			return &toolInfo
		}
	}
	return nil
}

func previewResult(m map[string]any) any {
	if m == nil {
		return nil
	}
	if r, ok := m["result"]; ok {
		if rs, ok := r.(string); ok {
			if len(rs) > 160 {
				return rs[:160] + "..."
			}
			return rs
		}
	}
	return m
}
