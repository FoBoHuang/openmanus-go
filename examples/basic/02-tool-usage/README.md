# 工具使用示例

这个示例展示了如何在 OpenManus-Go 框架中注册、配置和使用各种内置工具。

## 🎯 学习目标

通过这个示例，你将学会：
- 如何注册多种内置工具
- 如何直接调用工具方法
- 如何让 Agent 智能选择工具
- 工具的配置和权限管理
- 工具使用的最佳实践

## 🔧 支持的工具

### 核心工具（无依赖）
- **文件系统工具 (fs)** - 文件读写、目录操作
- **HTTP 工具 (http)** - HTTP 请求和响应处理
- **爬虫工具 (crawler)** - 网页内容抓取

### 可选工具（需要外部依赖）
- **浏览器工具 (browser)** - 需要 Chrome/Chromium
- **Redis 工具 (redis)** - 需要 Redis 服务
- **MySQL 工具 (mysql)** - 需要 MySQL 服务

## 📋 前置条件

### 基本要求
1. **Go 环境**：Go 1.21+
2. **项目构建**：运行 `make build`
3. **工作目录**：确保 `workspace/` 目录存在

### 可选依赖
```bash
# Chrome/Chromium（用于浏览器工具）
# Ubuntu/Debian
sudo apt-get install chromium-browser

# macOS
brew install chromium

# Redis（用于 Redis 工具）
# Ubuntu/Debian
sudo apt-get install redis-server

# macOS
brew install redis

# MySQL（用于 MySQL 工具）
# Ubuntu/Debian
sudo apt-get install mysql-server

# macOS
brew install mysql
```

## 🚀 运行示例

```bash
# 进入示例目录
cd examples/basic/02-tool-usage

# 运行示例
go run main.go
```

## 📊 预期输出

### 无 API Key 模式
```
🔧 OpenManus-Go Tool Usage Example
==================================

⚠️  未设置 LLM API Key，将演示工具注册和基本调用

✅ 基础组件已创建

🔧 注册内置工具...
  ✅ 文件系统工具 (fs)
  ✅ HTTP 工具 (http)
  ⚠️  浏览器工具注册失败 (可能缺少 Chrome): browser initialization failed
  ✅ 爬虫工具 (crawler)
  ⚠️  Redis 工具注册失败 (可能缺少 Redis 服务): connection refused
  ⚠️  MySQL 工具注册失败 (可能缺少 MySQL 服务): connection refused

📊 总计注册了 3 个工具

📋 工具详细信息:
================
1. fs
   描述: 文件系统操作工具，支持文件读写、目录操作等功能
   参数: operation, path, content

2. http
   描述: HTTP 客户端工具，支持 GET、POST 等请求方法
   参数: url, method, headers, body

3. crawler
   描述: 网页爬虫工具，支持网页内容抓取和解析
   参数: url, selector, wait_time

🧪 直接工具调用演示
==================

📁 文件系统工具演示:
  ✅ 写文件成功: true
  ✅ 读文件成功，内容: Tool test at 2024-01-20 15:30:45
  ✅ 目录列表 (3 个文件):
    - tool_test.txt (file)
    - hello.txt (file)
    - traces (directory)

🌐 HTTP 工具演示:
  ✅ HTTP 请求成功
    状态码: 200
    内容类型: application/json
    响应体: {"slideshow": {"author": "Yours Truly", "date": "date of publication"...

💡 提示：设置 API Key 后可以看到 Agent 智能选择和使用工具的完整过程

📊 工具使用总结
===============
🔧 可用工具数量: 3
✅ 成功注册的工具: [fs http crawler]

💡 工具使用最佳实践:
1. 根据需求选择合适的工具
2. 注意工具的依赖服务（如 Redis、MySQL）
3. 合理设置工具的访问权限和路径限制
4. 使用 Agent 让 LLM 智能选择工具
5. 定期保存和分析执行轨迹

🎉 工具使用示例完成！
```

