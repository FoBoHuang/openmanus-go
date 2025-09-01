# MCP 工具注册同步机制

## 问题描述

在 OpenManus-Go 中，MCP 工具的注册是异步执行的，这可能导致以下问题：

1. **竞态条件**：`toolExecutor` 和 `planner` 在创建时，MCP 工具可能还没有注册到 `toolRegistry` 中
2. **时序依赖**：当前使用固定的 2 秒等待时间，这可能不够稳定
3. **错误处理**：没有明确的机制来处理 MCP 工具注册失败的情况

## 当前工作原理

尽管存在竞态条件，系统仍然能够正常工作，原因如下：

### 1. 共享引用机制

```go
// toolExecutor 和 planner 都持有对同一个 toolRegistry 的引用
toolExecutor := tool.NewExecutor(toolRegistry, 30*time.Second)
planner := NewPlanner(llmClient, toolRegistry)

// 当异步 goroutine 向 toolRegistry 添加工具时，
// 所有持有该引用的组件都能看到新工具
```

### 2. 运行时动态查找

```go
// 每次工具执行时，都是实时从 registry.tools map 中查找
func (e *Executor) Execute(ctx context.Context, action state.Action) (*state.Observation, error) {
    // 实时调用 registry.Invoke，会查找最新注册的工具
    result, err := e.registry.Invoke(execCtx, action.Name, action.Args)
    // ...
}
```

### 3. 线程安全保证

```go
type Registry struct {
    tools map[string]Tool
    mu    sync.RWMutex  // 确保并发安全
}

func (r *Registry) Get(name string) (Tool, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    // 线程安全地获取工具
}
```

## 改进方案

为了解决潜在的竞态条件，我们添加了以下改进：

### 1. 工具状态检查方法

```go
// 检查是否有MCP工具注册
func (r *Registry) HasMCPTools() bool

// 获取MCP工具数量
func (r *Registry) GetMCPToolCount() int

// 等待MCP工具注册完成（带超时）
func (r *Registry) WaitForMCPTools(ctx context.Context, expectedCount int, timeout time.Duration) bool
```

### 2. 使用示例

```go
// 在创建 Agent 后，可以等待 MCP 工具准备就绪
agent := agent.NewBaseAgentWithMCP(llmClient, toolRegistry, agentConfig, cfg)

// 可选：等待 MCP 工具注册完成
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

expectedMCPTools := len(cfg.MCP.Servers) // 假设每个服务器至少有一个工具
if toolRegistry.WaitForMCPTools(ctx, expectedMCPTools, 5*time.Second) {
    logger.Info("MCP tools are ready")
} else {
    logger.Warn("MCP tools may not be fully ready, but execution can continue")
}

// 执行任务
result, err := agent.Loop(ctx, goal)
```

### 3. 错误处理改进

```go
// 在工具执行时提供更好的错误信息
func (e *Executor) Execute(ctx context.Context, action state.Action) (*state.Observation, error) {
    result, err := e.registry.Invoke(execCtx, action.Name, action.Args)
    if err != nil {
        // 检查是否是因为 MCP 工具未就绪
        if strings.Contains(err.Error(), "not found") {
            if !e.registry.HasMCPTools() {
                err = fmt.Errorf("tool '%s' not found (MCP tools may still be initializing): %w", action.Name, err)
            }
        }
    }
    // ...
}
```

## 最佳实践

1. **对于生产环境**：建议在开始执行任务前等待 MCP 工具准备就绪
2. **对于开发环境**：可以依赖当前的异步机制，因为通常有足够的时间让工具注册完成
3. **错误处理**：始终检查工具执行的错误，并提供有意义的错误信息

## 总结

- **当前实现能工作**：因为使用了共享引用和运行时查找
- **存在潜在问题**：竞态条件可能导致早期工具调用失败
- **改进方案**：添加了同步机制和更好的错误处理
- **建议**：在关键场景下使用等待机制确保工具准备就绪
