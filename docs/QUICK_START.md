# 快速入门指南

本指南将在5分钟内帮助您上手 OpenManus-Go，完成第一个AI Agent任务。

## 🎯 学习目标

完成本指南后，您将能够：
- ✅ 安装和配置 OpenManus-Go
- ✅ 运行第一个AI Agent任务
- ✅ 理解基本的工作流程
- ✅ 知道如何进一步学习

## 🚀 快速开始

### 步骤1: 环境准备

```bash
# 1. 确认 Go 版本 (需要 1.21+)
go version

# 2. 克隆项目
git clone https://github.com/your-org/openmanus-go.git
cd openmanus-go

# 3. 安装依赖
go mod download

# 4. 构建项目
make build
# 或者使用 go build
go build -o bin/openmanus cmd/openmanus/main.go
```

### 步骤2: 配置设置

```bash
# 1. 复制配置模板
cp configs/config.example.toml configs/config.toml

# 2. 编辑配置文件，设置LLM API Key
vim configs/config.toml
```

**最小配置示例**：
```toml
[llm]
model = "deepseek-chat"
base_url = "https://api.deepseek.com/v1"
api_key = "your-api-key-here"  # 🔑 替换为您的API Key
temperature = 0.1
max_tokens = 4000

[agent]
max_steps = 15
max_duration = "10m"
```

### 步骤3: 运行第一个任务

```bash
# 交互模式（推荐）
./bin/openmanus run --config configs/config.toml --interactive

# 或单次任务
./bin/openmanus run --config configs/config.toml "创建一个包含当前时间的hello.txt文件"
```

## 🎮 交互模式演示

启动交互模式后，您会看到：

```
🤖 OpenManus-Go Interactive Mode
Type your goals and press Enter. Type 'quit' or 'exit' to stop.
Commands: /help, /status, /trace, /config

🎯 Goal: 
```

**试试这些任务**：

```bash
# 文件操作
🎯 Goal: 在workspace目录创建一个测试文件，内容为今天的日期

# 网络请求
🎯 Goal: 获取https://httpbin.org/json的内容并保存到data.json

# 数据处理
🎯 Goal: 列出当前目录的所有文件并生成一个清单
```

## 🔧 验证安装

运行以下命令验证安装：

```bash
# 检查版本
./bin/openmanus --version

# 验证配置
./bin/openmanus config validate --config configs/config.toml

# 查看可用工具
./bin/openmanus tools list --config configs/config.toml

# 测试LLM连接
./bin/openmanus config test-llm --config configs/config.toml
```

## 🛠️ 可用工具一览

OpenManus-Go 提供了6个内置工具：

| 工具 | 功能 | 示例任务 |
|------|------|----------|
| **fs** | 文件系统操作 | 创建文件、读取目录、文件操作 |
| **http** | HTTP请求 | API调用、数据获取、网络请求 |
| **crawler** | 网页爬虫 | 抓取网页内容、提取信息 |
| **browser** | 浏览器自动化 | 页面操作、截图、表单填写 |
| **redis** | Redis数据库 | 缓存操作、数据存储 |
| **mysql** | MySQL数据库 | 数据查询、存储操作 |

## 📝 示例任务

### 1. 文件操作任务
```bash
🎯 Goal: 创建一个项目报告文件，包含当前目录的文件统计信息
```

### 2. 网络数据任务
```bash
🎯 Goal: 从GitHub API获取某个用户的信息并保存为JSON文件
```

### 3. 数据分析任务
```bash
🎯 Goal: 分析workspace中的文本文件，统计总字数和行数
```

## 🎯 下一步学习

### 如果您想：

**🔍 深入了解架构**
→ 阅读 [架构设计](ARCHITECTURE.md) 和 [核心概念](CONCEPTS.md)

**🛠️ 开发自定义工具**
→ 查看 [工具开发指南](TOOLS.md)

**🔌 集成外部服务**
→ 学习 [MCP集成](MCP_INTEGRATION.md)

**🚀 部署到生产**
→ 参考 [部署指南](DEPLOYMENT.md)

**📖 查看更多示例**
→ 浏览 [使用示例](EXAMPLES.md) 和 [examples目录](../examples/)

## ❓ 常见问题

### Q: 启动时提示"goal is required"
**A**: 确保使用了 `--interactive` 参数：
```bash
./bin/openmanus run --config configs/config.toml --interactive
```

### Q: API调用失败
**A**: 检查配置文件中的API Key设置：
1. 确认API Key正确
2. 检查网络连接
3. 验证base_url是否正确

### Q: 工具执行权限错误
**A**: 检查配置文件中的路径设置：
```toml
[tools.filesystem]
allowed_paths = ["./workspace", "./data"]
```

### Q: 响应速度慢
**A**: 优化LLM配置：
```toml
[llm]
max_tokens = 2000        # 减少token数量
temperature = 0.1        # 降低随机性
```

## 🚨 故障排除

如果遇到问题：

1. **检查日志输出** - 启用详细模式：
   ```bash
   ./bin/openmanus run --config configs/config.toml --verbose --debug
   ```

2. **验证配置** - 使用配置验证：
   ```bash
   ./bin/openmanus config validate --config configs/config.toml
   ```

3. **查看帮助** - 获取详细帮助信息：
   ```bash
   ./bin/openmanus --help
   ./bin/openmanus run --help
   ```

4. **参考文档** - 查看 [故障排除指南](TROUBLESHOOTING.md)

## 🎉 成功！

恭喜您完成了 OpenManus-Go 的快速入门！现在您已经：

✅ 成功运行了第一个AI Agent任务  
✅ 了解了基本的工作流程  
✅ 掌握了配置和使用方法  

继续探索更多功能，开始您的AI Agent开发之旅吧！

---

**下一步推荐**：[基础概念](CONCEPTS.md) → [使用示例](EXAMPLES.md) → [架构设计](ARCHITECTURE.md)
