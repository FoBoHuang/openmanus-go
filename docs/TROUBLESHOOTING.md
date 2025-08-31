# 故障排除指南

本文档提供 OpenManus-Go 常见问题的诊断和解决方案，帮助您快速定位和修复问题。

## 📋 目录

- [常见启动问题](#常见启动问题)
- [配置相关问题](#配置相关问题)
- [工具调用问题](#工具调用问题)
- [性能问题](#性能问题)
- [网络连接问题](#网络连接问题)
- [多Agent协作问题](#多agent协作问题)
- [调试工具和技巧](#调试工具和技巧)

## 🚀 常见启动问题

### 问题1：程序无法启动

**症状**：
```bash
$ ./bin/openmanus run --config configs/config.toml
Error: failed to load config: no such file or directory
```

**诊断步骤**：
```bash
# 1. 检查文件是否存在
ls -la configs/config.toml

# 2. 检查文件权限
ls -la bin/openmanus

# 3. 验证配置文件格式
./bin/openmanus config validate --config configs/config.toml
```

**解决方案**：
```bash
# 1. 创建配置文件
cp configs/config.example.toml configs/config.toml

# 2. 设置执行权限
chmod +x bin/openmanus

# 3. 检查依赖
ldd bin/openmanus  # Linux
otool -L bin/openmanus  # macOS
```

### 问题2：权限错误

**症状**：
```bash
Error: permission denied: ./bin/openmanus
```

**解决方案**：
```bash
# 设置执行权限
chmod +x bin/openmanus

# 检查目录权限
chmod 755 workspace data logs

# 如果是容器环境，检查用户权限
id
whoami
```

### 问题3：端口被占用

**症状**：
```bash
Error: failed to start server: listen tcp :8080: bind: address already in use
```

**诊断和解决**：
```bash
# 1. 查找占用端口的进程
netstat -tulpn | grep 8080
lsof -i :8080

# 2. 停止占用进程
kill -9 <PID>

# 3. 或者修改配置使用其他端口
[server]
port = 8081
```

## ⚙️ 配置相关问题

### 问题1：LLM API 密钥错误

**症状**：
```bash
Error: LLM request failed: 401 Unauthorized
Error: invalid API key provided
```

**解决方案**：
```bash
# 1. 验证API密钥
curl -H "Authorization: Bearer $OPENMANUS_API_KEY" \
     https://api.deepseek.com/v1/models

# 2. 检查配置文件
cat configs/config.toml | grep api_key

# 3. 验证环境变量
echo $OPENMANUS_API_KEY

# 4. 测试LLM连接
./bin/openmanus config test-llm --config configs/config.toml
```

### 问题2：配置文件格式错误

**症状**：
```bash
Error: failed to parse config: Near line 15 (last key parsed 'llm.model'): bare keys cannot contain ']'
```

**解决方案**：
```bash
# 1. 验证TOML格式
./bin/openmanus config validate --config configs/config.toml

# 2. 检查常见格式错误
# 错误示例：
[llm]
model = gpt-4  # 缺少引号

# 正确格式：
[llm]
model = "gpt-4"

# 3. 使用在线TOML验证器检查格式
```

### 问题3：环境变量未生效

**症状**：
配置中的 `${VARIABLE}` 没有被替换

**解决方案**：
```bash
# 1. 验证环境变量
env | grep OPENMANUS

# 2. 检查变量语法
# 正确：${OPENMANUS_API_KEY}
# 错误：$OPENMANUS_API_KEY

# 3. 设置默认值
api_key = "${OPENMANUS_API_KEY:-default-value}"

# 4. 导出环境变量
export OPENMANUS_API_KEY="your-key"
```

## 🛠️ 工具调用问题

### 问题1：工具未找到

**症状**：
```bash
Error: tool not found: custom_tool
```

**解决方案**：
```bash
# 1. 查看可用工具
./bin/openmanus tools list --config configs/config.toml

# 2. 检查工具配置
[tools]
enabled = ["fs", "http", "custom_tool"]  # 确保工具已启用

# 3. 验证工具注册
# 在代码中确保工具已注册
tool.Register(NewCustomTool())
```

### 问题2：文件访问权限

**症状**：
```bash
Error: access denied: /path/to/file
Tool execution failed: permission denied
```

**解决方案**：
```bash
# 1. 检查文件权限
ls -la /path/to/file

# 2. 检查工具配置
[tools.filesystem]
allowed_paths = ["./workspace", "./data"]
blocked_paths = ["/etc", "/sys"]

# 3. 确保路径在允许列表中
# 错误：访问 /home/user/document.txt
# 解决：将 /home/user 添加到 allowed_paths

# 4. 检查目录权限
chmod 755 workspace
chown user:group workspace
```

### 问题3：网络请求失败

**症状**：
```bash
Error: HTTP request failed: dial tcp: lookup api.example.com: no such host
Error: HTTP request timeout
```

**解决方案**：
```bash
# 1. 测试网络连接
ping api.example.com
curl -I https://api.example.com

# 2. 检查DNS设置
nslookup api.example.com
dig api.example.com

# 3. 检查防火墙和代理
curl --proxy http://proxy:8080 https://api.example.com

# 4. 调整超时设置
[tools.http]
timeout = "60s"  # 增加超时时间
```

### 问题4：数据库连接失败

**症状**：
```bash
Error: failed to connect to database: dial tcp :3306: connect: connection refused
Error: Redis connection failed
```

**解决方案**：
```bash
# 1. 检查数据库服务状态
# MySQL
systemctl status mysql
mysql -u user -p -e "SELECT 1"

# Redis
systemctl status redis
redis-cli ping

# 2. 验证连接字符串
[tools.database.mysql]
dsn = "user:password@tcp(localhost:3306)/database"

[tools.database.redis]
addr = "localhost:6379"
password = ""

# 3. 测试网络连接
telnet localhost 3306
telnet localhost 6379

# 4. 检查防火墙设置
sudo ufw status
sudo iptables -L
```

## 📈 性能问题

### 问题1：响应速度慢

**症状**：
- Agent 执行时间过长
- LLM 请求耗时较长

**诊断和解决**：
```bash
# 1. 启用性能分析
./bin/openmanus run --config configs/config.toml --verbose --debug "your task"

# 2. 检查配置优化
[llm]
model = "gpt-3.5-turbo"  # 使用更快的模型
max_tokens = 1500        # 减少token数量
temperature = 0.1        # 降低随机性

[agent]
max_steps = 10           # 减少最大步数
reflection_steps = 5     # 增加反思间隔

# 3. 监控资源使用
top -p $(pgrep openmanus)
htop

# 4. 检查网络延迟
curl -w "@curl-format.txt" -s -o /dev/null https://api.openai.com
```

### 问题2：内存使用过高

**症状**：
```bash
Error: out of memory
系统变慢，swap使用率高
```

**解决方案**：
```bash
# 1. 监控内存使用
free -h
ps aux | grep openmanus

# 2. 调整配置减少内存使用
[agent]
max_tokens = 2000        # 减少token预算
max_steps = 10           # 减少执行步数

# 3. 启用垃圾回收优化
export GOGC=50           # 更频繁的GC
export GOMEMLIMIT=1GiB   # 设置内存限制

# 4. 检查内存泄漏
# 使用pprof工具分析
go tool pprof http://localhost:6060/debug/pprof/heap
```

### 问题3：CPU 使用率过高

**症状**：
- 系统负载高
- 响应变慢

**解决方案**：
```bash
# 1. 监控CPU使用
top -p $(pgrep openmanus)
sar -u 1 10

# 2. 限制并发
[agent]
max_concurrent_tasks = 2  # 限制并发任务数

# 3. 优化工具配置
[tools.http]
timeout = "30s"          # 减少超时等待
max_redirects = 3        # 限制重定向

# 4. 使用CPU限制
nice -n 10 ./bin/openmanus run
taskset -c 0,1 ./bin/openmanus run  # 限制CPU核心
```

## 🌐 网络连接问题

### 问题1：API 请求频率限制

**症状**：
```bash
Error: rate limit exceeded
HTTP 429 Too Many Requests
```

**解决方案**：
```bash
# 1. 实现重试机制
[agent]
max_retries = 3
retry_backoff = "5s"

# 2. 减少请求频率
[llm]
request_interval = "1s"  # 请求间隔

# 3. 使用不同的API密钥或端点
[llm]
api_key = "${BACKUP_API_KEY}"
base_url = "https://backup-api.example.com/v1"
```

### 问题2：SSL/TLS 证书问题

**症状**：
```bash
Error: x509: certificate signed by unknown authority
Error: tls: handshake failure
```

**解决方案**：
```bash
# 1. 更新证书
# Ubuntu/Debian
sudo apt-get update && sudo apt-get install ca-certificates

# CentOS/RHEL
sudo yum update ca-certificates

# 2. 检查系统时间
date
sudo ntpdate -s time.nist.gov

# 3. 临时跳过证书验证（仅测试用）
export OPENMANUS_SKIP_TLS_VERIFY=true

# 4. 手动下载证书
openssl s_client -connect api.example.com:443 -showcerts
```

### 问题3：代理服务器问题

**症状**：
- 无法访问外部API
- 代理认证失败

**解决方案**：
```bash
# 1. 设置代理环境变量
export HTTP_PROXY=http://proxy.company.com:8080
export HTTPS_PROXY=http://proxy.company.com:8080
export NO_PROXY=localhost,127.0.0.1,internal.company.com

# 2. 配置代理认证
export HTTP_PROXY=http://username:password@proxy.company.com:8080

# 3. 测试代理连接
curl --proxy $HTTP_PROXY https://api.example.com

# 4. 配置工具代理
[tools.http]
proxy_url = "http://proxy.company.com:8080"
proxy_auth = "username:password"
```

## 🤝 多Agent协作问题

### 问题1：工作流执行失败

**症状**：
```bash
Error: workflow execution failed: task dependency not satisfied
Error: agent creation failed
```

**解决方案**：
```bash
# 1. 验证工作流定义
./bin/openmanus flow validate --workflow workflow.json

# 2. 检查任务依赖
# 确保依赖任务ID正确
{
  "id": "task2",
  "dependencies": ["task1"]  # 确保task1存在
}

# 3. 验证Agent类型
{
  "agent_type": "data_analysis"  # 确保类型正确
}

# 4. 调试执行过程
./bin/openmanus flow --workflow workflow.json --debug
```

### 问题2：任务超时

**症状**：
```bash
Error: task execution timeout
Context deadline exceeded
```

**解决方案**：
```bash
# 1. 调整超时设置
{
  "timeout": "15m",           # 任务级超时
  "global_timeout": "30m"     # 工作流级超时
}

# 2. 优化任务设计
# 将大任务分解为小任务
# 减少任务复杂度

# 3. 监控执行进度
./bin/openmanus flow status --execution-id <id>
```

### 问题3：Agent 间通信问题

**症状**：
- 共享状态不一致
- 数据传递失败

**解决方案**：
```bash
# 1. 检查状态存储配置
[storage]
type = "redis"

[storage.redis]
addr = "localhost:6379"

# 2. 验证Redis连接
redis-cli ping

# 3. 检查数据序列化
# 确保共享数据可以正确序列化/反序列化

# 4. 启用详细日志
[logging]
level = "debug"
```

## 🔍 调试工具和技巧

### 启用详细日志

```bash
# 1. 命令行调试
./bin/openmanus run --config configs/config.toml --verbose --debug "your task"

# 2. 配置文件调试
[logging]
level = "debug"
output = "both"  # 输出到控制台和文件
```

### 使用配置验证工具

```bash
# 验证配置文件
./bin/openmanus config validate --config configs/config.toml

# 测试LLM连接
./bin/openmanus config test-llm --config configs/config.toml

# 测试工具可用性
./bin/openmanus tools test --config configs/config.toml

# 显示当前配置
./bin/openmanus config show --config configs/config.toml
```

### 查看执行轨迹

```bash
# 查看最新执行轨迹
cat data/traces/latest.json | jq '.'

# 分析执行步骤
jq '.steps[] | {action: .action.name, result: .observation.success}' data/traces/latest.json

# 查看错误信息
jq '.steps[] | select(.observation.success == false) | .observation.error' data/traces/latest.json
```

### 性能分析

```bash
# 1. 启用性能分析服务器
# 在代码中添加：
import _ "net/http/pprof"
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()

# 2. 分析CPU使用
go tool pprof http://localhost:6060/debug/pprof/profile

# 3. 分析内存使用
go tool pprof http://localhost:6060/debug/pprof/heap

# 4. 查看Goroutine
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

### 网络调试

```bash
# 1. 监控网络连接
netstat -tulpn | grep openmanus
ss -tulpn | grep :8080

# 2. 抓包分析
sudo tcpdump -i any -s 65535 -w openmanus.pcap port 8080

# 3. HTTP请求调试
curl -v -H "Content-Type: application/json" \
     -d '{"message": "test"}' \
     http://localhost:8080/chat

# 4. 测试工具调用
curl -X POST http://localhost:8080/tools/invoke \
     -H "Content-Type: application/json" \
     -d '{
       "tool": "fs",
       "args": {
         "operation": "read",
         "path": "./test.txt"
       }
     }'
```

### 系统资源监控

```bash
# 1. 实时监控
htop
iotop
iftop

# 2. 历史监控
sar -u 1 10    # CPU使用率
sar -r 1 10    # 内存使用
sar -d 1 10    # 磁盘IO

# 3. 进程监控
ps aux | grep openmanus
pstree -p openmanus

# 4. 文件描述符
lsof -p $(pgrep openmanus)
```

## 📞 获取帮助

### 社区支持

- **GitHub Issues**: 报告Bug和功能请求
- **讨论区**: 技术交流和经验分享
- **文档反馈**: 改进建议和错误报告

### 提交Bug报告

包含以下信息：
1. **版本信息**: `./bin/openmanus --version`
2. **系统环境**: 操作系统、Go版本等
3. **配置文件**: 脱敏后的配置
4. **错误日志**: 完整的错误信息
5. **重现步骤**: 详细的操作步骤

### 紧急问题处理

1. **服务不可用**: 检查基础设施状态
2. **数据丢失风险**: 立即停止服务并备份
3. **安全问题**: 隔离服务并评估影响
4. **性能严重下降**: 监控资源使用并调整配置

---

通过系统化的故障排除方法，大多数问题都可以快速定位和解决。记住保持冷静，按步骤诊断，并及时记录问题和解决方案！

**相关文档**: [最佳实践](BEST_PRACTICES.md) → [监控运维](MONITORING.md) → [性能优化](PERFORMANCE.md)
