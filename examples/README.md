# OpenManus-Go 示例程序

本目录包含了 OpenManus-Go 框架的完整示例程序，展示了从基础使用到高级功能的各种应用场景。

## 📁 目录结构

### 🚀 基础示例 (`basic/`)
适合初学者，展示框架的基本概念和使用方法：
- **01-hello-world/** - 最简单的 Agent 使用示例
- **02-tool-usage/** - 内置工具调用示例
- **03-configuration/** - 配置管理和环境设置示例

### 🔌 MCP 集成示例 (`mcp/`)
展示 Model Context Protocol 的完整集成：
- **01-mcp-server/** - MCP 服务器启动和管理
- **02-mcp-client/** - MCP 客户端连接和工具调用
- **03-mcp-integration/** - MCP 完整集成和工具发现

### 🤝 多 Agent 协作示例 (`multi-agent/`)
展示多 Agent 协作和工作流管理：
- **01-sequential/** - 顺序执行工作流
- **02-parallel/** - 并行执行工作流
- **03-dag-workflow/** - DAG 依赖关系工作流

### 🎯 实际应用场景 (`applications/`)
真实世界的应用案例：
- **01-data-analysis/** - 数据分析和报告生成
- **02-web-scraping/** - 网页内容抓取和处理
- **03-file-processing/** - 批量文件处理和转换
- **04-automation-task/** - 自动化任务执行

### 🔧 高级功能示例 (`advanced/`)
展示框架的高级特性：
- **01-custom-tools/** - 自定义工具开发
- **02-memory-management/** - 记忆管理和状态持久化
- **03-reflection/** - 反思机制和错误恢复

## 🚀 快速开始

### 1. 环境准备

```bash
# 确保已构建项目
make build

# 复制配置文件
cp configs/config.example.toml configs/config.toml

# 设置 API Key（重要！）
vim configs/config.toml
```

### 2. 运行基础示例

```bash
# 运行 Hello World 示例
cd examples/basic/01-hello-world
go run main.go

# 运行工具使用示例
cd examples/basic/02-tool-usage
go run main.go
```

### 3. 运行 MCP 示例

```bash
# 启动 MCP 服务器
cd examples/mcp/01-mcp-server
go run main.go

# 在另一个终端运行客户端
cd examples/mcp/02-mcp-client
go run main.go
```

### 4. 运行多 Agent 示例

```bash
# 运行顺序工作流
cd examples/multi-agent/01-sequential
go run main.go

# 运行并行工作流
cd examples/multi-agent/02-parallel
go run main.go
```

## 🔧 配置说明

所有示例程序都使用项目根目录的 `configs/config.toml` 配置文件。主要配置项：

```toml
[llm]
model = "deepseek-chat"
base_url = "https://api.deepseek.com/v1"
api_key = "your-api-key-here"  # 必须设置！
temperature = 0.1
max_tokens = 4000

[agent]
max_steps = 10
max_tokens = 8000
max_duration = "5m"
```

## 📋 示例程序特点

### ✨ 渐进式学习
- 从简单到复杂，逐步介绍功能
- 每个示例都有详细的注释和说明
- 独立运行，不相互依赖

### 🛠️ 实用性强
- 基于真实使用场景设计
- 可以直接修改用于实际项目
- 展示最佳实践和常见模式

### 📚 文档完整
- 每个示例都有独立的 README
- 详细的代码注释
- 运行结果示例和故障排除

## 🧪 运行所有示例

使用提供的脚本可以批量运行和测试示例：

```bash
# 运行所有示例程序
./scripts/run-all.sh

# 测试示例程序（不需要 LLM API）
./scripts/test-examples.sh

# 设置示例环境
./scripts/setup.sh
```

## 📝 示例程序清单

| 分类 | 示例名称 | 功能描述 | 难度 | 运行时间 |
|------|----------|----------|------|----------|
| 基础 | hello-world | 最简单的 Agent 使用 | ⭐ | < 30s |
| 基础 | tool-usage | 内置工具调用 | ⭐⭐ | < 1m |
| 基础 | configuration | 配置管理 | ⭐⭐ | < 30s |
| MCP | mcp-server | MCP 服务器 | ⭐⭐ | < 30s |
| MCP | mcp-client | MCP 客户端 | ⭐⭐ | < 1m |
| MCP | mcp-integration | MCP 完整集成 | ⭐⭐⭐ | < 2m |
| 多Agent | sequential | 顺序工作流 | ⭐⭐⭐ | < 2m |
| 多Agent | parallel | 并行工作流 | ⭐⭐⭐ | < 2m |
| 多Agent | dag-workflow | DAG 工作流 | ⭐⭐⭐⭐ | < 3m |
| 应用 | data-analysis | 数据分析 | ⭐⭐⭐ | < 2m |
| 应用 | web-scraping | 网页抓取 | ⭐⭐⭐ | < 3m |
| 应用 | file-processing | 文件处理 | ⭐⭐ | < 1m |
| 应用 | automation-task | 自动化任务 | ⭐⭐⭐⭐ | < 5m |
| 高级 | custom-tools | 自定义工具 | ⭐⭐⭐⭐ | < 2m |
| 高级 | memory-management | 记忆管理 | ⭐⭐⭐⭐ | < 2m |
| 高级 | reflection | 反思机制 | ⭐⭐⭐⭐⭐ | < 3m |

## 🐛 故障排除

### 常见问题

**Q: 示例运行时提示 "LLM request failed"**
A: 请检查 `configs/config.toml` 中的 `api_key` 是否正确设置。

**Q: MCP 服务器连接失败**
A: 确保 MCP 服务器已启动，并检查端口是否被占用。

**Q: 工具调用失败**
A: 检查网络连接，某些工具需要访问外部服务。

**Q: 多 Agent 示例运行缓慢**
A: 这是正常现象，多 Agent 协作需要更多的 LLM 调用。

### 获取帮助

- 查看具体示例的 README 文件
- 检查日志输出中的错误信息
- 访问项目文档：`docs/` 目录
- 提交 Issue：[GitHub Issues](https://github.com/your-org/openmanus-go/issues)

## 🤝 贡献示例

欢迎贡献新的示例程序！请遵循以下规范：

1. **目录结构**：按照现有分类创建目录
2. **代码规范**：遵循 Go 代码规范，添加详细注释
3. **文档完整**：包含 README.md 和运行说明
4. **测试验证**：确保示例可以正常运行
5. **实用性**：基于真实使用场景，具有学习价值

---

**OpenManus-Go Examples** - 通过实例学习，快速掌握 AI Agent 开发！ 🚀✨
