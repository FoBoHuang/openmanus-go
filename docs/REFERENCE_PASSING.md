# 引用传递和运行时查找机制 - 代码路径分析

## 完整的执行流程

### 1. 初始化阶段 (pkg/agent/core.go)

```go
func NewBaseAgentWithMCP(...) *BaseAgent {
    // 创建一个 Registry 实例
    toolRegistry := tool.NewRegistry()  // 假设这里传入了一个 Registry
    
    // 异步注册 MCP 工具
    go func() {
        // ... MCP 发现逻辑
        // 关键：向 **同一个** toolRegistry 实例添加工具
        toolRegistry.RegisterMCPTools(mcpToolInfos, mcpExecutor)
    }()
    
    // 创建 toolExecutor 和 planner，传入 **同一个** toolRegistry 引用
    toolExecutor := tool.NewExecutor(toolRegistry, 30*time.Second)  // 引用传递
    planner := NewPlanner(llmClient, toolRegistry)                  // 引用传递
    
    return &BaseAgent{
        toolExecutor: toolExecutor,
        planner:      planner,
        // ...
    }
}
```

### 2. 工具执行阶段 (用户运行命令时)

```go
// 用户执行: ./bin/openmanus run '查看今日麦格米特的股价'

// pkg/agent/core.go - Agent.Loop 被调用
func (a *BaseAgent) Loop(ctx context.Context, goal string) (string, error) {
    // ...
    action, err := a.planner.Plan(ctx, goal, trace)  // 规划器选择工具
    // ...
    obs, err := a.toolExecutor.Execute(ctx, action)  // 执行器执行工具
    // ...
}
```

### 3. 运行时查找阶段 (pkg/tool/exec.go)

```go
func (e *Executor) Execute(ctx context.Context, action state.Action) (*state.Observation, error) {
    // 关键：每次执行都实时调用 registry.Invoke
    // 这里的 e.registry 指向的是初始化时传入的那个 toolRegistry 实例
    result, err := e.registry.Invoke(execCtx, action.Name, action.Args)
    // ...
}
```

### 4. 实时工具查找 (pkg/tool/registry.go)

```go
func (r *Registry) Invoke(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
    // 实时查找工具
    tool, err := r.Get(name)  // 每次都重新查找
    // ...
}

func (r *Registry) Get(name string) (Tool, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    // 关键：从当前的 tools map 中查找
    // 如果异步 goroutine 已经注册了 MCP 工具，这里就能找到
    tool, exists := r.tools[name]
    if !exists {
        return nil, fmt.Errorf("tool '%s' not found", name)
    }
    return tool, nil
}
```

## 关键点说明

### 1. 引用传递的体现

```go
// 同一个 toolRegistry 实例的指针被传递给多个组件
var sharedRegistry *tool.Registry = toolRegistry

// 所有这些组件都持有指向同一个实例的引用：
toolExecutor.registry = sharedRegistry    // 指向同一个实例
planner.toolRegistry = sharedRegistry     // 指向同一个实例
asyncGoroutine.registry = sharedRegistry  // 异步 goroutine 也操作同一个实例
```

### 2. 运行时查找的体现

```go
// 不是这样（预缓存）：
type Executor struct {
    cachedTools map[string]Tool  // 如果是这样，异步添加的工具就看不到
}

// 而是这样（运行时查找）：
type Executor struct {
    registry *Registry  // 持有注册表引用，每次都实时查找
}

func (e *Executor) Execute(...) {
    // 每次执行都调用 registry.Get()，能看到最新注册的工具
    result, err := e.registry.Invoke(...)
}
```

### 3. 线程安全的保证

```go
type Registry struct {
    tools map[string]Tool
    mu    sync.RWMutex  // 读写锁保证并发安全
}

// 异步写入（注册工具）
func (r *Registry) RegisterMCPTools(...) {
    r.mu.Lock()         // 写锁
    r.tools[name] = tool
    r.mu.Unlock()
}

// 运行时读取（查找工具）
func (r *Registry) Get(name string) {
    r.mu.RLock()        // 读锁
    tool := r.tools[name]
    r.mu.RUnlock()
}
```

## 时序图

```
时间线: -------|-------|-------|-------|------>
        T1     T2     T3     T4     T5

T1: 创建 toolRegistry, toolExecutor, planner
T2: 启动异步 goroutine 注册 MCP 工具  
T3: 用户执行命令
T4: toolExecutor.Execute() 调用 registry.Invoke()
T5: registry.Get() 找到 T2 时注册的 MCP 工具

关键：T4 时刻的查找能看到 T2 时刻注册的工具，
     因为它们操作的是同一个 tools map
```

这就是为什么即使存在竞态条件，系统仍然能够正常工作的原因！
