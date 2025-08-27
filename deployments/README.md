# OpenManus-Go 部署指南

本目录包含 OpenManus-Go 的各种部署配置和文档。

## 📁 目录结构

```
deployments/
├── README.md                    # 部署指南（本文件）
├── docker/
│   └── Dockerfile              # 生产环境 Docker 镜像
├── docker-compose.yaml         # 容器编排配置
├── init.sql                    # 数据库初始化脚本
├── nginx/
│   └── nginx.conf              # Nginx 反向代理配置
├── prometheus/
│   ├── prometheus.yml          # Prometheus 监控配置
│   └── rules/
│       └── openmanus.yml       # 告警规则
└── grafana/
    ├── datasources/
    │   └── datasources.yml     # 数据源配置
    └── dashboards/
        └── dashboard.yml       # 仪表板配置
```

## 🚀 快速部署

### 1. 基础部署

最简单的部署方式，包含核心服务：

```bash
# 设置环境变量
export OPENMANUS_LLM_API_KEY="your-api-key"
export OPENMANUS_LLM_MODEL="deepseek-chat"

# 启动基础服务
docker-compose up -d
```

启动的服务：
- **openmanus**: 主应用 (端口 8080)
- **redis**: 状态存储 (端口 6379)  
- **mysql**: 数据库 (端口 3306)

### 2. 完整部署

包含所有服务和监控：

```bash
# 启动完整服务栈
docker-compose --profile full --profile monitoring --profile proxy up -d
```

额外启动的服务：
- **elasticsearch**: 搜索引擎 (端口 9200)
- **minio**: 对象存储 (端口 9000/9001)
- **grafana**: 监控面板 (端口 3000)
- **prometheus**: 指标收集 (端口 9090)
- **jaeger**: 分布式追踪 (端口 16686)
- **nginx**: 反向代理 (端口 80/443)

## ⚙️ 环境变量配置

### 必需环境变量

```bash
# LLM 配置
export OPENMANUS_LLM_API_KEY="your-api-key"

# 可选配置
export OPENMANUS_LLM_BASE_URL="https://api.deepseek.com/v1"
export OPENMANUS_LLM_MODEL="deepseek-chat"
export OPENMANUS_LLM_TEMPERATURE="0.1"
export OPENMANUS_LLM_MAX_TOKENS="4000"
```

### 数据库配置

```bash
# MySQL
export MYSQL_ROOT_PASSWORD="your-root-password"
export MYSQL_PASSWORD="your-app-password"

# Redis（通常不需要密码）
export REDIS_PASSWORD=""  # 可选
```

### 监控配置

```bash
# Grafana
export GRAFANA_ADMIN_USER="admin"
export GRAFANA_ADMIN_PASSWORD="your-admin-password"

# MinIO
export MINIO_ROOT_USER="minioadmin"
export MINIO_ROOT_PASSWORD="your-minio-password"
```

## 🐳 Docker 镜像

### 构建自定义镜像

```bash
# 构建生产镜像
make docker-build

# 指定版本构建
VERSION=v1.0.0 make docker-build

# 推送到注册表
make docker-push
```

### 镜像特性

- **多阶段构建**: 优化镜像大小
- **非 root 用户**: 增强安全性
- **健康检查**: 自动健康监控
- **Alpine 基础**: 最小化攻击面

## 📊 监控和可观测性

### Prometheus 指标

系统自动暴露以下指标：

```
# Agent 相关指标
openmanus_agent_executions_total
openmanus_agent_duration_seconds
openmanus_agent_errors_total
openmanus_task_queue_length

# MCP 相关指标  
openmanus_mcp_server_up
openmanus_mcp_tool_calls_total
openmanus_mcp_tool_duration_seconds
openmanus_mcp_tool_errors_total

# 系统指标
process_resident_memory_bytes
process_cpu_seconds_total
go_gc_duration_seconds
```

### Grafana 仪表板

预配置的仪表板包含：

1. **系统概览**: 整体性能指标
2. **Agent 性能**: 任务执行详情
3. **MCP 监控**: MCP 服务器和工具状态
4. **基础设施**: Redis、MySQL 状态
5. **错误追踪**: 错误率和异常分析

访问地址: `http://localhost:3000` (admin/admin)

