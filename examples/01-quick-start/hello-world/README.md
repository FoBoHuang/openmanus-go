# Hello World 示例

这是 OpenManus-Go 框架最基础的入门示例，帮助你了解框架的基本结构和使用方法。

## 🎯 学习目标

- 了解 OpenManus-Go 框架的基本组件
- 学会创建和配置 Agent
- 理解工具注册和使用流程
- 掌握简单任务的执行方法

## 📋 示例内容

本示例展示了：

1. **配置管理**：加载和验证配置文件
2. **组件创建**：LLM 客户端、工具注册表、Agent
3. **工具注册**：文件系统和 HTTP 工具
4. **任务执行**：文件操作和网络请求任务
5. **结果展示**：执行轨迹和工具信息

## 🚀 运行示例

### 前置要求

- Go 1.21+
- (可选) 有效的 LLM API Key

### 运行步骤

```bash
# 1. 进入示例目录
cd examples/01-quick-start/hello-world

# 2. 运行示例
go run main.go

# 或者使用构建好的二进制
../../../bin/openmanus run "创建一个测试文件"
```

### 配置 API Key (可选)

如果你有 LLM API Key，可以获得完整体验：

```bash
# 1. 复制配置模板
cp ../../../configs/config.example.toml ../../../configs/config.toml

# 2. 编辑配置文件
vim ../../../configs/config.toml

# 3. 设置 API Key
[llm]
api_key = "your-actual-api-key"
```

## 📊 示例输出

### 模拟模式（无 API Key）
```
🚀 OpenManus-Go Hello World 示例
==============================

📁 加载配置文件: ../../../configs/config.toml
⚠️  未设置 LLM API Key
📝 继续演示框架结构（模拟模式）...

🤖 创建 LLM 客户端...
✅ LLM 客户端已创建 (模型: deepseek-chat)

🔧 注册基础工具...
  ✅ 文件系统工具 (fs)
  ✅ HTTP 工具 (http)
📊 共注册 2 个工具

🧠 创建 Agent...
✅ Agent 已创建

📋 任务 1: 在 workspace 目录创建一个名为 hello.txt 的文件
🔄 模拟执行中...
💭 Agent 分析任务...
🔧 选择工具: fs (文件系统)
📝 执行操作: 写入文件
✅ 模拟结果: 文件创建成功
```

### 完整模式（有 API Key）
```
🚀 OpenManus-Go Hello World 示例
==============================

📁 加载配置文件: ../../../configs/config.toml
✅ 配置加载成功

🔄 正在执行...
🤔 [STEP 1/5] Planning next action...
⚡ [EXEC] Executing fs...
✅ [RESULT] fs completed: file created successfully
🏁 [DONE] Execution completed!
✅ 执行结果: 已成功创建 hello.txt 文件
```

## 🔧 技术要点

### 1. 配置管理
```go
// 加载配置文件
cfg, err := config.LoadFromFile(configPath)
if err != nil {
    cfg = config.DefaultConfig() // 使用默认配置
}
```

### 2. 组件创建
```go
// 创建 LLM 客户端
llmClient := llm.NewOpenAIClient(cfg.ToLLMConfig())

// 创建工具注册表
toolRegistry := tool.NewRegistry()

// 创建 Agent
agent := agent.NewBaseAgent(llmClient, toolRegistry, agentConfig)
```

### 3. 工具注册
```go
// 注册文件系统工具
fsTool := builtin.NewFileSystemTool(
    []string{"../../../workspace"}, // 允许路径
    []string{},                      // 禁止路径
)
toolRegistry.Register(fsTool)
```

### 4. 任务执行
```go
// 执行任务
result, err := agent.Loop(ctx, "你的任务描述")
```

## 📂 相关文件

执行后在 `workspace` 目录会生成：
- `hello.txt` - 测试文件
- `traces/` - 执行轨迹记录（如果配置了持久化）

## 🐛 故障排除

### 常见问题

1. **配置文件未找到**
   - 确保从正确的目录运行示例
   - 检查相对路径是否正确

2. **工具注册失败**
   - 检查目录权限
   - 确保 workspace 目录存在

3. **任务执行失败**
   - 检查 API Key 是否正确
   - 查看控制台错误信息
   - 检查网络连接

### 调试技巧

```bash
# 1. 启用详细日志
export LOG_LEVEL=debug
go run main.go

# 2. 检查配置
../../../bin/openmanus config validate

# 3. 测试工具
../../../bin/openmanus tools test
```

## 📚 下一步学习

- [基础任务示例](../basic-tasks/) - 学习更多任务类型
- [配置管理](../configuration/) - 深入了解配置选项
- [工具使用示例](../../02-tool-usage/) - 掌握各种工具

## 💡 扩展练习

1. 修改任务描述，尝试不同的文件操作
2. 添加更多工具到注册表
3. 调整 Agent 配置参数
4. 编写自己的简单任务

## 🤝 获取帮助

- 查看 [主文档](../../../README.md)
- 提交 [Issues](https://github.com/your-org/openmanus-go/issues)
- 参与 [讨论](https://github.com/your-org/openmanus-go/discussions)
