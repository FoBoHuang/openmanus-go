# API 参考文档

本文档提供 OpenManus-Go 的完整 API 参考，包括核心接口、MCP 协议和 REST API。

## 📋 目录

- [核心接口](#核心接口)
- [Agent API](#agent-api)
- [工具 API](#工具-api)
- [MCP 协议](#mcp-协议)
- [REST API](#rest-api)
- [配置 API](#配置-api)

## 🔧 核心接口

### Agent 接口

Agent 是系统的核心控制器，负责协调整个执行过程。

```go
type Agent interface {
    // 执行完整的任务循环
    Loop(ctx context.Context, goal string) (*state.Trace, error)
    
    // 规划下一步行动
    Plan(ctx context.Context, goal string, trace *state.Trace) (*state.Action, error)
    
    // 执行具体行动
    Act(ctx context.Context, action *state.Action) (*state.Observation, error)
    
    // 反思执行结果
    Reflect(ctx context.Context, trace *state.Trace) (*state.Reflection, error)
    
    // 获取 Agent 状态
    GetStatus() AgentStatus
    
    // 停止执行
    Stop() error
}
```

#### 使用示例

```go
package main

import (
    "context"
    "log"
    
    "openmanus-go/pkg/agent"
    "openmanus-go/pkg/config"
    "openmanus-go/pkg/llm"
    "openmanus-go/pkg/tool"
)

func main() {
    // 创建配置
    cfg := config.LoadConfig("config.toml")
    
    // 创建 LLM 客户端
    llmClient := llm.NewOpenAIClient(cfg.ToLLMConfig())
    
    // 创建工具注册表
    toolRegistry := tool.DefaultRegistry
    
    // 创建 Agent
    agent := agent.NewBaseAgent(llmClient, toolRegistry, nil)
    
    // 执行任务
    ctx := context.Background()
    trace, err := agent.Loop(ctx, "创建一个 hello.txt 文件")
    if err != nil {
        log.Fatalf("执行失败: %v", err)
    }
    
    log.Printf("任务完成: %s", trace.Status)
}
```

### Tool 接口

工具接口定义了所有工具必须实现的方法。

```go
type Tool interface {
    // 工具名称（唯一标识）
    Name() string
    
    // 工具描述
    Description() string
    
    // 输入参数的 JSON Schema
    InputSchema() map[string]any
    
    // 输出结果的 JSON Schema  
    OutputSchema() map[string]any
    
    // 执行工具逻辑
    Invoke(ctx context.Context, args map[string]any) (map[string]any, error)
}
```

#### 创建自定义工具

```go
type CustomTool struct {
    *tool.BaseTool
}

func NewCustomTool() *CustomTool {
    inputSchema := tool.CreateJSONSchema("object", map[string]any{
        "message": tool.StringProperty("要处理的消息"),
        "count":   tool.NumberProperty("重复次数"),
    }, []string{"message"})

    outputSchema := tool.CreateJSONSchema("object", map[string]any{
        "success": tool.BooleanProperty("操作是否成功"),
        "result":  tool.StringProperty("处理结果"),
    }, []string{"success"})

    baseTool := tool.NewBaseTool(
        "custom_tool",
        "自定义工具示例",
        inputSchema,
        outputSchema,
    )

    return &CustomTool{BaseTool: baseTool}
}

func (t *CustomTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
    message := args["message"].(string)
    count := int(args["count"].(float64))
    
    result := strings.Repeat(message+" ", count)
    
    return map[string]any{
        "success": true,
        "result":  result,
    }, nil
}

// 注册工具
func init() {
    tool.Register(NewCustomTool())
}
```

### LLM Client 接口

LLM 客户端接口提供与大语言模型通信的抽象。

```go
type Client interface {
    // 同步聊天
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    
    // 流式聊天
    ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatResponse, error)
    
    // 获取模型名称
    GetModel() string
    
    // 设置模型名称
    SetModel(model string)
    
    // 获取支持的功能
    GetCapabilities() Capabilities
}
```

#### 聊天请求和响应

```go
type ChatRequest struct {
    Model       string           `json:"model"`
    Messages    []Message        `json:"messages"`
    Tools       []ToolDefinition `json:"tools,omitempty"`
    Temperature *float64         `json:"temperature,omitempty"`
    MaxTokens   *int             `json:"max_tokens,omitempty"`
    Stream      bool             `json:"stream,omitempty"`
}

type ChatResponse struct {
    ID      string   `json:"id"`
    Object  string   `json:"object"`
    Created int64    `json:"created"`
    Model   string   `json:"model"`
    Choices []Choice `json:"choices"`
    Usage   Usage    `json:"usage"`
}

type Choice struct {
    Index        int      `json:"index"`
    Message      Message  `json:"message"`
    FinishReason string   `json:"finish_reason"`
}

type Message struct {
    Role      string     `json:"role"`
    Content   string     `json:"content"`
    ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}
```

### 状态管理

```go
type Trace struct {
    ID        string      `json:"id"`
    Goal      string      `json:"goal"`
    Steps     []Step      `json:"steps"`
    Budget    Budget      `json:"budget"`
    Status    TraceStatus `json:"status"`
    Result    string      `json:"result,omitempty"`
    Error     string      `json:"error,omitempty"`
    CreatedAt time.Time   `json:"created_at"`
    UpdatedAt time.Time   `json:"updated_at"`
}

type Step struct {
    ID          string       `json:"id"`
    Action      Action       `json:"action"`
    Observation Observation  `json:"observation"`
    Reflection  string       `json:"reflection,omitempty"`
    Timestamp   time.Time    `json:"timestamp"`
}

type Action struct {
    Name string         `json:"name"`
    Args map[string]any `json:"args"`
}

type Observation struct {
    Success bool           `json:"success"`
    Result  map[string]any `json:"result,omitempty"`
    Error   string         `json:"error,omitempty"`
}
```

## 🔧 Agent API

### BaseAgent

BaseAgent 是默认的 Agent 实现。

#### 构造函数

```go
func NewBaseAgent(
    llmClient llm.Client,
    toolRegistry *tool.Registry,
    store state.Store,
) *BaseAgent
```

#### 配置选项

```go
type AgentConfig struct {
    MaxSteps        int           `mapstructure:"max_steps"`
    MaxTokens       int           `mapstructure:"max_tokens"`
    MaxDuration     time.Duration `mapstructure:"max_duration"`
    ReflectionSteps int           `mapstructure:"reflection_steps"`
    MaxRetries      int           `mapstructure:"max_retries"`
    RetryBackoff    time.Duration `mapstructure:"retry_backoff"`
}
```

#### 方法详解

**Loop() 方法**
```go
func (a *BaseAgent) Loop(ctx context.Context, goal string) (*state.Trace, error)
```
- 执行完整的任务循环直到目标达成或预算耗尽
- 自动处理规划、执行、观察、反思的循环
- 支持上下文取消和超时控制

**Plan() 方法**
```go
func (a *BaseAgent) Plan(ctx context.Context, goal string, trace *state.Trace) (*state.Action, error)
```
- 基于目标和历史轨迹规划下一步行动
- 使用 LLM 进行智能决策
- 返回具体的工具调用动作

**Act() 方法**
```go
func (a *BaseAgent) Act(ctx context.Context, action *state.Action) (*state.Observation, error)
```
- 执行具体的工具调用
- 处理参数验证和错误恢复
- 返回执行结果观察

## 🛠️ 工具 API

### 工具注册表

```go
type Registry struct {
    tools map[string]Tool
    mu    sync.RWMutex
}

// 注册工具
func (r *Registry) Register(tool Tool) error

// 获取工具
func (r *Registry) Get(name string) (Tool, bool)

// 列出所有工具
func (r *Registry) List() []Tool

// 获取工具清单（供 LLM 使用）
func (r *Registry) GetToolsManifest() []ToolDefinition

// 调用工具
func (r *Registry) Invoke(ctx context.Context, name string, args map[string]any) (map[string]any, error)
```

### 工具执行器

```go
type Executor struct {
    registry *Registry
    timeout  time.Duration
}

// 执行单个动作
func (e *Executor) Execute(ctx context.Context, action *state.Action) (*state.Observation, error)

// 批量执行动作
func (e *Executor) BatchExecute(ctx context.Context, actions []*state.Action) ([]*state.Observation, error)
```

### 内置工具

#### 文件系统工具

```go
// 工具名称: "fs"
type FileSystemArgs struct {
    Operation string `json:"operation"` // read, write, list, delete, mkdir, exists, stat
    Path      string `json:"path"`
    Content   string `json:"content,omitempty"`
    Recursive bool   `json:"recursive,omitempty"`
}

type FileSystemResult struct {
    Success bool   `json:"success"`
    Content string `json:"content,omitempty"`
    Files   []File `json:"files,omitempty"`
    Info    *File  `json:"info,omitempty"`
    Error   string `json:"error,omitempty"`
}
```

#### HTTP 工具

```go
// 工具名称: "http"
type HTTPArgs struct {
    URL     string            `json:"url"`
    Method  string            `json:"method"`
    Headers map[string]string `json:"headers,omitempty"`
    Body    string            `json:"body,omitempty"`
    Timeout int               `json:"timeout,omitempty"`
}

type HTTPResult struct {
    Success     bool              `json:"success"`
    StatusCode  int               `json:"status_code"`
    Headers     map[string]string `json:"headers"`
    Body        string            `json:"body"`
    ContentType string            `json:"content_type"`
    Error       string            `json:"error,omitempty"`
}
```

#### 浏览器工具

```go
// 工具名称: "browser"
type BrowserArgs struct {
    Operation string            `json:"operation"` // navigate, click, type, get_text, screenshot
    URL       string            `json:"url,omitempty"`
    Selector  string            `json:"selector,omitempty"`
    Text      string            `json:"text,omitempty"`
    Options   map[string]any    `json:"options,omitempty"`
}
```

## 📡 MCP 协议

### JSON-RPC 2.0 基础

MCP 基于 JSON-RPC 2.0 协议，所有请求和响应都遵循以下格式：

#### 请求格式
```json
{
  "jsonrpc": "2.0",
  "id": "unique-request-id",
  "method": "method_name",
  "params": {
    // 方法参数
  }
}
```

#### 响应格式
```json
{
  "jsonrpc": "2.0",
  "id": "unique-request-id",
  "result": {
    // 成功结果
  }
}
```

#### 错误格式
```json
{
  "jsonrpc": "2.0",
  "id": "unique-request-id",
  "error": {
    "code": -32600,
    "message": "Invalid Request",
    "data": {}
  }
}
```

### MCP 方法

#### 初始化
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "tools": {}
    },
    "clientInfo": {
      "name": "openmanus-go",
      "version": "1.0.0"
    }
  }
}
```

#### 获取工具列表
```json
{
  "jsonrpc": "2.0",
  "id": "2",
  "method": "tools/list",
  "params": {}
}
```

响应:
```json
{
  "jsonrpc": "2.0",
  "id": "2",
  "result": {
    "tools": [
      {
        "name": "fs",
        "description": "文件系统操作工具",
        "inputSchema": {
          "type": "object",
          "properties": {
            "operation": {"type": "string"},
            "path": {"type": "string"}
          },
          "required": ["operation", "path"]
        }
      }
    ]
  }
}
```

#### 调用工具
```json
{
  "jsonrpc": "2.0",
  "id": "3",
  "method": "tools/call",
  "params": {
    "name": "fs",
    "arguments": {
      "operation": "read",
      "path": "./test.txt"
    }
  }
}
```

### MCP 客户端

```go
type Client struct {
    transport Transport
    timeout   time.Duration
}

// 初始化连接
func (c *Client) Initialize(ctx context.Context) error

// 获取工具列表
func (c *Client) ListTools(ctx context.Context) ([]ToolDefinition, error)

// 调用工具
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (map[string]interface{}, error)

// 关闭连接
func (c *Client) Close() error
```

### MCP 服务器

```go
type Server struct {
    registry *tool.Registry
    handler  *Handler
}

// 启动服务器
func (s *Server) Start(ctx context.Context, addr string) error

// 停止服务器
func (s *Server) Stop() error

// 注册工具
func (s *Server) RegisterTool(tool Tool) error
```

## 🌐 REST API

OpenManus-Go 提供 REST API 接口，方便非 MCP 客户端集成。

### 端点列表

| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/health` | 健康检查 |
| GET | `/tools` | 获取工具列表 |
| POST | `/tools/invoke` | 调用工具 |
| GET | `/tools/{name}` | 获取特定工具信息 |
| POST | `/chat` | 聊天接口 |

### 健康检查

```bash
GET /health
```

响应:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "tools_count": 6,
  "version": "1.0.0"
}
```

### 获取工具列表

```bash
GET /tools
```

响应:
```json
{
  "tools": [
    {
      "name": "fs",
      "description": "文件系统操作工具",
      "input_schema": {...},
      "output_schema": {...}
    }
  ]
}
```

### 调用工具

```bash
POST /tools/invoke
Content-Type: application/json

{
  "tool": "fs",
  "args": {
    "operation": "read",
    "path": "./test.txt"
  }
}
```

响应:
```json
{
  "success": true,
  "result": {
    "content": "file content here"
  },
  "execution_time": 0.05
}
```

### 聊天接口

```bash
POST /chat
Content-Type: application/json

{
  "message": "创建一个测试文件",
  "stream": false
}
```

响应:
```json
{
  "response": "我将为您创建一个测试文件",
  "trace_id": "trace-123",
  "tools_used": ["fs"],
  "execution_time": 2.5
}
```

## ⚙️ 配置 API

### 配置结构

```go
type Config struct {
    LLM     LLMConfig     `mapstructure:"llm"`
    Agent   AgentConfig   `mapstructure:"agent"`
    Server  ServerConfig  `mapstructure:"server"`
    Storage StorageConfig `mapstructure:"storage"`
    Tools   ToolsConfig   `mapstructure:"tools"`
    Logging LoggingConfig `mapstructure:"logging"`
}

type LLMConfig struct {
    Model       string        `mapstructure:"model"`
    BaseURL     string        `mapstructure:"base_url"`
    APIKey      string        `mapstructure:"api_key"`
    Temperature float64       `mapstructure:"temperature"`
    MaxTokens   int           `mapstructure:"max_tokens"`
    Timeout     time.Duration `mapstructure:"timeout"`
}
```

### 配置加载

```go
// 从文件加载
func LoadConfig(path string) (*Config, error)

// 从环境变量加载
func LoadConfigFromEnv() (*Config, error)

// 验证配置
func (c *Config) Validate() error

// 转换为 LLM 配置
func (c *Config) ToLLMConfig() *llm.Config
```

### 环境变量支持

```bash
# LLM 配置
export OPENMANUS_LLM_MODEL="gpt-4"
export OPENMANUS_LLM_API_KEY="your-api-key"

# 服务器配置
export OPENMANUS_SERVER_HOST="0.0.0.0"
export OPENMANUS_SERVER_PORT="8080"

# 存储配置
export OPENMANUS_STORAGE_TYPE="redis"
export OPENMANUS_REDIS_ADDR="redis:6379"
```

## 🔍 错误码

### Agent 错误

| 错误码 | 描述 |
|--------|------|
| `AGENT_001` | Agent 初始化失败 |
| `AGENT_002` | 任务执行超时 |
| `AGENT_003` | 预算耗尽 |
| `AGENT_004` | LLM 调用失败 |

### 工具错误

| 错误码 | 描述 |
|--------|------|
| `TOOL_001` | 工具未找到 |
| `TOOL_002` | 参数验证失败 |
| `TOOL_003` | 工具执行失败 |
| `TOOL_004` | 权限不足 |

### MCP 错误

| 错误码 | 描述 |
|--------|------|
| `MCP_001` | 协议版本不兼容 |
| `MCP_002` | 连接失败 |
| `MCP_003` | 方法未实现 |
| `MCP_004` | 传输错误 |

## 📚 使用示例

### 完整示例

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "openmanus-go/pkg/agent"
    "openmanus-go/pkg/config"
    "openmanus-go/pkg/llm"
    "openmanus-go/pkg/tool"
    "openmanus-go/pkg/state"
)

func main() {
    // 1. 加载配置
    cfg, err := config.LoadConfig("config.toml")
    if err != nil {
        log.Fatalf("配置加载失败: %v", err)
    }
    
    // 2. 创建 LLM 客户端
    llmClient := llm.NewOpenAIClient(cfg.ToLLMConfig())
    
    // 3. 创建工具注册表并注册工具
    toolRegistry := tool.NewRegistry()
    tool.RegisterBuiltinTools(toolRegistry)
    
    // 4. 创建状态存储
    store := state.NewFileStore(cfg.Storage.BasePath)
    
    // 5. 创建 Agent
    agent := agent.NewBaseAgent(llmClient, toolRegistry, store)
    
    // 6. 执行任务
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    
    trace, err := agent.Loop(ctx, "分析当前目录的文件，生成一个统计报告")
    if err != nil {
        log.Fatalf("任务执行失败: %v", err)
    }
    
    // 7. 输出结果
    fmt.Printf("任务状态: %s\n", trace.Status)
    fmt.Printf("执行步数: %d\n", len(trace.Steps))
    fmt.Printf("使用时间: %v\n", trace.UpdatedAt.Sub(trace.CreatedAt))
    
    if trace.Status == state.TraceStatusCompleted {
        fmt.Printf("任务结果: %s\n", trace.Result)
    }
}
```

---

这份 API 参考文档涵盖了 OpenManus-Go 的所有主要接口和功能。更多详细信息请参考源码注释和示例代码。

**相关文档**: [核心概念](CONCEPTS.md) → [工具开发](TOOLS.md) → [MCP集成](MCP_INTEGRATION.md)