### 日志管理

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f openmanus
docker-compose logs -f redis
docker-compose logs -f mysql
```

## 🔧 高级配置

### 数据持久化

所有重要数据都通过 Docker 卷持久化：

```yaml
volumes:
  redis_data:     # Redis 数据
  mysql_data:     # MySQL 数据
  es_data:        # Elasticsearch 数据（可选）
  minio_data:     # MinIO 数据（可选）
  grafana_data:   # Grafana 配置
  prometheus_data: # Prometheus 数据
```

### 备份策略

```bash
# 数据库备份
docker-compose exec mysql mysqldump -u root -p openmanus > backup.sql

# Redis 备份
docker-compose exec redis redis-cli BGSAVE

# 配置备份
tar -czf config-backup.tar.gz deployments/
```

### 扩展配置

#### 负载均衡

修改 `docker-compose.yaml` 添加多个 openmanus 实例：

```yaml
services:
  openmanus-1:
    # ... 配置
  openmanus-2:  
    # ... 配置
    
  nginx:
    # 更新 upstream 配置
```

#### SSL/TLS 配置

1. 生成证书：
```bash
mkdir -p deployments/nginx/ssl
# 生成或复制 SSL 证书到该目录
```

2. 更新 nginx 配置启用 HTTPS

#### 外部数据库

使用外部数据库时，修改环境变量：

```bash
export OPENMANUS_MYSQL_DSN="user:password@tcp(external-mysql:3306)/openmanus"
export OPENMANUS_REDIS_ADDR="external-redis:6379"
```

## 🔐 安全配置

### 网络安全

- 所有服务运行在隔离的 Docker 网络中
- 使用非 root 用户运行应用
- 端口仅在需要时暴露

### 数据安全

```bash
# 设置强密码
export MYSQL_ROOT_PASSWORD="$(openssl rand -base64 32)"
export MYSQL_PASSWORD="$(openssl rand -base64 32)"
export GRAFANA_ADMIN_PASSWORD="$(openssl rand -base64 32)"

# 启用数据库 SSL（生产环境推荐）
# 配置 Redis AUTH
# 配置 API 认证
```

### 文件权限

```bash
# 设置配置文件权限
chmod 600 configs/config.toml
chmod 600 deployments/mysql/conf.d/*
chmod 600 deployments/nginx/ssl/*
```

## 🚀 生产环境部署

### 推荐配置

```bash
# 复制并创建生产配置
cp configs/config.example.toml configs/config.prod.toml

# 编辑生产配置，设置生产环境参数
vim configs/config.prod.toml

# 主要修改项：
# - 设置 LLM API key
# - 修改 host = "0.0.0.0" (容器环境)
# - 配置 Redis/MySQL 服务地址
# - 启用监控和安全选项
```

### 性能调优

1. **MySQL 调优**:
```sql
-- 在 init.sql 中添加
SET GLOBAL innodb_buffer_pool_size = 1G;
SET GLOBAL max_connections = 200;
```

2. **Redis 调优**:
```yaml
# docker-compose.yaml
command: >
  redis-server 
  --maxmemory 512mb
  --maxmemory-policy allkeys-lru
```

3. **应用调优**:
```bash
# 设置 Go 运行时参数
export GOMAXPROCS=4
export GOGC=100
```

### 健康检查

所有服务都配置了健康检查：

```bash
# 检查服务状态
docker-compose ps

# 手动健康检查
curl http://localhost:8080/health
curl http://localhost:9090/-/healthy
curl http://localhost:3000/api/health
```

## 📋 维护操作

### 日常维护

```bash
# 查看资源使用
docker stats

# 清理未使用的资源
docker system prune -f

# 更新服务
docker-compose pull
docker-compose up -d
```

### 故障排除

#### 常见问题

1. **服务启动失败**:
```bash
# 查看日志
docker-compose logs service-name

# 检查配置
docker-compose config
```

2. **数据库连接问题**:
```bash
# 测试 MySQL 连接
docker-compose exec mysql mysql -u openmanus -p

# 测试 Redis 连接
docker-compose exec redis redis-cli ping
```

3. **内存不足**:
```bash
# 增加交换空间
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
```

### 性能监控

```bash
# 实时监控
docker stats --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}"

# 导出指标
curl http://localhost:9090/api/v1/query?query=up

# 查看 Grafana 仪表板
open http://localhost:3000
```

## 📚 更多资源

- [配置指南](../configs/README.md)
- [API 文档](../docs/API.md)
- [架构设计](../docs/ARCHITECTURE.md)
- [故障排除](../docs/TROUBLESHOOTING.md)

---

如有问题，请查看 [GitHub Issues](https://github.com/your-org/openmanus-go/issues) 或联系维护团队。
