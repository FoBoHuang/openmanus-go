# OpenManus-Go 示例

这个目录包含了 OpenManus-Go 框架的各种使用示例，帮助你快速上手并掌握框架的核心功能。

## 📁 示例目录结构

```
examples/
├── 01-quick-start/           # 快速入门示例
│   ├── hello-world/         # 最简单的 Hello World
│   ├── basic-tasks/         # 基础任务示例
│   └── configuration/       # 配置管理示例
├── 02-tool-usage/           # 工具使用示例
│   ├── filesystem/          # 文件系统工具
│   ├── network/             # 网络请求工具
│   ├── browser/             # 浏览器自动化
│   └── database/            # 数据库操作
├── 03-mcp-integration/      # MCP 集成示例
│   ├── mcp-server/          # MCP 服务器示例
│   ├── mcp-client/          # MCP 客户端示例
│   └── external-services/   # 外部服务集成
├── 04-real-world/           # 实际应用场景
│   ├── data-processing/     # 数据处理任务
│   ├── web-automation/      # 网页自动化
│   └── report-generation/   # 报告生成
└── scripts/                 # 辅助脚本
    ├── setup.sh             # 环境设置
    ├── run-all.sh           # 运行所有示例
    └── test-examples.sh     # 测试示例
```

## 🚀 快速开始

### 1. 环境准备

```bash
# 1. 构建项目
cd ../..  # 回到项目根目录
make build

# 2. 设置示例环境
cd examples
./scripts/setup.sh

# 3. 复制配置文件
cp ../configs/config.example.toml ../configs/config.toml
# 编辑 config.toml 设置你的 LLM API Key
```

### 2. 运行示例

```bash
# 运行 Hello World 示例
cd 01-quick-start/hello-world
go run main.go

# 或者使用构建好的二进制文件
../../bin/openmanus run "创建一个测试文件"

# 运行所有示例（自动测试）
./scripts/run-all.sh
```

## 📋 示例说明

### 01-quick-start - 快速入门

**适合人群**：首次使用 OpenManus-Go 的开发者

- **hello-world**：最基础的示例，展示框架基本结构
- **basic-tasks**：简单任务执行，展示 Agent 的工作流程  
- **configuration**：配置管理和验证

### 02-tool-usage - 工具使用

**适合人群**：需要了解工具系统的开发者

- **filesystem**：文件系统操作示例
- **network**：HTTP 请求和网页爬虫
- **browser**：浏览器自动化和页面操作
- **database**：Redis 和 MySQL 数据库操作

### 03-mcp-integration - MCP 集成

**适合人群**：需要集成外部服务的开发者

- **mcp-server**：创建 MCP 服务器
- **mcp-client**：连接外部 MCP 服务
- **external-services**：集成第三方 API 服务

### 04-real-world - 实际应用

**适合人群**：需要解决实际业务问题的开发者

- **data-processing**：数据清理、转换、分析
- **web-automation**：网页自动化、表单填写
- **report-generation**：自动化报告生成

## 🛠️ 示例运行要求

### 基础要求
- Go 1.21+
- 有效的 LLM API Key（推荐 DeepSeek）

### 可选要求
- Redis（用于数据库示例）
- MySQL（用于数据库示例）
- Chrome/Chromium（用于浏览器示例）
- Docker（用于容器化示例）

## 🧪 测试示例

```bash
# 测试所有示例
./scripts/test-examples.sh

# 测试特定类别
./scripts/test-examples.sh --category quick-start

# 测试特定示例
./scripts/test-examples.sh --example hello-world
```

## 📝 自定义示例

### 创建新示例

1. 选择合适的类别目录
2. 创建新的示例目录
3. 添加 `main.go` 和 `README.md`
4. 更新此文档

### 示例模板

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "openmanus-go/pkg/agent"
    "openmanus-go/pkg/config"
    "openmanus-go/pkg/llm"
    "openmanus-go/pkg/tool"
)

func main() {
    fmt.Println("🚀 示例名称")
    fmt.Println("=" + strings.Repeat("=", len("示例名称")+4))
    
    // 1. 加载配置
    cfg := config.LoadConfig("../../../configs/config.toml")
    
    // 2. 创建组件
    llmClient := llm.NewOpenAIClient(cfg.ToLLMConfig())
    toolRegistry := tool.DefaultRegistry
    
    // 3. 创建 Agent
    agent := agent.NewBaseAgent(llmClient, toolRegistry, nil)
    
    // 4. 执行任务
    result, err := agent.Loop(context.Background(), "你的任务描述")
    if err != nil {
        log.Fatalf("任务执行失败: %v", err)
    }
    
    fmt.Printf("✅ 任务完成: %s\n", result)
}
```

## 💡 使用建议

### 学习路径
1. 从 `01-quick-start/hello-world` 开始
2. 按顺序学习各个类别的示例
3. 运行 `scripts/run-all.sh` 查看完整演示
4. 根据需求定制和扩展示例

### 故障排除
- 检查 API Key 是否正确设置
- 确认必要的服务（Redis、MySQL）是否运行
- 查看日志输出获取详细错误信息
- 参考各示例的 README 文件

### 性能优化
- 合理设置 `max_steps` 避免过长执行
- 使用合适的 `temperature` 值
- 监控 token 使用量
- 启用执行轨迹分析

## 🤝 贡献示例

我们欢迎你贡献新的示例！

1. 确保示例有实际价值
2. 提供清晰的文档和注释
3. 包含必要的错误处理
4. 添加适当的测试

## 📞 获取帮助

- 查看 [主文档](../README.md)
- 提交 [GitHub Issues](https://github.com/your-org/openmanus-go/issues)
- 参与 [讨论区](https://github.com/your-org/openmanus-go/discussions)

---

祝你在 OpenManus-Go 的学习之旅中收获满满！🎉