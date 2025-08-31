# 安装指南

本指南提供了 OpenManus-Go 的详细安装和配置说明，适用于不同的使用场景和环境。

## 📋 系统要求

### 基础要求
- **Go**: 1.21 或更高版本
- **操作系统**: Linux, macOS, Windows
- **内存**: 最少 512MB，推荐 2GB+
- **磁盘**: 最少 100MB 可用空间

### 可选组件
- **Redis**: 用于高性能状态存储 (推荐)
- **MySQL**: 用于数据库操作工具
- **Chrome/Chromium**: 用于浏览器自动化工具
- **Docker**: 用于容器化部署

## 🚀 安装方式

### 方式1: 从源码构建（推荐）

```bash
# 1. 克隆项目
git clone https://github.com/your-org/openmanus-go.git
cd openmanus-go

# 2. 检查 Go 版本
go version  # 确保 >= 1.21

# 3. 下载依赖
go mod download

# 4. 构建项目
make build

# 5. 验证安装
./bin/openmanus --version
```

### 方式2: 使用 Go install

```bash
# 直接安装最新版本
go install github.com/your-org/openmanus-go/cmd/openmanus@latest

# 验证安装
openmanus --version
```

### 方式3: 下载预构建二进制

```bash
# 下载适合您系统的二进制文件
# Linux AMD64
wget https://github.com/your-org/openmanus-go/releases/latest/download/openmanus-linux-amd64.tar.gz
tar -xzf openmanus-linux-amd64.tar.gz

# macOS AMD64  
wget https://github.com/your-org/openmanus-go/releases/latest/download/openmanus-darwin-amd64.tar.gz
tar -xzf openmanus-darwin-amd64.tar.gz

# macOS ARM64 (Apple Silicon)
wget https://github.com/your-org/openmanus-go/releases/latest/download/openmanus-darwin-arm64.tar.gz
tar -xzf openmanus-darwin-arm64.tar.gz

# Windows
# 下载 openmanus-windows-amd64.zip 并解压
```

### 方式4: Docker 安装

```bash
# 拉取镜像
docker pull ghcr.io/your-org/openmanus-go:latest

# 运行容器
docker run -it --rm \
  -v $(pwd)/workspace:/app/workspace \
  -v $(pwd)/configs:/app/configs \
  ghcr.io/your-org/openmanus-go:latest run --interactive
```

## ⚙️ 配置设置

### 1. 创建配置文件

```bash
# 复制配置模板
cp configs/config.example.toml configs/config.toml

# 编辑配置文件
vim configs/config.toml
```

### 2. 基础配置

**最小配置 (适合快速开始)**:
```toml
[llm]
model = "deepseek-chat"
base_url = "https://api.deepseek.com/v1"
api_key = "your-api-key-here"  # 🔑 必须设置
temperature = 0.1
max_tokens = 4000

[agent]
max_steps = 15
max_duration = "10m"
```

**推荐配置 (适合日常使用)**:
```toml
[llm]
model = "deepseek-chat"
base_url = "https://api.deepseek.com/v1"
api_key = "${OPENMANUS_API_KEY}"  # 使用环境变量
temperature = 0.1
max_tokens = 4000
timeout = 60

[agent]
max_steps = 20
max_tokens = 12000
max_duration = "15m"
reflection_steps = 3
max_retries = 3

[tools.filesystem]
allowed_paths = ["./workspace", "./data"]
max_file_size = "50MB"

[tools.http]
timeout = 30
blocked_domains = ["localhost", "127.0.0.1"]

[logging]
level = "info"
format = "text"
output = "console"
```

### 3. 生产环境配置

```toml
[llm]
model = "gpt-4"
base_url = "${LLM_BASE_URL}"
api_key = "${LLM_API_KEY}"
temperature = 0.0
max_tokens = 8000
timeout = 120

[agent]
max_steps = 30
max_tokens = 50000
max_duration = "30m"
reflection_steps = 5

[storage]
type = "redis"

[storage.redis]
addr = "${REDIS_URL}"
password = "${REDIS_PASSWORD}"
db = 0
max_retries = 3

[tools.database.mysql]
dsn = "${MYSQL_DSN}"
max_open_conns = 25
max_idle_conns = 10

[tools.database.redis]
addr = "${REDIS_URL}"
password = "${REDIS_PASSWORD}"
db = 1
pool_size = 20

[security]
enable_cors = true
cors_origins = ["https://your-domain.com"]

[monitoring]
enabled = true
metrics_port = 9090

[logging]
level = "info"
format = "json"
output = "file"
file_path = "/var/log/openmanus/app.log"
```

## 🗝️ LLM API 配置

### DeepSeek (推荐 - 性价比高)

```toml
[llm]
model = "deepseek-chat"
base_url = "https://api.deepseek.com/v1"
api_key = "sk-your-deepseek-key"
```

