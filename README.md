# OpenManus-Go

🤖 **OpenManus-Go** 是一个基于 Go 语言开发的通用 AI Agent 框架，采用 Plan → Tool Use → Observation → Reflection 循环模式，帮助用户通过自然语言指令完成复杂任务。

## ✨ 核心特性

### 🎯 智能 Agent 架构
- **自动规划**：将复杂目标分解为可执行的步骤
- **工具调用**：智能选择并执行适合的工具
- **实时观测**：监控每步执行结果并调整策略
- **反思机制**：定期评估进度并优化执行路径

### 🔌 MCP (Model Context Protocol) 集成
- **MCP 服务器**：提供标准化的工具接口
- **MCP 客户端**：连接和调用外部 MCP 服务
- **自动发现**：动态发现可用的 MCP 工具
- **统一管理**：将 MCP 工具与内置工具统一管理

### 🛠️ 丰富的内置工具生态
- **文件系统**：文件读写、目录操作、权限管理
- **网络请求**：HTTP 客户端、网页爬虫
- **浏览器自动化**：页面操作、截图、数据提取
- **数据库操作**：Redis、MySQL、Elasticsearch 数据操作
- **可扩展架构**：易于添加自定义工具

### 🚀 企业级特性
- **配置管理**：支持 TOML 配置文件和环境变量
- **日志系统**：结构化日志，支持多种输出格式
- **状态追踪**：持久化执行轨迹，支持断点续传
- **容器化部署**：完整的 Docker 支持

## 🏗️ 系统架构

```mermaid
graph TB
    User[用户输入] --> Agent[Agent 核心]
    Agent --> Planner[智能规划器]
    Agent --> Executor[工具执行器]
    Agent --> Memory[记忆管理]
    Agent --> Reflector[反思器]
    
    Planner --> LLM[大语言模型]
    Executor --> Registry[工具注册表]
    
    Registry --> Builtin[内置工具]
    Registry --> MCP[MCP工具]
    
    Builtin --> FS[文件系统]
    Builtin --> HTTP[网络请求]
    Builtin --> Browser[浏览器]
    Builtin --> DB[数据库]
    
    MCP --> External[外部服务]
```

## 📖 目录

