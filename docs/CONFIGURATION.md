# 配置说明

本文档详细介绍 OpenManus-Go 的配置系统，包括所有配置选项、最佳实践和常见配置场景。

## 📋 目录

- [配置概述](#配置概述)
- [配置文件格式](#配置文件格式)
- [LLM 配置](#llm-配置)
- [Agent 配置](#agent-配置)
- [工具配置](#工具配置)
- [服务器配置](#服务器配置)
- [存储配置](#存储配置)
- [监控配置](#监控配置)
- [环境变量](#环境变量)
- [配置验证](#配置验证)

## 🔧 配置概述

OpenManus-Go 使用 TOML 格式的配置文件，支持：
- 分层配置结构
- 环境变量替换
- 配置验证
- 热重载（部分配置）
- 多环境配置

### 配置文件位置

配置文件按以下优先级查找：
1. 命令行指定的路径 (`--config`)
2. `./configs/config.toml`
3. `$HOME/.openmanus/config.toml`
4. `/etc/openmanus/config.toml`

### 配置加载方式

```bash
# 指定配置文件
./bin/openmanus run --config /path/to/config.toml

# 使用默认配置
./bin/openmanus run

# 从环境变量加载
export OPENMANUS_CONFIG_FILE="/path/to/config.toml"
./bin/openmanus run
```

## 📄 配置文件格式

### 完整配置示例

```toml
# OpenManus-Go 配置文件
# 版本: 1.0.0

[llm]
model = "deepseek-chat"
base_url = "https://api.deepseek.com/v1"
api_key = "${OPENMANUS_API_KEY}"
temperature = 0.1
max_tokens = 4000
timeout = "60s"

[agent]
max_steps = 15
max_tokens = 10000
max_duration = "10m"
reflection_steps = 3
max_retries = 3
retry_backoff = "2s"

[server]
host = "localhost"
port = 8080
read_timeout = "30s"
write_timeout = "30s"
idle_timeout = "60s"

[storage]
type = "file"
base_path = "./data/traces"

[storage.redis]
addr = "localhost:6379"
password = ""
db = 0
max_retries = 3
dial_timeout = "5s"
read_timeout = "3s"
write_timeout = "3s"

[logging]
level = "info"
output = "console"
format = "text"
file_path = "./logs/openmanus.log"

[monitoring]
enabled = false
metrics_port = 9090
prometheus_path = "/metrics"

# 工具配置
[tools.http]
timeout = "45s"
max_redirects = 5
user_agent = "OpenManus-Go/1.0"
allowed_domains = []
blocked_domains = ["localhost", "127.0.0.1"]

[tools.filesystem]
allowed_paths = ["./workspace", "./data"]
blocked_paths = ["/etc", "/sys", "/proc"]
max_file_size = "100MB"

[tools.browser]
headless = true
timeout = "60s"
user_agent = "OpenManus-Go/1.0"
chrome_args = ["--no-sandbox", "--disable-dev-shm-usage"]

[tools.database.mysql]
dsn = "user:password@tcp(localhost:3306)/database"
max_open_conns = 10
max_idle_conns = 5
conn_max_lifetime = "1h"

[tools.database.redis]
addr = "localhost:6379"
password = ""
db = 1
pool_size = 10

# MCP 服务器配置
[[mcp_servers]]
name = "example-service"
transport = "http"
url = "https://api.example.com/mcp"
timeout = "30s"
retry_count = 3

[security]
enable_cors = true
cors_origins = ["*"]
cors_methods = ["GET", "POST", "OPTIONS"]
```

## 🧠 LLM 配置

LLM 配置控制与大语言模型的交互。

```toml
[llm]
model = "deepseek-chat"                    # 模型名称
base_url = "https://api.deepseek.com/v1"   # API 端点
api_key = "${OPENMANUS_API_KEY}"           # API 密钥
temperature = 0.1                          # 生成温度 (0.0-1.0)
max_tokens = 4000                          # 最大 token 数
timeout = "60s"                            # 请求超时时间
```

### 支持的模型

#### DeepSeek (推荐)
```toml
[llm]
model = "deepseek-chat"
base_url = "https://api.deepseek.com/v1"
api_key = "sk-your-deepseek-key"
temperature = 0.1
max_tokens = 4000
```

#### OpenAI
```toml
[llm]
model = "gpt-4"
base_url = "https://api.openai.com/v1"
api_key = "sk-your-openai-key"
temperature = 0.0
max_tokens = 8000
```

#### Azure OpenAI
```toml
[llm]
model = "gpt-4"
base_url = "https://your-resource.openai.azure.com/openai/deployments/gpt-4"
api_key = "your-azure-key"
# 额外参数
api_version = "2024-02-15-preview"
```

#### 本地模型 (Ollama)
```toml
[llm]
model = "llama2"
base_url = "http://localhost:11434/v1"
api_key = "dummy"  # Ollama 不需要真实密钥
temperature = 0.2
max_tokens = 2000
```

### 参数说明

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `model` | string | "gpt-3.5-turbo" | 使用的模型名称 |
| `base_url` | string | - | API 端点 URL |
| `api_key` | string | - | API 密钥 (支持环境变量) |
| `temperature` | float | 0.1 | 生成随机性 (0.0-1.0) |
| `max_tokens` | int | 4000 | 单次请求最大 token 数 |
| `timeout` | duration | "60s" | 请求超时时间 |

## 🤖 Agent 配置

Agent 配置控制 AI Agent 的行为和执行策略。

```toml
[agent]
max_steps = 15                  # 最大执行步数
max_tokens = 10000              # 总 token 预算
max_duration = "10m"            # 最大执行时间
reflection_steps = 3            # 反思间隔步数
max_retries = 3                 # 最大重试次数
retry_backoff = "2s"            # 重试间隔时间
enable_memory = true            # 启用记忆功能
memory_window = 50              # 记忆窗口大小
```

### 预算控制

#### 步数预算
```toml
max_steps = 20  # 防止无限循环，建议 10-30
```

#### Token 预算
```toml
max_tokens = 15000  # 控制 LLM 成本，建议 5000-50000
```

#### 时间预算
```toml
max_duration = "15m"  # 防止长时间运行，建议 5m-30m
```

### 反思机制

```toml
reflection_steps = 3  # 每 3 步进行一次反思
```

反思有助于：
- 评估任务进度
- 调整执行策略
- 避免重复错误
- 优化后续步骤

### 错误处理

```toml
max_retries = 3         # 单个操作最大重试次数
retry_backoff = "2s"    # 重试间隔时间
```

## 🛠️ 工具配置

工具配置控制各个工具的行为和权限。

### HTTP 工具

```toml
[tools.http]
timeout = "45s"                               # 请求超时
max_redirects = 5                             # 最大重定向次数
user_agent = "OpenManus-Go/1.0"              # User-Agent
allowed_domains = ["api.example.com"]        # 允许的域名 (空=全部)
blocked_domains = ["localhost", "127.0.0.1"] # 禁止的域名
max_response_size = "10MB"                    # 最大响应大小
```

#### 安全配置
```toml
[tools.http]
# 生产环境建议限制域名
allowed_domains = [
    "api.github.com",
    "httpbin.org",
    "*.example.com"
]
blocked_domains = [
    "localhost",
    "127.0.0.1",
    "169.254.169.254",  # AWS 元数据服务
    "internal.company.com"
]
```

### 文件系统工具

```toml
[tools.filesystem]
allowed_paths = ["./workspace", "./data"]     # 允许访问的路径
blocked_paths = ["/etc", "/sys", "/proc"]     # 禁止访问的路径
max_file_size = "100MB"                       # 最大文件大小
enable_symlinks = false                       # 是否允许符号链接
```

#### 权限控制
```toml
[tools.filesystem]
# 开发环境
allowed_paths = ["./workspace", "./examples", "./data"]

# 生产环境
allowed_paths = ["/app/workspace", "/app/data"]
blocked_paths = ["/etc", "/sys", "/proc", "/root", "/var"]
```

### 浏览器工具

```toml
[tools.browser]
headless = true                               # 无头模式
timeout = "60s"                               # 页面加载超时
user_agent = "OpenManus-Go/1.0"              # User-Agent
chrome_args = [                              # Chrome 启动参数
    "--no-sandbox",
    "--disable-dev-shm-usage",
    "--disable-gpu"
]
```

#### Docker 环境配置
```toml
[tools.browser]
headless = true
chrome_args = [
    "--no-sandbox",
    "--disable-dev-shm-usage",
    "--disable-gpu",
    "--remote-debugging-port=9222",
    "--disable-web-security"
]
```

### 数据库工具

#### MySQL 配置
```toml
[tools.database.mysql]
dsn = "user:password@tcp(localhost:3306)/database?parseTime=true"
max_open_conns = 10                           # 最大连接数
max_idle_conns = 5                            # 最大空闲连接数
conn_max_lifetime = "1h"                      # 连接最大生存时间
```

#### Redis 配置
```toml
[tools.database.redis]
addr = "localhost:6379"                       # Redis 地址
password = ""                                 # Redis 密码
db = 1                                        # 数据库号
pool_size = 10                                # 连接池大小
```

## 🌐 服务器配置

服务器配置控制 HTTP 服务器和 MCP 服务器的行为。

```toml
[server]
host = "localhost"              # 监听地址
port = 8080                     # 监听端口
read_timeout = "30s"            # 读取超时
write_timeout = "30s"           # 写入超时
idle_timeout = "60s"            # 空闲超时
max_header_bytes = "1MB"        # 最大请求头大小
```

### 生产环境配置

```toml
[server]
host = "0.0.0.0"                # 监听所有接口
port = 8080
read_timeout = "30s"
write_timeout = "30s"
idle_timeout = "120s"
max_header_bytes = "1MB"

# 启用 TLS
tls_cert_file = "/etc/ssl/certs/server.crt"
tls_key_file = "/etc/ssl/private/server.key"
```

## 💾 存储配置

存储配置控制状态轨迹的持久化。

### 文件存储

```toml
[storage]
type = "file"
base_path = "./data/traces"     # 存储目录
max_files = 1000                # 最大文件数
compress = true                 # 启用压缩
```

### Redis 存储

```toml
[storage]
type = "redis"

[storage.redis]
addr = "localhost:6379"         # Redis 地址
password = ""                   # Redis 密码
db = 0                          # 数据库号
key_prefix = "openmanus:"       # 键前缀
max_retries = 3                 # 最大重试次数
dial_timeout = "5s"             # 连接超时
read_timeout = "3s"             # 读取超时
write_timeout = "3s"            # 写入超时
```

### S3 存储

```toml
[storage]
type = "s3"

[storage.s3]
region = "us-east-1"            # AWS 区域
bucket = "openmanus-traces"     # S3 存储桶
access_key = "${AWS_ACCESS_KEY_ID}"
secret_key = "${AWS_SECRET_ACCESS_KEY}"
endpoint = ""                   # 自定义端点 (可选)
```

## 📊 监控配置

监控配置控制指标收集和健康检查。

```toml
[monitoring]
enabled = true                  # 启用监控
metrics_port = 9090             # 指标端口
prometheus_path = "/metrics"    # Prometheus 路径
health_path = "/health"         # 健康检查路径
```

### 日志配置

```toml
[logging]
level = "info"                  # 日志级别: debug, info, warn, error
output = "console"              # 输出: console, file, both
format = "text"                 # 格式: text, json
file_path = "./logs/openmanus.log"  # 日志文件路径
max_size = "100MB"              # 最大文件大小
max_backups = 10                # 最大备份文件数
max_age = "30d"                 # 最大保留时间
compress = true                 # 压缩旧文件
```

#### 生产环境日志配置

```toml
[logging]
level = "info"
output = "file"
format = "json"
file_path = "/var/log/openmanus/app.log"
max_size = "100MB"
max_backups = 30
max_age = "90d"
compress = true
```

## 🌍 环境变量

OpenManus-Go 支持使用环境变量替换配置值。

### 语法

在配置文件中使用 `${VARIABLE_NAME}` 语法：

```toml
[llm]
api_key = "${OPENMANUS_API_KEY}"
base_url = "${LLM_BASE_URL:-https://api.openai.com/v1}"  # 带默认值
```

### 常用环境变量

```bash
# LLM 配置
export OPENMANUS_API_KEY="your-api-key"
export LLM_BASE_URL="https://api.deepseek.com/v1"
export LLM_MODEL="deepseek-chat"

# 服务器配置
export SERVER_HOST="0.0.0.0"
export SERVER_PORT="8080"

# 数据库配置
export REDIS_URL="redis://localhost:6379"
export REDIS_PASSWORD="your-redis-password"
export MYSQL_DSN="user:password@tcp(localhost:3306)/database"

# 存储配置
export STORAGE_TYPE="redis"
export STORAGE_BASE_PATH="/app/data"

# AWS 配置 (S3 存储)
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"
```

### Docker 环境变量

```bash
docker run -d \
  -e OPENMANUS_API_KEY="your-api-key" \
  -e REDIS_URL="redis:6379" \
  -e SERVER_HOST="0.0.0.0" \
  -p 8080:8080 \
  openmanus-go:latest
```

## ✅ 配置验证

### 验证命令

```bash
# 验证配置文件
./bin/openmanus config validate --config configs/config.toml

# 测试 LLM 连接
./bin/openmanus config test-llm --config configs/config.toml

# 测试工具可用性
./bin/openmanus config test-tools --config configs/config.toml

# 显示当前配置
./bin/openmanus config show --config configs/config.toml
```

### 验证规则

配置验证会检查：
- 必需字段是否存在
- 数据类型是否正确
- 数值范围是否有效
- 文件路径是否存在
- 网络连接是否正常

### 常见验证错误

**1. API Key 未设置**
```
错误: LLM API key is required
解决: 设置 api_key 字段或 OPENMANUS_API_KEY 环境变量
```

**2. 路径不存在**
```
错误: Storage base path does not exist: ./invalid/path
解决: 创建目录或修改 base_path 配置
```

**3. 端口被占用**
```
错误: Port 8080 is already in use
解决: 修改 port 配置或停止占用进程
```

## 📱 配置场景

### 开发环境

```toml
[llm]
model = "deepseek-chat"
api_key = "${OPENMANUS_API_KEY}"
temperature = 0.1

[agent]
max_steps = 10
max_duration = "5m"

[storage]
type = "file"
base_path = "./data"

[logging]
level = "debug"
format = "text"
output = "console"

[tools.filesystem]
allowed_paths = ["./workspace", "./examples"]
```

### 测试环境

```toml
[llm]
model = "gpt-3.5-turbo"
temperature = 0.0  # 确保结果一致

[agent]
max_steps = 5
max_duration = "2m"

[storage]
type = "memory"  # 测试后自动清理

[logging]
level = "warn"
output = "console"

[tools.http]
timeout = "10s"
blocked_domains = ["*"]  # 禁止所有网络访问
```

### 生产环境

```toml
[llm]
model = "gpt-4"
api_key = "${OPENMANUS_API_KEY}"
temperature = 0.0
timeout = "120s"

[agent]
max_steps = 30
max_tokens = 50000
max_duration = "30m"

[server]
host = "0.0.0.0"
port = 8080

[storage]
type = "redis"

[storage.redis]
addr = "${REDIS_URL}"
password = "${REDIS_PASSWORD}"

[monitoring]
enabled = true
metrics_port = 9090

[logging]
level = "info"
format = "json"
output = "file"
file_path = "/var/log/openmanus/app.log"

[security]
enable_cors = true
cors_origins = ["https://your-domain.com"]
```

## 🔧 高级配置

### 多环境配置

使用不同的配置文件：

```bash
# 开发环境
./bin/openmanus run --config configs/config.dev.toml

# 测试环境  
./bin/openmanus run --config configs/config.test.toml

# 生产环境
./bin/openmanus run --config configs/config.prod.toml
```

### 配置继承

```toml
# config.base.toml (基础配置)
[llm]
temperature = 0.1
max_tokens = 4000

[agent]
max_steps = 15

# config.prod.toml (生产环境，继承基础配置)
include = "config.base.toml"

[llm]
model = "gpt-4"  # 覆盖基础配置
api_key = "${PROD_API_KEY}"

[monitoring]
enabled = true  # 新增配置
```

### 动态配置

某些配置支持运行时修改：

```bash
# 动态修改日志级别
curl -X POST http://localhost:8080/admin/config \
  -d '{"logging.level": "debug"}'

# 动态修改 Agent 参数
curl -X POST http://localhost:8080/admin/config \
  -d '{"agent.max_steps": 20}'
```

---

通过合理的配置，您可以让 OpenManus-Go 在不同环境中发挥最佳性能！

**相关文档**: [安装指南](INSTALLATION.md) → [快速入门](QUICK_START.md) → [部署指南](DEPLOYMENT.md)