**获取方式**: [DeepSeek 官网](https://platform.deepseek.com/) 注册获取

### OpenAI

```toml
[llm]
model = "gpt-4"
base_url = "https://api.openai.com/v1"
api_key = "sk-your-openai-key"
```

### Azure OpenAI

```toml
[llm]
model = "gpt-4"
base_url = "https://your-resource.openai.azure.com/openai/deployments/gpt-4"
api_key = "your-azure-key"
```

### 本地模型 (Ollama)

```toml
[llm]
model = "llama2"
base_url = "http://localhost:11434/v1"
api_key = "dummy"  # Ollama 不需要真实 key
```

## 🗄️ 数据库配置

### Redis 配置 (推荐用于状态存储)

**安装 Redis**:
```bash
# Ubuntu/Debian
sudo apt-get install redis-server

# macOS
brew install redis

# 启动服务
redis-server

# Docker 方式
docker run -d --name redis -p 6379:6379 redis:alpine
```

**配置**:
```toml
[storage.redis]
addr = "localhost:6379"
password = ""
db = 0

[tools.database.redis]
addr = "localhost:6379"
password = ""
db = 1  # 使用不同的数据库
```

### MySQL 配置

**安装 MySQL**:
```bash
# Ubuntu/Debian
sudo apt-get install mysql-server

# macOS
brew install mysql

# Docker 方式
docker run -d --name mysql \
  -e MYSQL_ROOT_PASSWORD=password \
  -e MYSQL_DATABASE=openmanus \
  -p 3306:3306 mysql:8.0
```

**配置**:
```toml
[tools.database.mysql]
dsn = "user:password@tcp(localhost:3306)/openmanus?parseTime=true"
max_open_conns = 10
max_idle_conns = 5
```

## 🌐 浏览器配置

### Chrome/Chromium 安装

**Ubuntu/Debian**:
```bash
wget -q -O - https://dl.google.com/linux/linux_signing_key.pub | sudo apt-key add -
echo "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main" | sudo tee /etc/apt/sources.list.d/google-chrome.list
sudo apt-get update
sudo apt-get install google-chrome-stable
```

**macOS**:
```bash
brew install --cask google-chrome
```

**Docker 环境**:
```dockerfile
FROM golang:1.21-alpine AS builder
# ... 构建代码

FROM alpine:latest
RUN apk --no-cache add chromium
ENV CHROME_BIN=/usr/bin/chromium-browser
```

**配置**:
```toml
[tools.browser]
headless = true
timeout = 60
chrome_args = [
  "--no-sandbox",
  "--disable-dev-shm-usage",
  "--disable-gpu"
]
```

## 🐳 Docker 部署

### 1. 单容器部署

```bash
# 构建镜像
docker build -t openmanus-go .

# 运行容器
docker run -d --name openmanus \
  -p 8080:8080 \
  -v $(pwd)/configs:/app/configs \
  -v $(pwd)/workspace:/app/workspace \
  -e OPENMANUS_API_KEY=your-api-key \
  openmanus-go:latest run --interactive
```

### 2. Docker Compose 部署

创建 `docker-compose.yml`:
```yaml
version: '3.8'

services:
  openmanus:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./configs:/app/configs
      - ./workspace:/app/workspace
      - ./data:/app/data
    environment:
      - OPENMANUS_API_KEY=${OPENMANUS_API_KEY}
      - REDIS_URL=redis:6379
    depends_on:
      - redis
    command: ["./openmanus", "run", "--interactive"]

  redis:
    image: redis:alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  mysql:
    image: mysql:8.0
    ports:
      - "3306:3306"
    environment:
      - MYSQL_ROOT_PASSWORD=password
      - MYSQL_DATABASE=openmanus
    volumes:
      - mysql_data:/var/lib/mysql

volumes:
  redis_data:
  mysql_data:
```

启动服务:
```bash
# 设置环境变量
export OPENMANUS_API_KEY=your-api-key

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f openmanus
```

## ✅ 验证安装

### 1. 基础验证

```bash
# 检查版本
./bin/openmanus --version

# 检查帮助
./bin/openmanus --help

# 验证配置
./bin/openmanus config validate --config configs/config.toml
```

### 2. 功能验证

```bash
# 测试 LLM 连接
./bin/openmanus config test-llm --config configs/config.toml

# 查看可用工具
./bin/openmanus tools list --config configs/config.toml

# 测试工具功能
./bin/openmanus tools test --name fs --config configs/config.toml
```

### 3. 端到端测试

```bash
# 运行简单任务
./bin/openmanus run --config configs/config.toml "创建一个测试文件"

# 启动交互模式
./bin/openmanus run --config configs/config.toml --interactive
```

## 🔧 故障排除

### 常见问题

**1. Go 版本不兼容**
```bash
go version  # 检查版本
# 如果 < 1.21，请升级 Go
```

**2. 依赖下载失败**
```bash
# 设置代理
go env -w GOPROXY=https://goproxy.cn,direct
go mod download
```

**3. 权限错误**
```bash
# 确保二进制文件有执行权限
chmod +x bin/openmanus
```

**4. 配置文件错误**
```bash
# 验证 TOML 格式
./bin/openmanus config validate --config configs/config.toml
```

**5. API 连接失败**
```bash
# 检查网络连接
curl -H "Authorization: Bearer $API_KEY" https://api.deepseek.com/v1/models

# 检查配置
./bin/openmanus config test-llm --config configs/config.toml
```

### 调试模式

```bash
# 启用详细日志
./bin/openmanus run --config configs/config.toml --verbose --debug "your task"

# 查看配置信息
./bin/openmanus config show --config configs/config.toml
```

## 🚀 性能优化

### 配置优化

```toml
# 减少 token 使用
[llm]
max_tokens = 2000
temperature = 0.1

# 优化执行控制
[agent]
max_steps = 10
reflection_steps = 2

# 启用缓存
[storage]
type = "redis"
```

### 系统优化

```bash
# 设置 Go 环境变量
export GOGC=100
export GOMEMLIMIT=1GiB

# 限制并发
export GOMAXPROCS=4
```

## 📝 配置模板

### 开发环境
- 配置文件: `configs/config.example.toml` 
- 特点: 简单配置，快速启动

### 测试环境
- 包含完整的工具配置
- 启用详细日志
- 使用内存存储

### 生产环境
- 使用环境变量
- Redis 状态存储
- 完整的监控配置
- 安全设置

---

安装完成后，请查看 [快速入门指南](QUICK_START.md) 开始使用 OpenManus-Go！

**下一步推荐**: [快速入门](QUICK_START.md) → [核心概念](CONCEPTS.md) → [使用示例](EXAMPLES.md)
