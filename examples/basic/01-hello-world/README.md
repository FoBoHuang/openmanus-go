# Hello World 示例

这是 OpenManus-Go 框架的最简单使用示例，展示了如何创建和使用一个基础的 AI Agent。

## 🎯 学习目标

通过这个示例，你将学会：
- 如何创建 LLM 客户端
- 如何创建工具注册表
- 如何注册内置工具
- 如何创建和配置 Agent
- OpenManus-Go 框架的基本结构

## 📋 前置条件

1. **Go 环境**：Go 1.21+
2. **项目构建**：运行 `make build` 构建项目
3. **配置文件**：复制 `configs/config.example.toml` 到 `configs/config.toml`

## 🚀 运行示例

```bash
# 进入示例目录
cd examples/basic/01-hello-world

# 运行示例
go run main.go
```

## 📊 预期输出

```
🚀 OpenManus-Go Hello World Example
====================================

⚠️  警告：未设置 LLM API Key
请在 configs/config.toml 中设置正确的 api_key

示例配置：
[llm]
model = "deepseek-chat"
base_url = "https://api.deepseek.com/v1"
api_key = "your-actual-api-key"

📝 继续演示框架结构（不会进行实际的 LLM 调用）...

✅ LLM 客户端已创建
✅ 工具注册表已创建
✅ 文件系统工具已注册
✅ Agent 已创建

📋 任务 1: 创建一个名为 hello.txt 的文件，内容为 'Hello, OpenManus-Go!'
------------------------------------
🔄 模拟执行中...
💭 Agent 思考：需要使用文件系统工具
🔧 工具调用：fs(operation='write', path='workspace/hello.txt', content='Hello, OpenManus-Go!')
✅ 模拟结果：文件创建成功

📋 任务 2: 检查 hello.txt 文件是否存在
------------------------------------
🔄 模拟执行中...
💭 Agent 思考：需要使用文件系统工具
🔧 工具调用：fs(operation='exists', path='workspace/hello.txt')
✅ 模拟结果：文件存在

📊 框架信息总览
================
🔧 已注册工具数量: 1
⚙️  Agent 配置 - 最大步数: 3
🤖 LLM 模型: deepseek-chat

📋 可用工具列表:
  - fs: 文件系统操作工具，支持文件读写、目录操作等功能

🎉 Hello World 示例完成！

📚 下一步学习建议：
  1. 查看 ../02-tool-usage/ 学习工具使用
  2. 查看 ../03-configuration/ 学习配置管理
  3. 设置真实的 API Key 体验完整功能

💡 提示：运行 'make build' 构建完整项目
💡 提示：运行 './bin/openmanus run --help' 查看 CLI 帮助
```

## 🔧 代码结构解析

### 1. 配置加载
```go
cfg := config.DefaultConfig()
```
加载默认配置，实际使用中会从 `configs/config.toml` 加载。

### 2. LLM 客户端创建
```go
llmClient := llm.NewOpenAIClient(cfg.ToLLMConfig())
```
创建与大语言模型通信的客户端。

### 3. 工具注册
```go
toolRegistry := tool.NewRegistry()
fsTool := builtin.NewFileSystemTool([]string{"./workspace"}, []string{})
toolRegistry.Register(fsTool)
```
创建工具注册表并注册文件系统工具。

### 4. Agent 创建
```go
agentConfig := agent.DefaultConfig()
baseAgent := agent.NewBaseAgent(llmClient, toolRegistry, agentConfig)
```
创建 Agent 实例，这是执行任务的核心组件。

### 5. 任务执行
```go
result, err := baseAgent.Loop(ctx, task)
```
Agent 执行任务的主循环，包含规划、执行、观察、反思的完整流程。

## 🔑 设置真实 API Key

要体验完整功能，请在 `configs/config.toml` 中设置真实的 API Key：

```toml
[llm]
model = "deepseek-chat"
base_url = "https://api.deepseek.com/v1"
api_key = "sk-your-actual-api-key"
temperature = 0.1
max_tokens = 4000
```

## 🐛 故障排除

**Q: 运行时提示包导入错误**
A: 确保在项目根目录运行 `go mod tidy` 更新依赖。

**Q: 没有看到实际的 LLM 调用**
A: 这是正常的，没有设置 API Key 时程序会进入演示模式。

**Q: 想看到真实的执行效果**
A: 设置真实的 API Key 后重新运行示例。

## 📚 相关文档

- [工具使用示例](../02-tool-usage/README.md)
- [配置管理示例](../03-configuration/README.md)
- [项目架构文档](../../../docs/ARCHITECTURE.md)
- [工具开发指南](../../../docs/TOOLS.md)

---

这个示例展示了 OpenManus-Go 框架的基本使用方法。继续学习其他示例来掌握更多高级功能！
