# Single Agent Example

这个示例演示了如何使用 OpenManus-Go 创建和运行单个 Agent 来执行简单任务。

## 功能演示

- 创建和配置 Agent
- 注册基础工具（HTTP、文件系统）
- 执行多个任务目标
- 保存执行轨迹

## 运行示例

1. 确保已设置 API Key：
```bash
export OPENMANUS_LLM_API_KEY="your-api-key-here"
```

2. 运行示例：
```bash
cd examples/single_agent
go run main.go
```

## 示例任务

1. **文件创建**：创建一个包含问候语的文本文件
2. **文件读取**：读取刚才创建的文件内容
3. **目录列表**：列出当前目录下的所有文件

## 预期输出

```
🤖 OpenManus-Go Single Agent Example
=====================================

📋 Task 1: 创建一个名为 hello.txt 的文件，内容为 'Hello, OpenManus-Go!'
--------------------------------------------------
✅ Result: Successfully created hello.txt file with the specified content

📋 Task 2: 读取刚才创建的 hello.txt 文件内容  
--------------------------------------------------
✅ Result: File content: Hello, OpenManus-Go!

📋 Task 3: 列出当前目录下的所有文件
--------------------------------------------------
✅ Result: Found 3 files: main.go, README.md, hello.txt

🎉 All tasks completed!
```

## 关键概念

### Agent 配置
```go
agentConfig := agent.DefaultConfig()
agentConfig.MaxSteps = 5  // 限制最大步数
```

### 工具注册
```go
toolRegistry := tool.NewRegistry()
httpTool := builtin.NewHTTPTool()
toolRegistry.Register(httpTool)
```

### 任务执行
```go
result, err := baseAgent.Loop(ctx, goal)
```

### 轨迹保存
```go
trace := baseAgent.GetTrace()
store.Save(trace)
```

## 自定义扩展

你可以通过以下方式扩展这个示例：

1. **添加更多工具**：注册 Redis、MySQL、Browser 等工具
2. **复杂任务**：尝试更复杂的多步骤任务
3. **错误处理**：添加更完善的错误处理和重试逻辑
4. **配置文件**：使用配置文件而不是硬编码配置

## 故障排除

- **API Key 错误**：确保设置了有效的 OpenAI API Key
- **权限错误**：确保有文件系统写入权限
- **网络错误**：检查网络连接和防火墙设置
