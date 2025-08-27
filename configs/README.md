# OpenManus-Go 配置指南

本目录包含 OpenManus-Go 的配置文件和模板。

## 📁 文件说明

```
configs/
├── README.md              # 配置指南（本文件）
├── config.example.toml    # 配置模板（包含所有选项和说明）
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

### 3. 运行测试

```bash
# 测试配置
./bin/openmanus run --config configs/config.toml "Hello, OpenManus!"
```

## ⚙️ 配置说明

### 🤖 LLM 配置

```toml
[llm]
model = "deepseek-chat"                 # 推荐模型
base_url = "https://api.deepseek.com/v1"
api_key = "sk-your-key-here"           # 🔑 必须设置
temperature = 0.1                       # 控制输出随机性
max_tokens = 4000                       # 最大生成长度
timeout = 60                            # 请求超时
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
```

### 💾 存储配置

#### 文件存储（默认）
```toml
[storage]
type = "file"
base_path = "./data/traces"
```

#### Redis 存储（推荐）
```toml
[storage]
type = "redis"

[storage.redis]
addr = "localhost:6379"
password = ""
db = 0
```

### 🛠️ 工具配置

#### HTTP 工具
```toml
[tools.http]
timeout = 45
blocked_domains = ["localhost", "127.0.0.1"]
```

#### 文件系统工具
```toml
[tools.filesystem]
allowed_paths = ["./workspace", "./data"]
blocked_paths = ["/etc", "/sys", "/proc"]
max_file_size = "100MB"
```

#### 浏览器工具
```toml
[tools.browser]
headless = true
timeout = 60
chrome_args = ["--no-sandbox", "--disable-dev-shm-usage"]
```

### 🔌 MCP 服务器配置

```toml
[[mcp_servers]]
name = "stock-helper"
transport = "sse"
url = "https://api.example.com/mcp"
timeout = 30
retry_count = 3

[[mcp_servers]]
name = "weather-service"
transport = "http"
url = "https://weather.example.com/mcp"
```

## 🏗️ 环境特定配置

### 开发环境

```toml
[logging]
level = "debug"
output = "console"

[storage]
type = "file"  # 简单的文件存储

[server]
host = "localhost"
port = 8080
```

### 生产环境

```toml
[logging]
level = "info"
output = "file"
format = "json"

[storage]
type = "redis"  # 使用 Redis 存储

[server]
host = "0.0.0.0"  # 监听所有接口
port = 8080

[monitoring]
enabled = true
metrics_port = 9090

[security]
enable_cors = true
cors_origins = ["https://yourdomain.com"]
```

### Docker 环境

```toml
[server]
host = "0.0.0.0"  # 容器中必须绑定所有接口

[storage.redis]
addr = "redis:6379"  # 使用服务名

[tools.database.mysql]
dsn = "user:pass@tcp(mysql:3306)/db"  # 使用服务名

[logging]
format = "json"  # 便于日志收集
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
  "169.254.169.254"  # AWS 元数据服务
]

[tools.filesystem]
blocked_paths = [
  "/etc", "/sys", "/proc", 
  "/root", "/var/log"
]
```

### 网络安全

```toml
[security]
enable_cors = true
cors_origins = ["https://trusted-domain.com"]
cors_methods = ["GET", "POST"]

[rate_limiting]
enabled = true
requests_per_minute = 60
```

## 🔧 配置验证

### 验证配置语法

```bash
# 检查 TOML 语法
toml-lint configs/config.toml

# 或者使用 Go 验证
./bin/openmanus config validate
```

### 测试连接

```bash
# 测试 LLM 连接
./bin/openmanus config test-llm

# 测试 Redis 连接
./bin/openmanus config test-redis

# 测试所有工具
./bin/openmanus tools test
```

## 📊 性能调优

### 高并发配置

```toml
[performance]
worker_count = 8          # CPU 核数的 2 倍
queue_size = 200
gc_percent = 100

[storage.redis]
pool_size = 20
max_retries = 3

[tools.database.mysql]
max_open_conns = 20
max_idle_conns = 10
```

### 内存优化

```toml
[agent]
max_tokens = 8000         # 减少内存使用

[logging]
level = "warn"            # 减少日志输出

[performance]
gc_percent = 50           # 更积极的垃圾回收
```

## 🐛 故障排除

### 常见问题

1. **API Key 错误**
   ```
   Error: unauthorized: invalid API key
   ```
   - 检查 `[llm] api_key` 设置
   - 验证 API key 是否有效

2. **连接超时**
   ```
   Error: context deadline exceeded
   ```
   - 增加 `[llm] timeout` 值
   - 检查网络连接

3. **权限拒绝**
   ```
   Error: permission denied
   ```
   - 检查 `[tools.filesystem] allowed_paths`
   - 确认文件权限

### 调试模式

```toml
[logging]
level = "debug"           # 启用详细日志
output = "both"           # 同时输出到控制台和文件

[agent]
max_steps = 3             # 限制步数便于调试
```

## 📚 配置参考

完整的配置选项请参考：
- [config.example.toml](config.example.toml) - 包含所有选项和详细说明
- [项目文档](../docs/) - 详细的架构和 API 文档
- [部署指南](../deployments/README.md) - 生产环境配置

---

如有配置问题，请查看 [GitHub Issues](https://github.com/your-org/openmanus-go/issues) 或参考项目文档。
