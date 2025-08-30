# OpenManus-Go 统一工具选择迁移指南

## 概述

本文档描述了将 openmanus-go 项目的工具选择逻辑从"智能优先级策略"迁移到"统一集合策略"的完整过程，使其与 OpenManus (Python) 项目的逻辑保持一致。

## 迁移目标

将原有的复杂工具选择逻辑：
```
智能条件判断 → MCP工具优先 → 内置工具回退 → 复杂路由
```

改为简化的统一选择逻辑：
```
统一工具集合 → LLM智能选择 → 透明执行
```

## 核心修改文件

### 1. `pkg/tool/schema.go`
**主要变更：**
- 新增工具类型系统 (`ToolType`, `ToolWithType`)
- 新增 `MCPTool` 实现
- 扩展 `ToolInfo` 包含类型信息

**关键代码：**
```go
type ToolType string
const (
    ToolTypeBuiltin ToolType = "builtin"
    ToolTypeMCP     ToolType = "mcp"
)

type ToolWithType interface {
    Tool
    Type() ToolType
    ServerName() string
}
```

### 2. `pkg/tool/registry.go`
**主要变更：**
- 新增 `RegisterMCPTools` 方法
- 新增 `UnregisterMCPTools` 方法
- 更新 `GetToolsManifest` 支持工具类型

**关键代码：**
```go
func (r *Registry) RegisterMCPTools(mcpTools []ToolInfo, executor MCPExecutor) error {
    // 将MCP工具注册到统一注册表
}
```

### 3. `pkg/agent/mcp_executor.go`
**主要变更：**
- 实现 `MCPExecutor` 接口
- 新增 `ExecuteMCPTool` 方法

**关键代码：**
```go
func (e *MCPExecutor) ExecuteMCPTool(ctx context.Context, serverName, toolName string, args map[string]any) (map[string]any, error) {
    // 统一的MCP工具执行接口
}
```

### 4. `pkg/agent/core.go`
**主要变更：**
- 重构 `NewBaseAgentWithMCP` 实现统一工具集成
- 简化 `Act` 方法移除路由逻辑

**关键变更：**
```go
// 旧逻辑：复杂的条件判断和组件创建
if appConfig != nil && len(appConfig.MCP.Servers) > 0 {
    mcpDiscovery := NewMCPDiscoveryService(appConfig)
    mcpSelector := NewMCPToolSelector(mcpDiscovery, llmClient)
    planner = NewPlannerWithMCP(...)
}

// 新逻辑：统一的工具集成
go func() {
    allTools := mcpDiscovery.GetAllTools()
    toolRegistry.RegisterMCPTools(mcpToolInfos, mcpExecutor)
}()
```

### 5. `pkg/agent/planner.go`
**主要变更：**
- 移除 `NewPlannerWithMCP` 和MCP特殊逻辑
- 简化 `Plan` 方法为统一规划
- 更新系统提示移除工具优先级概念

**关键变更：**
```go
// 旧逻辑：复杂的条件判断
func (p *Planner) Plan(ctx context.Context, goal string, trace *state.Trace) (state.Action, error) {
    if p.mcpSelector != nil && p.shouldUseMCPTools(goal, trace) {
        action, err := p.tryMCPToolSelection(ctx, goal, trace)
        if err == nil {
            return action, nil
        }
    }
    return p.standardPlan(ctx, goal, trace)
}

// 新逻辑：统一选择
func (p *Planner) Plan(ctx context.Context, goal string, trace *state.Trace) (state.Action, error) {
    return p.standardPlan(ctx, goal, trace)
}
```

## 架构对比

### 原架构（智能优先级）
```
用户请求
    ↓
条件判断 (shouldUseMCPTools)
    ↓
智能选择器 (MCPToolSelector)
    ↓
MCP执行器 (MCPExecutor)
    ↓
回退到标准规划器
    ↓
内置工具执行器
```

### 新架构（统一集合）
```
用户请求
    ↓
统一规划器 (Planner)
    ↓
LLM从统一工具集合选择
    ↓
统一工具执行器 (Executor)
    ↓
自动路由到正确实现
```

## 迁移步骤

### 步骤1：扩展工具系统
1. 定义工具类型枚举
2. 创建 `ToolWithType` 接口
3. 实现 `MCPTool` 包装器

### 步骤2：增强工具注册表
1. 添加MCP工具注册方法
2. 更新工具清单生成
3. 支持工具类型标识

### 步骤3：重构Agent核心
1. 修改Agent创建逻辑
2. 统一工具集成流程
3. 简化执行接口

### 步骤4：简化规划器
1. 移除MCP特殊逻辑
2. 更新系统提示
3. 统一工具选择流程

### 步骤5：测试验证
1. 创建测试示例
2. 验证工具选择行为
3. 确保向后兼容性

## 兼容性考虑

### 保持的功能
- ✅ 所有原有的内置工具
- ✅ MCP工具发现和执行
- ✅ 工具参数验证
- ✅ 错误处理和重试
- ✅ 执行统计和监控

### 移除的功能
- ❌ 复杂的工具优先级逻辑
- ❌ 条件性的MCP工具触发
- ❌ 显式的工具路由
- ❌ MCP特殊的系统提示

### 新增的功能
- ✨ 统一的工具类型系统
- ✨ 透明的工具执行
- ✨ 简化的配置和使用
- ✨ 更好的扩展性

## 性能影响

### 优化方面
- **减少条件判断**：移除复杂的if-else逻辑
- **简化调用栈**：统一的执行路径
- **降低内存使用**：减少冗余组件

### 权衡考虑
- **LLM调用**：可能增加工具选择的LLM调用
- **工具加载**：统一注册可能增加启动时间
- **内存占用**：所有工具都加载到内存

## 使用示例

### 原使用方式
```go
// 复杂的配置和创建
if len(appConfig.MCP.Servers) > 0 {
    agent = agent.NewBaseAgentWithMCP(...)
} else {
    agent = agent.NewBaseAgent(...)
}
```

### 新使用方式
```go
// 统一的创建方式
agent := agent.NewBaseAgentWithMCP(llmClient, toolRegistry, config, appConfig)
// 自动处理MCP工具集成，无需条件判断
```

## 迁移清单

- [x] 定义工具类型系统
- [x] 扩展工具注册表
- [x] 实现MCP工具包装器
- [x] 重构Agent核心逻辑
- [x] 简化规划器实现
- [x] 更新系统提示
- [x] 创建测试示例
- [x] 编写迁移文档

## 后续优化建议

1. **工具缓存**: 实现工具结果缓存以提高性能
2. **动态加载**: 支持工具的动态加载和卸载
3. **工具评分**: 基于执行历史对工具进行评分排序
4. **并行执行**: 支持多个工具的并行执行
5. **工具链**: 支持工具的组合和链式调用

## 总结

通过这次迁移，openmanus-go 项目现在采用了与 OpenManus (Python) 项目一致的"统一工具集合"策略，同时保持了Go语言的类型安全和性能优势。新的架构更简洁、更易维护，同时为未来的扩展提供了更好的基础。
