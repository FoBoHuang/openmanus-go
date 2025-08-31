# MCP集成指南

本文档详细介绍如何在 OpenManus-Go 中使用 MCP (Model Context Protocol) 集成功能，包括服务器搭建、客户端使用和第三方服务集成。

## 📋 目录

- [MCP 概述](#mcp-概述)
- [MCP 服务器](#mcp-服务器)
- [MCP 客户端](#mcp-客户端)
- [工具集成](#工具集成)
- [外部服务](#外部服务)
- [最佳实践](#最佳实践)

## 🌐 MCP 概述

MCP (Model Context Protocol) 是一个标准化的协议，用于 AI 模型与工具和服务之间的通信。OpenManus-Go 实现了完整的 MCP 支持，包括：

- MCP 服务器：将内置工具暴露为 MCP 服务
- MCP 客户端：连接到其他 MCP 服务器
- 标准化的工具接口
- REST API 兼容性

### 启动 MCP 服务器

```bash
# 启动 MCP 服务器（默认端口 8080）
./bin/openmanus mcp

# 指定端口和主机
./bin/openmanus mcp --host 0.0.0.0 --port 9000

# 生成工具文档
./bin/openmanus mcp --docs
```

### MCP 服务器端点

启动后，MCP 服务器提供以下端点：

- `POST /` - MCP 协议端点 (JSON-RPC)
- `GET /tools` - 获取工具列表 (REST API)
- `POST /tools/invoke` - 调用工具 (REST API)
- `GET /health` - 健康检查

### 可用工具

MCP 服务器暴露以下内置工具：

1. **HTTP 工具** - 发送 HTTP 请求
2. **文件系统工具** - 文件操作
3. **浏览器工具** - 网页自动化
4. **爬虫工具** - 网页抓取
5. **Redis 工具** - Redis 数据库操作
6. **MySQL 工具** - MySQL 数据库操作

### MCP 客户端使用

```go
package main

import (
    "context"
    "openmanus-go/pkg/mcp"
)

func main() {
    // 创建 MCP 客户端
    client := mcp.NewClient("http://localhost:8080")
    
    ctx := context.Background()
    
    // 初始化连接
    err := client.Initialize(ctx)
    if err != nil {
        panic(err)
    }
    
    // 获取工具列表
    tools, err := client.ListTools(ctx)
    if err != nil {
        panic(err)
    }
    
    // 调用工具
    result, err := client.CallTool(ctx, "http", map[string]interface{}{
        "url": "https://api.example.com/data",
        "method": "GET",
    })
    if err != nil {
        panic(err)
    }
}
```

## 多 Agent 协作

### 概述

多 Agent 协作功能允许创建复杂的工作流，其中多个 Agent 可以：

- 并行或顺序执行任务
- 共享状态和数据
- 基于依赖关系进行任务编排
- 支持 DAG (有向无环图) 工作流

### 执行模式

1. **Sequential (顺序)** - 任务按顺序执行
2. **Parallel (并行)** - 任务并行执行
3. **DAG (依赖图)** - 基于依赖关系执行

### Agent 类型

- **general** - 通用 Agent
- **data_analysis** - 数据分析 Agent
- **web_scraper** - 网页爬虫 Agent
- **file_processor** - 文件处理 Agent

### 启动多 Agent 流程

```bash
# 顺序执行 2 个 Agent
./bin/openmanus flow --mode sequential --agents 2

# 并行执行数据分析工作流
./bin/openmanus flow --mode parallel --data-analysis

# DAG 模式执行 3 个 Agent
./bin/openmanus flow --mode dag --agents 3
```

### 工作流定义

```go
package main

import (
    "openmanus-go/pkg/flow"
)

func createWorkflow() *flow.Workflow {
    workflow := flow.NewWorkflow("my-workflow", "My Workflow", flow.ExecutionModeDAG)
    
    // 任务 1: 数据收集
    task1 := flow.NewTask("collect", "数据收集", "general", "收集数据")
    
    // 任务 2: 数据处理（依赖任务 1）
    task2 := flow.NewTask("process", "数据处理", "data_analysis", "处理数据")
    task2.Dependencies = []string{"collect"}
    
    // 任务 3: 生成报告（依赖任务 2）
    task3 := flow.NewTask("report", "生成报告", "file_processor", "生成报告")
    task3.Dependencies = []string{"process"}
    
    workflow.AddTask(task1)
    workflow.AddTask(task2)
    workflow.AddTask(task3)
    
    return workflow
}
```

### 流程引擎使用

```go
package main

import (
    "context"
    "openmanus-go/pkg/flow"
    "openmanus-go/pkg/llm"
    "openmanus-go/pkg/tool"
)

func main() {
    // 创建组件
    llmClient := llm.NewOpenAIClient(config)
    toolRegistry := tool.NewRegistry()
    agentFactory := flow.NewDefaultAgentFactory(llmClient, toolRegistry)
    flowEngine := flow.NewDefaultFlowEngine(agentFactory, 5)
    
    // 创建工作流
    workflow := createWorkflow()
    
    // 执行工作流
    ctx := context.Background()
    execution, err := flowEngine.Execute(ctx, workflow, input)
    if err != nil {
        panic(err)
    }
    
    // 监听事件
    eventChan, _ := flowEngine.Subscribe(execution.ID)
    for event := range eventChan {
        fmt.Printf("Event: %s\n", event.Message)
    }
}
```

## 使用示例

### 1. MCP 客户端示例

参见 `examples/mcp_demo/main.go`：

```bash
# 启动 MCP 服务器
./bin/openmanus mcp &

# 运行 MCP 客户端示例
go run examples/mcp_demo/main.go
```

### 2. 多 Agent 协作示例

参见 `examples/multi_agent_demo/main.go`：

```bash
# 运行多 Agent 示例
go run examples/multi_agent_demo/main.go
```

### 3. 数据分析工作流

```bash
# 启动数据分析工作流
./bin/openmanus flow --data-analysis --mode parallel
```

## API 参考

### MCP 协议

#### 初始化

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {
      "name": "client-name",
      "version": "1.0.0"
    }
  }
}
```

#### 工具列表

```json
{
  "jsonrpc": "2.0",
  "id": "2",
  "method": "tools/list"
}
```

#### 工具调用

```json
{
  "jsonrpc": "2.0",
  "id": "3",
  "method": "tools/call",
  "params": {
    "name": "http",
    "arguments": {
      "url": "https://api.example.com",
      "method": "GET"
    }
  }
}
```

### REST API

#### 获取工具列表

```bash
curl http://localhost:8080/tools
```

#### 调用工具

```bash
curl -X POST http://localhost:8080/tools/invoke \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "http",
    "args": {
      "url": "https://api.example.com",
      "method": "GET"
    }
  }'
```

#### 健康检查

```bash
curl http://localhost:8080/health
```

## 配置

### MCP 服务器配置

MCP 服务器使用默认配置，包括所有可用的内置工具。可以通过配置文件自定义：

```toml
[tools.database.redis]
addr = "localhost:6379"
password = ""
db = 0

[tools.database.mysql]
dsn = "user:password@tcp(localhost:3306)/database"

[tools.browser]
headless = true
timeout = 30

[tools.http]
allowed_domains = ["*"]
blocked_domains = []
```

### 多 Agent 配置

```toml
[agent]
max_steps = 10
temperature = 0.1
max_tokens = 8000

[flow]
max_concurrency = 5
default_timeout = "5m"
```

## 故障排除

### 常见问题

1. **MCP 服务器启动失败**
   - 检查端口是否被占用
   - 确认工具依赖是否安装（如 Redis、MySQL）

2. **多 Agent 执行失败**
   - 检查 LLM API 密钥配置
   - 确认网络连接正常

3. **工具调用失败**
   - 检查工具参数是否正确
   - 查看服务器日志获取详细错误信息

### 调试模式

```bash
# 启用调试模式
./bin/openmanus mcp --debug
./bin/openmanus flow --debug --verbose
```

## 扩展开发

### 自定义 Agent 类型

```go
func (f *DefaultAgentFactory) CreateAgent(agentType string, config map[string]interface{}) (agent.Agent, error) {
    switch agentType {
    case "custom_agent":
        return createCustomAgent(config)
    default:
        return f.createGeneralAgent(config)
    }
}
```

### 自定义工具

```go
type CustomTool struct {
    *tool.BaseTool
}

func (t *CustomTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
    // 实现自定义逻辑
    return map[string]any{"result": "success"}, nil
}
```

### 工作流持久化

```go
// 保存工作流到文件
func saveWorkflow(workflow *flow.Workflow, filename string) error {
    data, err := json.MarshalIndent(workflow, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(filename, data, 0644)
}

// 从文件加载工作流
func loadWorkflow(filename string) (*flow.Workflow, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, err
    }
    
    var workflow flow.Workflow
    err = json.Unmarshal(data, &workflow)
    return &workflow, err
}
```

## 性能优化

### MCP 服务器

- 使用连接池管理数据库连接
- 实现工具结果缓存
- 配置适当的超时时间

### 多 Agent 协作

- 调整并发数量以平衡性能和资源使用
- 使用任务优先级进行调度
- 实现智能重试机制

## 安全考虑

### MCP 服务器

- 实现认证和授权机制
- 限制工具访问权限
- 配置网络访问控制

### 多 Agent 协作

- 验证任务输入参数
- 实现资源使用限制
- 监控异常行为

---

更多信息请参考：
- [OpenManus-Go 架构文档](ARCHITECTURE.md)
- [工具开发指南](TOOLS.md)
- [API 参考文档](API.md)
