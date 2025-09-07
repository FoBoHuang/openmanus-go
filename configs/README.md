# OpenManus-Go 配置指南

本目录包含 OpenManus-Go 的配置文件和模板。

## 📁 文件说明

```
configs/
├── README.md              # 配置指南（本文件）
├── config.example.toml    # 配置模板（包含所有选项和说明）
├── config.prod.toml       # 生产环境配置模板
└── config.toml           # 实际配置文件（需要手动创建）
```

## 🚀 快速开始

### 1. 创建配置文件

```bash
# 复制模板
cp configs/config.example.toml configs/config.toml

# 编辑配置
vim configs/config.toml
```

### 2. 必要设置

最少需要设置以下配置项：

```toml
[llm]
api_key = "your-actual-api-key"  # 🔑 必须设置！
```

### 3. 验证配置

```bash
# 验证配置语法
./bin/openmanus config validate --config configs/config.toml

# 显示当前配置
./bin/openmanus config show --config configs/config.toml
```

### 4. 运行测试

```bash
# 测试运行
./bin/openmanus run --config configs/config.toml "Hello, OpenManus!"
```

## ⚙️ 配置说明

### 🤖 LLM 配置

```toml
[llm]
model = "deepseek-chat"                 # 推荐模型
base_url = "https://api.deepseek.com/v1"
api_key = "sk-your-key-here"           # 🔑 必须设置
temperature = 0.1                       # 控制输出随机性 (0.0-1.0)
max_tokens = 4000                       # 最大生成长度
timeout = 60                            # 请求超时时间（秒）
```

**支持的模型提供商：**
- **DeepSeek**: `deepseek-chat`, `deepseek-coder`
- **OpenAI**: `gpt-3.5-turbo`, `gpt-4`, `gpt-4-turbo`
- **Anthropic**: `claude-3-sonnet`, `claude-3-haiku`
- **其他兼容 OpenAI API 的服务**

### 🤖 Agent 配置

```toml
[agent]
max_steps = 15                          # 最大执行步数
max_tokens = 10000                      # 令牌预算
max_duration = "10m"                    # 超时时间
reflection_steps = 3                    # 反思频率
max_retries = 3                         # 重试次数
retry_backoff = "2s"                    # 重试间隔
```

### 🔄 运行流程配置

```toml
[runflow]
use_data_analysis_agent = false         # 是否使用数据分析智能体
enable_multi_agent = false              # 是否启用多智能体协作
```

### 🌐 服务器配置

```toml
[server]
host = "localhost"                      # 监听地址，Docker 环境使用 "0.0.0.0"
port = 8080                             # 监听端口
```

### 💾 存储配置

#### 文件存储（默认）
```toml
[storage]
type = "file"
base_path = "./data/traces"
```

#### Redis 存储（推荐生产环境）
```toml
[storage]
type = "redis"

[storage.redis]
addr = "localhost:6379"
password = ""
db = 0
```

#### S3 存储（云端归档）
```toml
[storage]
type = "s3"

[storage.s3]
region = "us-east-1"
bucket = "openmanus-traces"
access_key = ""                         # 建议使用环境变量
secret_key = ""                         # 建议使用环境变量
```

### 📝 日志配置

```toml
[logging]
level = "info"                          # debug | info | warn | error
output = "console"                      # console | file | both
file_path = "./log/openmanus.log"       # 日志文件路径
```

### 🛠️ 工具配置

#### HTTP 工具
```toml
[tools.http]
timeout = 45                            # HTTP 请求超时时间（秒）
allowed_domains = []                    # 允许的域名，空数组表示允许所有
blocked_domains = ["localhost", "127.0.0.1", "169.254.169.254"]
```

#### 文件系统工具
```toml
[tools.filesystem]
allowed_paths = ["./workspace", "./data", "./examples"]
blocked_paths = ["/etc", "/sys", "/proc", "/root", "/var"]
```

#### 浏览器工具
```toml
[tools.browser]
headless = true                         # 无头浏览器模式
timeout = 60                            # 浏览器操作超时时间（秒）
user_agent = "OpenManus-Go/1.0"
```

#### 数据库工具

**MySQL**
```toml
[tools.database.mysql]
dsn = ""                                # MySQL 连接字符串，空字符串禁用
```

**Redis**
```toml
[tools.database.redis]
addr = "localhost:6379"
password = ""
db = 1
```

**Elasticsearch**
```toml
[tools.database.elasticsearch]
addresses = []                          # ES 地址列表，空数组禁用
username = ""
password = ""
```

### 🔌 MCP 服务器配置