- [✨ 核心特性](#-核心特性)
- [🏗️ 系统架构](#️-系统架构)
- [🚀 快速开始](#-快速开始)
- [💬 交互模式详解](#-交互模式详解)
- [📋 使用示例](#-使用示例)
- [🛠️ 内置工具](#️-内置工具)
- [🔌 MCP 支持](#-mcp-model-context-protocol-支持)
- [🐳 Docker 部署](#-docker-部署)
- [⚙️ 配置详解](#️-配置详解)
- [🏗️ 开发指南](#️-开发指南)
- [📊 监控和日志](#-监控和日志)
- [🧪 测试和验证](#-测试和验证)
- [🎯 应用场景](#-应用场景)

## 🚀 快速开始

### 环境要求

- Go 1.21+
- (可选) Docker 用于容器化部署
- (可选) Redis 用于状态存储
- (可选) Chrome/Chromium 用于浏览器自动化

### 安装和构建

```bash
# 1. 克隆项目
git clone https://github.com/your-org/openmanus-go.git
cd openmanus-go

# 2. 安装依赖
go mod download

# 3. 构建项目
make build

# 或者直接使用 go build
go build -o bin/openmanus cmd/openmanus/main.go
```

### 配置设置

```bash
# 1. 复制配置模板
cp configs/config.example.toml configs/config.toml

# 2. 编辑配置文件（设置 LLM API Key）
vim configs/config.toml
```

最小配置示例：
```toml
[llm]
model = "deepseek-chat"
base_url = "https://api.deepseek.com/v1"
api_key = "your-api-key-here"
temperature = 0.1
max_tokens = 4000

[agent]
max_steps = 15
max_duration = "10m"
reflection_steps = 3
```

### 基本使用

```bash
# 1. 交互模式（推荐新手使用）
./bin/openmanus run --config configs/config.toml --interactive
# 或使用 make 命令
make run-interactive

# 2. 单次任务执行（适合脚本调用）
./bin/openmanus run --config configs/config.toml "创建一个hello.txt文件，内容为当前时间"

# 3. 查看可用工具
./bin/openmanus tools list --config configs/config.toml

# 4. 配置验证
./bin/openmanus config validate --config configs/config.toml
```

## 💬 交互模式详解

交互模式是 OpenManus-Go 最直观、最强大的使用方式，允许您与 AI Agent 进行实时对话，逐步完成复杂任务。

### 🚀 启动交互模式

```bash
# 方法1: 使用构建好的二进制文件（推荐）
./bin/openmanus run --config configs/config.toml --interactive

# 方法2: 使用 make 命令
make run-interactive

# 方法3: 直接使用 go run
go run ./cmd/openmanus run --config configs/config.toml --interactive

# 方法4: 带高级参数启动
./bin/openmanus run --interactive --max-steps 20 --verbose --output session.log
```

### 🎮 交互界面

启动后您会看到友好的交互界面：

```
🤖 OpenManus-Go Interactive Mode
Type your goals and press Enter. Type 'quit' or 'exit' to stop.
Commands: /help, /status, /trace, /config

🎯 Goal: 
```

### 📝 基本操作

#### 1. 执行任务
直接输入您想要完成的任务：

```bash
🎯 Goal: 在 workspace 目录创建一个名为 report.txt 的文件，内容为今天的日期
🔄 Executing: 在 workspace 目录创建一个名为 report.txt 的文件，内容为今天的日期
✅ Result: 已成功在 workspace/report.txt 文件中写入今天的日期：2024-01-15

🎯 Goal: 检查刚才创建的文件内容
🔄 Executing: 检查刚才创建的文件内容
✅ Result: 文件 report.txt 的内容：2024年1月15日
```

#### 2. 内置命令

| 命令 | 功能 | 使用示例 |
|------|------|----------|
| `/help` | 显示帮助信息 | `/help` |
| `/status` | 显示 Agent 状态 | `/status` |
| `/trace` | 显示执行轨迹 | `/trace` |
| `/config` | 显示配置信息 | `/config` |
| `quit` 或 `exit` | 退出交互模式 | `quit` |

#### 3. 任务示例

**文件操作：**
```bash
🎯 Goal: 创建一个包含当前目录所有文件列表的清单文件
🎯 Goal: 将 data.json 文件转换为 CSV 格式
🎯 Goal: 备份 workspace 目录下的所有 .txt 文件
```

**网络操作：**
```bash
🎯 Goal: 获取 https://api.github.com/users/octocat 的用户信息并保存到文件
🎯 Goal: 检查 baidu.com 的网络连通性
🎯 Goal: 爬取某个网页的标题和描述信息
```

**数据处理：**
```bash
🎯 Goal: 分析 sales.csv 文件，计算总销售额并生成报告
🎯 Goal: 从多个 JSON 文件中提取特定字段合并成一个文件
🎯 Goal: 对比两个文件的差异并生成对比报告
```

### ⚙️ 高级参数

启动交互模式时可以使用多种参数来定制行为：

```bash
# 设置最大执行步数
./bin/openmanus run --interactive --max-steps 20

# 设置输出文件（保存会话结果）
./bin/openmanus run --interactive --output session-log.txt

# 启用详细日志模式
./bin/openmanus run --interactive --verbose

# 启用调试模式（查看详细执行过程）
./bin/openmanus run --interactive --debug

# 自定义 LLM 参数
./bin/openmanus run --interactive --temperature 0.2 --max-tokens 8000

# 禁用轨迹保存
./bin/openmanus run --interactive --save-trace=false

# 自定义轨迹保存路径
./bin/openmanus run "你的任务" --trace-path="./my-traces"
```

### 📊 轨迹管理

OpenManus-Go 提供完整的执行轨迹管理功能：

```bash
# 查看所有保存的轨迹
./bin/openmanus trace list

# 查看轨迹详细信息
./bin/openmanus trace show <trace-id> --steps --observations

# 清理旧轨迹
./bin/openmanus trace clean --days 30

# 删除指定轨迹
./bin/openmanus trace delete <trace-id>
```

详细文档请参考：[轨迹管理指南](docs/TRACE_MANAGEMENT.md)

### 🛠️ 可用工具一览

在交互模式中，Agent 可以智能选择并使用以下工具：

| 工具类型 | 工具名称 | 主要功能 | 适用场景 |
|----------|----------|----------|----------|
| **文件系统** | `fs` | 文件读写、目录操作 | 文件管理、数据存储 |
| **网络请求** | `http` | HTTP 客户端、API 调用 | 数据获取、服务调用 |
| **网页爬虫** | `crawler` | 网页内容抓取 | 数据收集、信息提取 |
| **浏览器自动化** | `browser` | 页面操作、截图 | UI 自动化、测试 |
| **数据库** | `redis`/`mysql` | 数据存储和查询 | 数据持久化、缓存 |
| **MCP 工具** | 动态加载 | 外部服务集成 | 扩展功能、第三方服务 |

### 🎯 完整使用示例

以下是一个完整的交互会话示例：

```bash
🤖 OpenManus-Go Interactive Mode
🎯 Goal: /help

Available commands:
  /help    - Show this help message  
  /status  - Show agent status
  /trace   - Show current execution trace
  /config  - Show configuration
  quit     - Exit the program

🎯 Goal: 创建一个项目报告，包含当前目录的文件统计

🔄 Executing: 创建一个项目报告，包含当前目录的文件统计
💭 Agent 正在分析任务...
🔧 选择工具: fs (文件系统)
📝 执行操作: 扫描目录、统计文件
✅ Result: 已创建项目报告 workspace/project_report.txt，包含：
- 总文件数: 15
- 文件类型分布: .go(8), .md(4), .toml(2), .txt(1)
- 总大小: 245KB

🎯 Goal: /trace

Current Trace:
  Step 1: [Plan] 分析需要创建项目报告的任务
  Step 2: [Tool] 使用文件系统工具扫描目录
  Step 3: [Observation] 获取文件列表和统计信息
  Step 4: [Tool] 生成报告并写入文件
  Step 5: [Reflection] 任务完成，报告已生成

🎯 Goal: 现在帮我验证刚才创建的报告文件内容是否正确

🔄 Executing: 现在帮我验证刚才创建的报告文件内容是否正确
✅ Result: 验证完成！报告文件 project_report.txt 内容正确：
- 文件统计准确
- 格式清晰易读  
- 包含完整的项目概览信息

🎯 Goal: /status

Agent Status:
  Status: Running
  Type: BaseAgent
  Executed Steps: 8
  Available Tools: 6
  Current Memory: 2.3MB

🎯 Goal: quit
Goodbye!
```

### 🐳 Docker 环境中使用交互模式

如果您使用 Docker 部署：

```bash
# 启动容器并进入交互模式
docker-compose up -d
docker-compose exec openmanus ./openmanus run --interactive

# 或者直接运行交互容器
docker run -it \
  -v $(pwd)/configs:/app/configs \
  -v $(pwd)/workspace:/app/workspace \
  openmanus-go:latest run --interactive
```

### 💡 使用技巧

1. **任务描述要清晰具体**：
   - ✅ 好："在 workspace 目录创建一个名为 data.json 的文件，包含今天的日期和时间"
   - ❌ 差："创建文件"

2. **善用内置命令**：
   - 使用 `/trace` 查看执行过程
   - 使用 `/status` 监控资源使用
   - 使用 `/help` 获取帮助

3. **复杂任务分步骤**：
   - 先执行简单的子任务
   - 验证中间结果
   - 再继续后续步骤

4. **充分利用上下文**：
   - Agent 会记住对话历史
   - 可以引用之前的操作结果
   - 支持连续的任务流

### ❌ 常见问题解决

**Q: 启动时提示 "goal is required"**
```bash
# 确保使用了 --interactive 参数
./bin/openmanus run --config configs/config.toml --interactive
```

**Q: API 调用失败**
```bash
# 检查配置文件中的 API Key
vim configs/config.toml
# 确保 api_key 不是 "your-api-key-here"
```

**Q: 工具执行权限错误**
```bash
# 检查文件系统工具的路径配置
[tools.filesystem]
allowed_paths = ["./workspace", "./data"]  # 确保路径正确
```

**Q: 响应速度慢**
```bash
# 优化 LLM 配置
[llm]
max_tokens = 2000        # 减少 token 数量
temperature = 0.1        # 降低随机性
```

交互模式让您能够像与智能助手对话一样完成各种复杂任务，是体验 OpenManus-Go 强大功能的最佳方式！

## 📋 使用示例

### 文件操作任务
```bash
# 文件创建和管理
./bin/openmanus run "在workspace目录创建一个report.txt文件，写入今天的日期和时间"

# 目录操作
./bin/openmanus run "检查workspace目录下有哪些文件，并创建一个文件清单"
```

### 网络数据获取
```bash
# HTTP 请求
./bin/openmanus run "获取https://httpbin.org/json的内容并保存到data.json文件"

# 网页爬虫
./bin/openmanus run "爬取某个网站的标题和描述信息"
```

### 数据处理任务
```bash
# 数据分析
./bin/openmanus run "分析workspace/sales.csv文件，生成月度销售报告"

# 格式转换
./bin/openmanus run "将JSON文件转换为CSV格式"
```

### MCP 集成示例
```bash
# 使用外部 MCP 服务
./bin/openmanus run "使用股票查询工具获取苹果公司的实时股价"

# 复合任务
./bin/openmanus run "查询比特币价格，如果超过50000美元，发送通知邮件"
```

## 🛠️ 内置工具

| 工具名称 | 功能描述 | 主要用途 |
|---------|----------|----------|
| `fs` | 文件系统操作 | 文件读写、目录管理、权限控制 |
| `http` | HTTP 客户端 | API 调用、数据获取、网络请求 |
| `crawler` | 网页爬虫 | 网页内容抓取、数据收集 |
| `browser` | 浏览器自动化 | 页面操作、截图、表单填写 |
| `redis` | Redis 数据库 | 缓存操作、数据存储 |
| `mysql` | MySQL 数据库 | 关系型数据操作 |
| `elasticsearch` | Elasticsearch 搜索引擎 | 全文搜索、数据分析、日志检索 |

### 工具安全特性

- **路径限制**：文件系统工具支持路径白名单和黑名单
- **域名过滤**：HTTP 工具支持域名访问控制
- **超时控制**：所有网络操作都有超时保护
- **资源限制**：支持文件大小、内存使用限制

## 🔌 MCP (Model Context Protocol) 支持

### MCP 服务器配置

在 `configs/config.toml` 中添加 MCP 服务器：

```toml
[[mcp_servers]]
name = "stock-helper"
transport = "sse"
url = "https://api.example.com/mcp/stock"
timeout = 30

[[mcp_servers]]
name = "weather-service"
transport = "http"
url = "https://weather.example.com/mcp"
```

### MCP 工具发现

启动时自动发现 MCP 工具：
```
🔍 正在发现 MCP 工具...
  ✅ stock-price (股价查询)
  ✅ weather-forecast (天气预报)  
  ✅ news-search (新闻搜索)
📊 共发现 3 个 MCP 工具
```

### MCP 工具使用

Agent 会自动选择最适合的 MCP 工具：
```bash
./bin/openmanus run "查询特斯拉今日股价并分析趋势"
# Agent 自动使用 stock-price 工具获取数据
```

## 🐳 Docker 部署

### 快速启动

```bash
# 1. 设置环境变量
export OPENMANUS_LLM_API_KEY="your-api-key"
export OPENMANUS_LLM_MODEL="deepseek-chat"

# 2. 启动服务
docker-compose up -d

# 3. 检查服务状态
docker-compose ps
```

### 完整部署（包含监控）

```bash
# 启动完整服务栈
docker-compose --profile full up -d

# 访问服务
# - OpenManus: http://localhost:8080
# - Grafana: http://localhost:3000
# - Redis: localhost:6379
```

### 容器服务说明

- **openmanus**: 主应用服务
- **redis**: 状态存储和缓存
- **mysql**: 持久化数据存储（可选）
- **grafana**: 监控面板（可选）
- **prometheus**: 指标收集（可选）

## ⚙️ 配置详解

### LLM 配置

```toml
[llm]
model = "deepseek-chat"                    # 支持 OpenAI 兼容模型
base_url = "https://api.deepseek.com/v1"   # API 端点
api_key = "sk-xxx"                         # API 密钥
temperature = 0.1                          # 生成温度 (0.0-1.0)
max_tokens = 4000                          # 单次最大 token 数
timeout = 60                               # 请求超时（秒）
```

### Agent 配置

```toml
[agent]
max_steps = 15                             # 最大执行步数
max_tokens = 10000                         # token 预算限制
max_duration = "10m"                       # 最大执行时间
reflection_steps = 3                       # 反思步数间隔
max_retries = 3                            # 失败重试次数
```

### 工具配置

```toml
[tools.filesystem]
allowed_paths = ["./workspace", "./data"]  # 允许访问路径
blocked_paths = ["/etc", "/sys"]           # 禁止访问路径
max_file_size = "100MB"                    # 最大文件大小

[tools.http]
timeout = 45                               # 请求超时
blocked_domains = ["localhost"]            # 禁止访问域名
user_agent = "OpenManus-Go/1.0"           # 用户代理

[tools.browser]
headless = true                            # 无头模式
timeout = 60                               # 页面超时
chrome_args = ["--no-sandbox"]            # Chrome 参数
```

## 🏗️ 开发指南

### 项目结构

```
openmanus-go/
├── cmd/openmanus/          # CLI 应用入口
├── pkg/                    # 核心库
│   ├── agent/             # Agent 实现
│   ├── tool/              # 工具系统
│   ├── llm/               # LLM 客户端
│   ├── config/            # 配置管理
│   ├── state/             # 状态管理
│   └── mcp/               # MCP 协议
├── examples/              # 使用示例
├── configs/               # 配置文件
├── deployments/           # 部署配置
└── docs/                  # 文档
```

### 创建自定义工具

```go
package main

import (
    "context"
    "openmanus-go/pkg/tool"
)

// 实现 Tool 接口
type CustomTool struct {
    *tool.BaseTool
}

func (t *CustomTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
    // 实现工具逻辑
    input := args["input"].(string)
    
    return map[string]any{
        "success": true,
        "result":  "处理结果: " + input,
    }, nil
}

// 注册工具
func init() {
    schema := tool.CreateJSONSchema("object", map[string]any{
        "input": tool.StringProperty("输入参数"),
    }, []string{"input"})
    
    baseTool := tool.NewBaseTool(
        "custom-tool",
        "自定义工具示例",
        schema,
        schema,
    )
    
    customTool := &CustomTool{BaseTool: baseTool}
    tool.Register(customTool)
}
```

### 扩展 MCP 集成

```go
// 添加新的 MCP 服务器
mcpConfig := &config.MCPServerConfig{
    Name:      "my-service",
    Transport: "sse",
    URL:       "https://my-mcp-server.com/api",
    Timeout:   30,
}

// 注册到配置
config.AddMCPServer(mcpConfig)
```

## 📊 监控和日志

### 日志配置

```toml
[logging]
level = "info"                              # debug|info|warn|error
output = "console"                          # console|file|both  
format = "json"                             # text|json
file_path = "./logs/openmanus.log"          # 日志文件路径
```

### 性能监控

```toml
[monitoring]
enabled = true                              # 启用监控
metrics_port = 9090                         # 指标端口
prometheus_path = "/metrics"                # Prometheus 路径
```

### 执行轨迹

每次执行都会生成详细的轨迹记录：
```bash
# 查看执行轨迹
ls ./workspace/traces/

# 轨迹包含的信息：
# - 执行步骤和时间
# - 工具调用和结果  
# - 错误和重试记录
# - 性能指标
```

## 🧪 测试和验证

### 运行测试

```bash
# 运行所有测试
make test

# 运行特定模块测试
go test ./pkg/agent/...

# 集成测试
make test-integration
```

### 工具测试

```bash
# 测试所有工具
./bin/openmanus tools test

# 测试特定工具
./bin/openmanus tools test --name fs

# 测试 MCP 连接
./bin/openmanus mcp test
```

### 配置验证

```bash
# 验证配置文件
./bin/openmanus config validate

# 检查 LLM 连接
./bin/openmanus config test-llm

# 检查工具可用性
./bin/openmanus config test-tools
```

## 🎯 应用场景

### 文件和数据处理
- 批量文件操作和格式转换
- 数据清理和格式化
- 日志分析和报告生成

### 网络数据收集
- API 数据获取和整合
- 网页内容抓取和监控
- 多源数据聚合

### 自动化运维
- 配置文件管理
- 系统状态检查
- 定时任务执行

### 业务流程自动化
- 表单数据处理
- 报告自动生成
- 多系统数据同步

## 📚 文档和资源

- [详细文档](./docs/) - 完整的开发和使用文档
- [示例代码](./examples/) - 丰富的使用示例
- [配置说明](./configs/) - 配置文件详解
- [部署指南](./deployments/) - 生产环境部署

## 🤝 贡献指南

我们欢迎所有形式的贡献！

### 如何贡献

1. Fork 本项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

### 贡献方向

- 🐛 Bug 修复和问题报告
- ✨ 新功能开发
- 🛠️ 工具开发和完善
- 📚 文档改进
- 🧪 测试覆盖率提升

## 📄 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。

## 🙏 致谢

- 感谢 [OpenManus](https://github.com/openmanus/openmanus) 项目的启发
- 感谢 [Model Context Protocol](https://modelcontextprotocol.io) 的开放标准
- 感谢所有贡献者的支持和反馈

---

**OpenManus-Go** - 让 AI Agent 开发变得简单而强大！ 🚀✨