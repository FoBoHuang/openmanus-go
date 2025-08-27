# OpenManus-Go MCP 架构

## 概述

OpenManus-Go 的 MCP (Model Context Protocol) 架构采用了智能化、模块化的设计，实现了与外部 MCP 服务器的高效交互。

## 核心组件

### 1. MCP 发现服务 (`pkg/agent/mcp_discovery.go`)

**功能**: 自动发现和管理来自配置的 MCP 服务器的工具
- 启动时自动扫描所有配置的 MCP 服务器
- 缓存工具列表，提供快速查询
- 支持语义匹配和工具评分

**关键方法**:
- `Start()`: 启动发现服务
- `GetAllTools()`: 获取所有可用工具
- `SearchTools()`: 基于查询搜索匹配的工具

### 2. MCP 工具选择器 (`pkg/agent/mcp_selector.go`)

**功能**: 使用 LLM 智能选择最适合的 MCP 工具并生成参数
- LLM 驱动的工具选择
- 自动参数生成和验证
- 支持中文查询和语义理解

**关键方法**:
- `AutoSelectAndCall()`: 自动选择工具并生成调用参数
- `SelectTool()`: 基于意图选择最佳工具
- `generateParameters()`: 使用 LLM 生成工具参数

### 3. MCP 执行器 (`pkg/agent/mcp_executor.go`)

**功能**: 执行 MCP 工具调用，处理结果
- 解析动作参数
- 调用外部 MCP 服务
- 返回原始结果给 LLM 分析

**关键方法**:
- `ExecuteTool()`: 执行 MCP 工具调用
- 直接返回原始 JSON 数据，不进行预处理

### 4. MCP 传输层 (`pkg/mcp/transport/`)

**功能**: 处理与外部 MCP 服务器的底层通信
- HTTP POST JSON-RPC 调用
- SSE (Server-Sent Events) 连接管理
- 错误处理和重试机制

**组件**:
- `http_sender.go`: HTTP 请求发送
- `sse_client.go`: SSE 连接管理
- `api_client.go`: API 客户端封装

## 工作流程

```
1. 用户查询 → 2. 智能规划器 → 3. MCP 工具选择
     ↓                ↓              ↓
8. 最终答案 ← 7. LLM 分析 ← 6. 原始结果 ← 5. MCP 执行 ← 4. 参数生成
```

### 详细流程:

1. **用户输入**: 用户提出查询（如"今日苹果股价"）
2. **智能规划**: `Planner` 检测关键词，决定是否使用 MCP 工具
3. **工具发现**: `MCPDiscoveryService` 提供可用工具列表
4. **智能选择**: `MCPToolSelector` 使用 LLM 选择最佳工具
5. **参数生成**: LLM 自动生成工具调用参数
6. **执行调用**: `MCPExecutor` 调用外部 MCP 服务
7. **结果返回**: 返回原始 JSON 数据给 LLM
8. **智能分析**: LLM 分析数据并决定是否给出最终答案或继续执行

## 配置方式

在 `config.toml` 中配置 MCP 服务器:

```toml
[mcp]
[mcp.servers]
[mcp.servers.mcp-stock-helper]
url = "https://mcp.example.com/stock-helper"
timeout = 30
```

## 关键设计原则

### 1. 数据透明性
- **原始数据传递**: 不对 MCP 返回数据进行预处理或格式化
- **LLM 决策**: 让 LLM 分析原始数据并决定下一步行动
- **避免信息丢失**: 保持数据完整性，避免过度解析

### 2. 智能化
- **语义匹配**: 支持中文和英文语义理解
- **自动参数生成**: LLM 自动生成合适的工具参数
- **智能工具选择**: 基于意图和上下文选择最佳工具

### 3. 模块化
- **松耦合**: 各组件职责明确，相互独立
- **可扩展**: 易于添加新的 MCP 服务器和工具
- **可测试**: 每个组件都可以独立测试

### 4. 容错性
- **优雅降级**: MCP 失败时回退到内置工具
- **错误处理**: 完善的错误捕获和处理机制
- **重试机制**: 自动重试失败的请求

## 移除的组件

为了简化架构和避免复杂性，以下组件已被移除:

- ~~`pkg/mcp/server.go`~~: 本地 MCP 服务器（不需要）
- ~~`pkg/mcp/client.go`~~: 旧的 MCP 客户端（已被 transport 替代）
- ~~`pkg/tool/builtin/mcp.go`~~: 旧的 MCP 工具实现（功能重复）
- ~~`pkg/agent/mcp_result_parser.go`~~: 结果解析器（违反数据透明原则）
- ~~`pkg/flow/`~~: 流程引擎（过于复杂，不适合当前需求）

## 优势

1. **高效**: 智能工具选择减少了试错时间
2. **准确**: LLM 驱动的参数生成提高了调用成功率
3. **灵活**: 支持动态添加新的 MCP 服务器
4. **透明**: 原始数据传递保证了信息完整性
5. **智能**: LLM 自主决策何时停止和给出答案

这个架构成功实现了用户要求的"从 MCP 获取数据后，优先看返回的数据是否满足任务目标，满足了就直接回答用户即可"。