```toml
[mcp.servers]

# Higress 股票助手
[mcp.servers.mcp-stock-helper]
url = "https://mcp.higress.ai/mcp-stock-helper/your-session-id"

# Higress 日历节假日助手
[mcp.servers.mcp-calendar-holiday-helper]
url = "https://mcp.higress.ai/mcp-calendar-holiday-helper/your-session-id"

# 自定义 MCP 服务器请求头（如需要）
# [mcp.servers.mcp-stock-helper.headers]
# Authorization = "Bearer <TOKEN>"
# X-API-Key = "<KEY>"
```

## 🏗️ 环境特定配置

### 开发环境

```toml
[logging]
level = "debug"                         # 详细日志
output = "console"

[storage]
type = "file"                           # 简单文件存储

[server]
host = "localhost"
port = 8080
```

### 生产环境

建议复制 `config.prod.toml` 模板：

```bash
cp configs/config.prod.toml configs/config.toml
```

生产环境特点：
- 使用环境变量管理敏感信息
- 启用 Redis 存储
- 更长的超时时间
- 更严格的安全设置

### Docker 环境

```toml
[server]
host = "0.0.0.0"                        # 容器中必须绑定所有接口

[storage.redis]
addr = "redis:6379"                     # 使用 Docker 服务名

[tools.database.mysql]
dsn = "user:pass@tcp(mysql:3306)/db"    # 使用 Docker 服务名

[logging]
output = "console"                      # 容器日志输出到控制台
```

## 🔒 安全配置

### API Key 管理

**推荐方法：环境变量**
```bash
export OPENMANUS_LLM_API_KEY="your-key"
```

然后在配置中引用：
```toml
[llm]
api_key = "${OPENMANUS_LLM_API_KEY}"
```

### 访问控制

```toml
[tools.http]
blocked_domains = [
  "localhost", 
  "127.0.0.1", 
  "169.254.169.254"                     # AWS 元数据服务
]

[tools.filesystem]
blocked_paths = [
  "/etc", "/sys", "/proc", 
  "/root", "/var", "/usr", "/boot"
]
```

## 🔧 配置验证

### 验证配置语法

```bash
# 验证配置文件
./bin/openmanus config validate --config configs/config.toml

# 显示当前配置
./bin/openmanus config show --config configs/config.toml

# 初始化默认配置
./bin/openmanus config init
```

### 测试配置

```bash
# 运行简单测试
./bin/openmanus run --config configs/config.toml "测试配置是否正常"

# 检查工具可用性
./bin/openmanus tools --help
```

## 📊 性能调优

### 高并发配置

```toml
[agent]
max_steps = 25                          # 增加执行步数
max_tokens = 15000                      # 更大的令牌预算

[storage.redis]
# 使用 Redis 提高并发性能
addr = "localhost:6379"
```

### 内存优化

```toml
[agent]
max_tokens = 8000                       # 减少内存使用

[logging]
level = "warn"                          # 减少日志输出
```

## 🐛 故障排除

### 常见问题

1. **API Key 错误**
   ```
   Error: llm.api_key is required
   ```
   - 检查 `[llm] api_key` 设置
   - 验证 API key 是否有效

2. **配置文件语法错误**
   ```
   Error: failed to read config file
   ```
   - 使用 `./bin/openmanus config validate` 检查语法
   - 确认 TOML 格式正确

3. **权限拒绝**
   ```
   Error: path not allowed
   ```
   - 检查 `[tools.filesystem] allowed_paths`
   - 确认文件路径权限

4. **连接超时**
   ```
   Error: context deadline exceeded
   ```
   - 增加 `[llm] timeout` 值
   - 检查网络连接

### 调试模式

```toml
[logging]
level = "debug"                         # 启用详细日志
output = "both"                         # 同时输出到控制台和文件

[agent]
max_steps = 5                           # 限制步数便于调试
```

## 📚 配置参考

### 配置文件模板
- [config.example.toml](config.example.toml) - 开发环境配置模板
- [config.prod.toml](config.prod.toml) - 生产环境配置模板

### 相关文档
- [项目文档](../docs/) - 详细的架构和 API 文档
- [部署指南](../deployments/README.md) - 生产环境部署
- [工具文档](../docs/TOOLS.md) - 工具系统详细说明
- [MCP 集成](../docs/MCP_INTEGRATION.md) - MCP 服务器集成指南

### 环境变量参考

所有配置项都支持环境变量，前缀为 `OPENMANUS_`：

```bash
# LLM 配置
export OPENMANUS_LLM_API_KEY="your-key"
export OPENMANUS_LLM_BASE_URL="https://api.openai.com/v1"

# Redis 配置
export OPENMANUS_STORAGE_REDIS_ADDR="redis:6379"
export OPENMANUS_STORAGE_REDIS_PASSWORD="your-password"

# 数据库配置
export OPENMANUS_TOOLS_DATABASE_MYSQL_DSN="user:pass@tcp(localhost:3306)/db"
```

---

如有配置问题，请查看 [GitHub Issues](https://github.com/OpenManus/openmanus-go/issues) 或参考项目文档。