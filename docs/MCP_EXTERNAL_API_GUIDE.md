# 外部 API 通过 MCP 接入 Agent 指南

本文档详细说明如何将外部 API（如 APP 推荐服务）通过 MCP 协议封装为工具，供 OpenManus-Go Agent 使用。

## 目录

- [整体架构](#整体架构)
- [通信流程详解](#通信流程详解)
  - [阶段一：启动时工具发现](#阶段一启动时工具发现)
  - [阶段二：LLM 选择工具](#阶段二llm-选择工具)
  - [阶段三：执行工具调用](#阶段三执行工具调用)
  - [阶段四：结果回传](#阶段四结果回传)
- [MCP 通信协议](#mcp-通信协议)
- [实践指南：编写 MCP Server 适配器](#实践指南编写-mcp-server-适配器)
  - [MCP Server 代码结构](#mcp-server-代码结构)
  - [核心实现要点](#核心实现要点)
  - [配置与启动](#配置与启动)
- [关键源码索引](#关键源码索引)

---

## 整体架构

Agent 并不直接调用外部 API，而是通过一个 **MCP Server 适配层** 进行协议转换。整体架构如下：

```
┌──────────────────────────────────────────────────────────────────────┐
│  openmanus-go Agent 进程                                              │
│                                                                      │
│  ┌──────────┐    ┌──────────┐    ┌───────────────┐    ┌───────────┐ │
│  │  Planner  │───▶│   LLM    │───▶│ Tool Executor │───▶│  MCPTool  │ │
│  │  (规划器)  │◀──│ (大模型)  │    │  (工具执行器)   │    │  .Invoke  │ │
│  └──────────┘    └──────────┘    └───────────────┘    └─────┬─────┘ │
│                                                             │       │
│  ┌───────────────────┐    ┌───────────────────────┐         │       │
│  │ MCPDiscovery      │    │ MCPExecutor           │◀────────┘       │
│  │ (工具发现服务)      │───▶│ (MCP工具执行器)        │                  │
│  └─────────┬─────────┘    └───────────┬───────────┘                  │
│            │                          │                              │
└────────────┼──────────────────────────┼──────────────────────────────┘
             │ tools/list               │ tools/call
             │ (HTTP POST /message)     │ (HTTP POST /message)
             ▼                          ▼
┌──────────────────────────────────────────────────────┐
│  MCP Server 适配器 (你编写的独立服务)                     │
│  监听端口 /message                                    │
│  JSON-RPC 2.0 协议                                   │
└───────────────────────────┬──────────────────────────┘
                            │ HTTP 请求
                            ▼
┌──────────────────────────────────────────────────────┐
│  外部 API 服务（如 APP 推荐服务）                        │
│  /api_server/xxx                                     │
└──────────────────────────────────────────────────────┘
```

核心思路：MCP Server 是一个**纯粹的协议翻译层**——把 Agent 发来的 JSON-RPC 请求翻译成对实际 API 的 HTTP 调用，再把结果翻译回 JSON-RPC 响应。

---

## 通信流程详解

### 阶段一：启动时工具发现

Agent 启动时，`NewBaseAgentWithMCP` 函数（`pkg/agent/core.go`）完成以下工作：

1. 读取配置中的 MCP Server 列表（`config.MCP.Servers`）
2. 创建 `MCPDiscoveryService` 和 `MCPExecutor`
3. 启动工具发现流程

```go
// pkg/agent/core.go — NewBaseAgentWithMCP
if appConfig != nil && len(appConfig.MCP.Servers) > 0 {
    mcpDiscovery := NewMCPDiscoveryService(appConfig)
    mcpExecutor = NewMCPExecutor(appConfig, mcpDiscovery)

    go func() {
        mcpDiscovery.Start(ctx)
        time.Sleep(2 * time.Second)
        allTools := mcpDiscovery.GetAllTools()
        // 将 MCP 工具注册到统一工具注册表
        toolRegistry.RegisterMCPTools(mcpToolInfos, mcpExecutor)
    }()
}
```

`MCPDiscoveryService`（`pkg/agent/mcp_discovery.go`）向每个配置的 MCP Server 发送 `tools/list` 请求：

```go
// pkg/agent/mcp_discovery.go — discoverToolsFromServer
msg, err := transport.ListTools(discoveryCtx, serverName, serverConfig, nil)
```

底层通过 `transport.ListTools`（`pkg/mcp/transport/api_client.go`）发送 HTTP POST：

```go
// pkg/mcp/transport/api_client.go — ListTools
payload := jsonrpcRequest{JSONRPC: "2.0", ID: reqID, Method: "tools/list", Params: map[string]any{}}
msgURL := DeriveMessageURL(cfg.URL)  // → http://localhost:9100/message
// HTTP POST 发送 JSON-RPC 请求
```

**MCP Server 收到请求后**，返回工具定义（name、description、inputSchema），Agent 将其注册到统一工具注册表。此后 LLM 就能"看到"这些工具。

此外，`MCPDiscoveryService` 每 **5 分钟**自动刷新一次工具列表，确保工具定义保持最新。

### 阶段二：LLM 选择工具

当用户给 Agent 一个目标时，Planner 将**所有可用工具**（内置 + MCP）转成 LLM function-calling 格式，发送给大模型：

```
Agent.Plan(goal)
  → Planner.buildLLMTools()     // 从 Registry 获取所有工具（包括 MCP 工具）
  → LLM.Chat(messages, tools)   // 把工具列表提交给 LLM
  → LLM 返回: "调用 find_person_for_male_user，参数 {user_id: '123'}"
```

LLM 看到的工具描述来自 MCP Server 中定义的 `name`、`description`、`inputSchema`，它会根据语义判断何时调用该工具。

### 阶段三：执行工具调用

LLM 决定调用某个 MCP 工具后，执行链路如下：

#### 步骤 1：BaseAgent.Act → Executor.Execute

```go
// pkg/agent/core.go
func (a *BaseAgent) Act(ctx context.Context, action state.Action) (*state.Observation, error) {
    return a.toolExecutor.Execute(ctx, action)
}
```

#### 步骤 2：Executor 从 Registry 中取出工具并调用 Invoke

```go
// pkg/tool/exec.go — Execute
result, err := e.registry.Invoke(execCtx, action.Name, action.Args)
```

#### 步骤 3：MCPTool.Invoke 委托给 MCPExecutor

由于工具类型是 `ToolTypeMCP`，实际由 `MCPTool.Invoke` 处理：

```go
// pkg/tool/schema.go
func (mt *MCPTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
    return mt.executor.ExecuteMCPTool(ctx, mt.serverName, mt.name, args)
}
```

#### 步骤 4：MCPExecutor 通过 transport.CallTool 发送 JSON-RPC

```go
// pkg/agent/mcp_executor.go — ExecuteMCPTool
serverConfig, exists := e.config.MCP.Servers[serverName]
result, err := e.callMCPTool(ctx, serverName, serverConfig, toolName, toolArgs)

// callMCPTool 内部调用：
msg, err := transport.CallTool(callCtx, serverName, serverConfig, toolName, args, nil)
```

`transport.CallTool`（`pkg/mcp/transport/api_client.go`）构造 JSON-RPC 请求并 POST 到 MCP Server 的 `/message` 端点：

```go
// pkg/mcp/transport/api_client.go — CallTool
payload := jsonrpcRequest{
    JSONRPC: "2.0", ID: reqID,
    Method: "tools/call",
    Params: toolsCallParams{Name: toolName, Arguments: args},
}
msgURL := DeriveMessageURL(cfg.URL)  // → http://localhost:9100/message
```

#### 步骤 5：MCP Server 转发请求到实际 API

MCP Server 收到 `tools/call` 请求后，将参数转发到实际的外部 API（如 `/api_server/xxx`），并将结果包装成 MCP 标准响应格式返回。

### 阶段四：结果回传

结果沿调用链逐层返回：

```
外部 API 返回 JSON 响应
  → MCP Server 包装为 {"content": [{"type": "text", "text": "..."}]}
    → MCPExecutor 解析出 result["result"] = "..."，并附加 _meta 元数据
      → Executor 封装为 Observation{Output: result, Latency: ...}
        → Agent 拿到结果，作为后续规划和反思的上下文
```

`MCPExecutor` 解析响应的逻辑（`pkg/agent/mcp_executor.go`）：

```go
// 标准 MCP 响应: {"content": [{"type": "text", "text": "..."}]}
if contentArray, isArray := content.([]interface{}); isArray && len(contentArray) > 0 {
    if contentItem, isMap := contentArray[0].(map[string]interface{}); isMap {
        if text, hasText := contentItem["text"].(string); hasText {
            result["result"] = text
        }
    }
}
result["_meta"] = map[string]interface{}{
    "server": serverName, "tool": toolName, "timestamp": time.Now().UTC(),
}
```

---

## MCP 通信协议

Agent 和 MCP Server 之间使用 **JSON-RPC 2.0 over HTTP** 协议，所有请求均为 **HTTP POST** 到 `{MCP Server URL}/message` 端点。

核心只有两个交互方法：

### tools/list — 工具发现

**请求：**

```json
{
  "jsonrpc": "2.0",
  "id": "mcp-recommend-server-tools-list-1234567890",
  "method": "tools/list",
  "params": {}
}
```

**响应：**

```json
{
  "jsonrpc": "2.0",
  "id": "mcp-recommend-server-tools-list-1234567890",
  "result": {
    "tools": [
      {
        "name": "find_person_for_male_user",
        "description": "为男性用户推荐匹配的人选",
        "inputSchema": {
          "type": "object",
          "properties": {
            "user_id": {
              "type": "string",
              "description": "用户ID"
            },
            "limit": {
              "type": "integer",
              "description": "返回推荐人数上限"
            }
          },
          "required": ["user_id"]
        }
      }
    ]
  }
}
```

### tools/call — 工具调用

**请求：**

```json
{
  "jsonrpc": "2.0",
  "id": "mcp-recommend-server-tools-call-1234567890",
  "method": "tools/call",
  "params": {
    "name": "find_person_for_male_user",
    "arguments": {
      "user_id": "12345",
      "limit": 10
    }
  }
}
```

**成功响应：**

```json
{
  "jsonrpc": "2.0",
  "id": "mcp-recommend-server-tools-call-1234567890",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"users\": [{\"id\": \"u001\", \"name\": \"...\"}]}"
      }
    ]
  }
}
```

**错误响应：**

```json
{
  "jsonrpc": "2.0",
  "id": "mcp-recommend-server-tools-call-1234567890",
  "error": {
    "code": -32603,
    "message": "recommend API error: connection refused"
  }
}
```

### initialize — 初始化（可选）

部分 MCP 客户端会先发送 `initialize` 请求，MCP Server 应返回协议版本和能力声明：

```json
// 响应
{
  "jsonrpc": "2.0",
  "id": "...",
  "result": {
    "protocolVersion": "2024-11-05",
    "capabilities": { "tools": {} },
    "serverInfo": { "name": "mcp-recommend-server", "version": "1.0.0" }
  }
}
```

---

## 实践指南：编写 MCP Server 适配器

### MCP Server 代码结构

项目中已提供一个完整的 MCP Server 适配器实现，位于 `cmd/mcp-recommend-server/main.go`，结构如下：

```
cmd/mcp-recommend-server/
└── main.go
    ├── 工具定义 (recommendTools)       — 定义工具的 name/description/inputSchema
    ├── server 结构体                    — 持有 API 基础地址和 HTTP 客户端
    ├── handleMessage()                 — 路由 JSON-RPC method
    ├── handleInitialize()              — 响应 initialize
    ├── handleToolsList()               — 响应 tools/list，返回工具定义
    ├── handleToolsCall()               — 响应 tools/call，转发到实际 API
    ├── callRecommendAPI()              — 调用外部 API 的具体逻辑
    └── main()                          — 启动 HTTP 服务
```

### 核心实现要点

#### 1. 定义工具的 inputSchema

这是最关键的部分——`inputSchema` 决定了 LLM 如何理解和调用你的工具。需要为每个参数提供准确的 `type` 和 `description`：

```go
var recommendTools = []map[string]interface{}{
    {
        "name":        "find_person_for_male_user",
        "description": "为男性用户推荐匹配的人选。根据用户画像和偏好，从推荐服务获取推荐列表。",
        "inputSchema": map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "user_id": map[string]interface{}{
                    "type":        "string",
                    "description": "男性用户的唯一标识ID",
                },
                "limit": map[string]interface{}{
                    "type":        "integer",
                    "description": "返回推荐人数上限，默认10",
                },
            },
            "required": []string{"user_id"},
        },
    },
}
```

#### 2. 处理 JSON-RPC 路由

MCP Server 的 `/message` 端点需要根据 `method` 字段路由到不同的处理逻辑：

```go
switch req.Method {
case "initialize":
    s.handleInitialize(w, req)
case "tools/list":
    s.handleToolsList(w, req)
case "tools/call":
    s.handleToolsCall(w, req)
default:
    writeRPCError(w, req.ID, -32601, "method not found")
}
```

#### 3. 转发到实际 API

在 `tools/call` 的处理逻辑中，将参数转发到真实的外部 API，并将响应包装成 MCP 标准格式：

```go
func (s *server) callRecommendAPI(args map[string]interface{}) (string, error) {
    url := fmt.Sprintf("%s/api_server/xxx", s.apiBase)
    payload, _ := json.Marshal(args)
    req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
    req.Header.Set("Content-Type", "application/json")
    resp, err := s.httpClient.Do(req)
    // ... 读取响应并返回
}
```

返回时包装为 MCP 标准 content 格式：

```go
writeRPCResult(w, req.ID, map[string]interface{}{
    "content": []map[string]interface{}{
        {"type": "text", "text": result},
    },
})
```

### 配置与启动

#### 1. 在 config.toml 中注册 MCP Server

```toml
[mcp.servers.mcp-recommend-server]
url = "http://localhost:9100"

# 如果需要认证，可以添加 headers
# [mcp.servers.mcp-recommend-server.headers]
# Authorization = "Bearer <TOKEN>"
```

#### 2. 启动 MCP Server 适配器

```bash
go run ./cmd/mcp-recommend-server/ --port 9100 --api-base "http://推荐服务地址"
```

#### 3. 启动 Agent

```bash
./bin/openmanus run --config configs/config.toml
```

Agent 启动后会自动发现并注册 MCP 工具，LLM 在需要时即可调用。

---

## 关键源码索引

| 功能 | 文件 | 关键函数/结构体 |
|------|------|----------------|
| Agent 初始化 MCP | `pkg/agent/core.go` | `NewBaseAgentWithMCP` |
| MCP 工具发现 | `pkg/agent/mcp_discovery.go` | `MCPDiscoveryService.Start`, `discoverToolsFromServer` |
| MCP 工具执行 | `pkg/agent/mcp_executor.go` | `MCPExecutor.ExecuteMCPTool`, `callMCPTool` |
| MCP 传输层 | `pkg/mcp/transport/api_client.go` | `ListTools`, `CallTool`, `DeriveMessageURL` |
| HTTP 发送 | `pkg/mcp/transport/http_sender.go` | `PostJSON` |
| MCP 协议类型 | `pkg/mcp/types.go` | `Message`, `Tool`, `CallToolParams`, `CallToolResult` |
| 工具接口 | `pkg/tool/schema.go` | `Tool`, `MCPTool`, `MCPExecutor` |
| 工具注册 | `pkg/tool/registry.go` | `Registry.RegisterMCPTools` |
| 工具执行器 | `pkg/tool/exec.go` | `Executor.Execute` |
| MCP Server 适配器 | `cmd/mcp-recommend-server/main.go` | `server.handleMessage` |
| MCP 配置 | `pkg/config/config.go` | `MCPConfig`, `MCPServerConfig` |