### 有 API Key 模式
设置 API Key 后，还会看到 Agent 智能使用工具的演示：

```
🤖 Agent 工具使用演示
=====================

📋 Agent 任务 1: 检查 workspace 目录下有哪些文件
------------------------------------
🤔 Agent 思考: 我需要使用文件系统工具来列出 workspace 目录的内容
🔧 工具调用: fs(operation="list", path="workspace")
📊 工具结果: 找到 5 个文件和目录
✅ 任务完成: workspace 目录包含以下文件：tool_test.txt、hello.txt、traces 目录等

📋 Agent 任务 2: 获取 https://httpbin.org/ip 的响应内容
------------------------------------
🤔 Agent 思考: 我需要使用 HTTP 工具来获取指定 URL 的内容
🔧 工具调用: http(url="https://httpbin.org/ip", method="GET")
📊 工具结果: HTTP 200，获取到 IP 信息
✅ 任务完成: 您的 IP 地址是 xxx.xxx.xxx.xxx

📋 Agent 任务 3: 创建一个名为 agent_test.txt 的文件，写入当前时间
------------------------------------
🤔 Agent 思考: 我需要使用文件系统工具来创建文件并写入当前时间
🔧 工具调用: fs(operation="write", path="workspace/agent_test.txt", content="2024-01-20 15:35:22")
📊 工具结果: 文件创建成功
✅ 任务完成: 已创建 agent_test.txt 文件，内容为当前时间
📝 执行轨迹已保存
```

## 🔧 工具配置说明

### 文件系统工具
```go
fsTool := builtin.NewFileSystemTool(
    []string{"./workspace", "./examples"},  // 允许访问的路径
    []string{"/etc", "/sys"},               // 禁止访问的路径
)
```

### HTTP 工具
```go
httpTool := builtin.NewHTTPTool()
// 默认配置，支持所有标准 HTTP 方法
```

### Redis 工具
```go
redisTool := builtin.NewRedisTool(
    "localhost:6379",  // Redis 地址
    "",                // 密码（可选）
    0,                 // 数据库编号
)
```

### MySQL 工具
```go
mysqlTool := builtin.NewMySQLTool(
    "user:password@tcp(localhost:3306)/database"  // 连接字符串
)
```

## 🔍 工具调用方式

### 直接调用
```go
result, err := tool.Invoke(ctx, map[string]any{
    "operation": "read",
    "path":      "workspace/test.txt",
})
```

### Agent 智能调用
```go
result, err := agent.Loop(ctx, "读取 workspace/test.txt 文件的内容")
```

## 🐛 故障排除

**Q: 浏览器工具注册失败**
A: 安装 Chrome 或 Chromium 浏览器，确保可执行文件在 PATH 中。

**Q: Redis/MySQL 工具注册失败**
A: 启动相应的服务，或者注释掉相关工具的注册代码。

**Q: HTTP 请求失败**
A: 检查网络连接，某些环境可能需要代理设置。

**Q: 文件系统操作权限错误**
A: 检查目录权限，确保程序有读写权限。

## 🔧 自定义工具配置

可以通过配置文件自定义工具行为：

```toml
[tools.filesystem]
allowed_paths = ["./workspace", "./data"]
blocked_paths = ["/etc", "/sys"]
max_file_size = "10MB"

[tools.http]
timeout = 30
max_redirects = 5
blocked_domains = ["localhost", "127.0.0.1"]

[tools.redis]
host = "localhost"
port = 6379
password = ""
database = 0
```

## 📚 相关文档

- [配置管理示例](../03-configuration/README.md)
- [MCP 集成示例](../../mcp/README.md)
- [工具开发指南](../../../docs/TOOLS.md)
- [API 参考文档](../../../docs/API.md)

---

这个示例展示了 OpenManus-Go 丰富的工具生态系统。继续学习其他示例来掌握更多高级功能！
