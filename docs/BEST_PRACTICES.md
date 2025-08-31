# 最佳实践

本文档汇总了使用 OpenManus-Go 的最佳实践，包括开发、部署和运维的经验总结。

## 📋 目录

- [开发最佳实践](#开发最佳实践)
- [配置最佳实践](#配置最佳实践)
- [性能最佳实践](#性能最佳实践)
- [安全最佳实践](#安全最佳实践)
- [运维最佳实践](#运维最佳实践)
- [错误处理实践](#错误处理实践)
- [测试最佳实践](#测试最佳实践)

## 💻 开发最佳实践

### 任务设计原则

#### 1. 任务描述要清晰具体

**✅ 推荐做法**：
```bash
# 清晰具体的任务描述
🎯 Goal: 分析workspace/sales.csv文件，计算2024年每个月的总销售额，并生成包含趋势图的HTML报告保存到reports/monthly_sales.html

# 提供必要的上下文信息
🎯 Goal: 从https://api.example.com/users接口获取用户数据（需要Bearer token认证），提取active状态的用户，按城市分组统计，结果保存为JSON格式到data/active_users_by_city.json
```

**❌ 避免的做法**：
```bash
# 模糊不清的任务描述
🎯 Goal: 处理文件
🎯 Goal: 获取数据
🎯 Goal: 生成报告
```

#### 2. 分解复杂任务

**✅ 推荐做法**：
```bash
# 将复杂任务分解为多个步骤
🎯 Goal: 
执行数据分析流程：
1. 从sales.csv文件读取销售数据
2. 清洗数据：移除空值和重复记录
3. 按产品类别计算销售统计
4. 生成柱状图和饼图
5. 创建包含图表的HTML报告
6. 保存到reports目录
```

**❌ 避免的做法**：
```bash
# 过于复杂的单一任务
🎯 Goal: 做一个完整的销售数据分析系统包括数据收集清洗分析可视化报告生成和自动发送邮件
```

#### 3. 提供必要的约束条件

**✅ 推荐做法**：
```go
agent := agent.NewBaseAgent(llmClient, toolRegistry, store)

// 设置合理的执行约束
agentConfig := &agent.Config{
    MaxSteps:    20,    // 防止无限循环
    MaxTokens:   8000,  // 控制成本
    MaxDuration: time.Minute * 15, // 避免长时间运行
}
```

### 工具使用策略

#### 1. 选择合适的工具

**✅ 推荐做法**：
```toml
# 根据任务类型启用相应工具
[tools]
enabled = ["fs", "http", "data_analysis"]  # 只启用需要的工具
disabled = ["browser", "mysql"]            # 禁用不需要的工具

# 文件处理任务
enabled = ["fs", "data_analysis"]

# 网页数据收集
enabled = ["http", "crawler", "browser"]

# 数据库操作
enabled = ["mysql", "redis", "data_analysis"]
```

#### 2. 工具权限最小化

**✅ 推荐做法**：
```toml
[tools.filesystem]
allowed_paths = ["./workspace", "./data"]    # 只允许访问必要路径
blocked_paths = ["/etc", "/sys"]            # 明确禁止敏感路径
max_file_size = "50MB"                      # 限制文件大小

[tools.http]
allowed_domains = ["api.company.com"]       # 限制访问域名
blocked_domains = ["localhost"]             # 禁止内网访问
timeout = "30s"                             # 设置合理超时
```

### 错误处理策略

#### 1. 优雅错误处理

**✅ 推荐做法**：
```go
func handleTask() error {
    // 设置超时上下文
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    
    // 使用重试机制
    var trace *state.Trace
    var err error
    
    for i := 0; i < 3; i++ {
        trace, err = agent.Loop(ctx, goal)
        if err == nil {
            break
        }
        
        log.Printf("尝试 %d 失败: %v", i+1, err)
        time.Sleep(time.Duration(i+1) * time.Second) // 指数退避
    }
    
    if err != nil {
        return fmt.Errorf("任务执行失败，已重试3次: %w", err)
    }
    
    return nil
}
```

#### 2. 详细日志记录

**✅ 推荐做法**：
```toml
[logging]
level = "info"
format = "json"
output = "both"  # 同时输出到控制台和文件
file_path = "./logs/openmanus.log"

# 生产环境
[logging]
level = "warn"
format = "json"
output = "file"
```

### 内存管理

#### 1. 避免内存泄漏

**✅ 推荐做法**：
```go
// 及时释放资源
func processLargeFile(filename string) error {
    agent := createAgent()
    defer agent.Stop() // 确保释放资源
    
    // 分块处理大文件
    goal := fmt.Sprintf(`
    分批处理大文件 %s：
    1. 每次读取1000行数据
    2. 处理后立即写入结果
    3. 释放内存后继续下一批
    `, filename)
    
    return agent.Loop(context.Background(), goal)
}
```

## ⚙️ 配置最佳实践

### 环境配置

#### 1. 分层配置管理

**✅ 推荐做法**：
```bash
# 目录结构
configs/
├── config.base.toml     # 基础配置
├── config.dev.toml      # 开发环境
├── config.test.toml     # 测试环境
├── config.prod.toml     # 生产环境
└── secrets/             # 敏感信息
    ├── dev.env
    ├── test.env
    └── prod.env
```

#### 2. 环境变量管理

**✅ 推荐做法**：
```bash
# 使用 .env 文件管理环境变量
# .env.dev
OPENMANUS_API_KEY="dev-api-key"
OPENMANUS_LOG_LEVEL="debug"
REDIS_URL="localhost:6379"

# .env.prod
OPENMANUS_API_KEY="${VAULT_API_KEY}"
OPENMANUS_LOG_LEVEL="info"
REDIS_URL="${REDIS_CLUSTER_URL}"
```

#### 3. 敏感信息保护

**✅ 推荐做法**：
```toml
# 使用环境变量存储敏感信息
[llm]
api_key = "${OPENMANUS_API_KEY}"
base_url = "${LLM_BASE_URL}"

[tools.database.mysql]
dsn = "${MYSQL_DSN}"

[tools.database.redis]
password = "${REDIS_PASSWORD}"
```

**❌ 避免的做法**：
```toml
# 不要在配置文件中硬编码敏感信息
[llm]
api_key = "sk-real-api-key-here"  # ❌ 危险

[tools.database.mysql]
dsn = "user:password@tcp(localhost:3306)/db"  # ❌ 密码暴露
```

### 性能配置

#### 1. LLM 参数优化

**✅ 推荐做法**：
```toml
# 开发环境 - 快速响应
[llm]
model = "gpt-3.5-turbo"
temperature = 0.1
max_tokens = 2000
timeout = "30s"

# 生产环境 - 质量优先
[llm]
model = "gpt-4"
temperature = 0.0   # 确保一致性
max_tokens = 4000
timeout = "60s"

# 批量处理 - 成本优化
[llm]
model = "deepseek-chat"  # 性价比高
temperature = 0.1
max_tokens = 1500
timeout = "45s"
```

#### 2. Agent 预算控制

**✅ 推荐做法**：
```toml
# 根据任务复杂度设置预算
[agent]
# 简单任务
max_steps = 5
max_duration = "2m"
max_tokens = 2000

# 中等任务
max_steps = 15
max_duration = "10m"
max_tokens = 8000

# 复杂任务
max_steps = 30
max_duration = "30m"
max_tokens = 20000
```

## 🚀 性能最佳实践

### 优化策略

#### 1. 缓存使用

**✅ 推荐做法**：
```go
// 实现结果缓存
type CachedAgent struct {
    *agent.BaseAgent
    cache map[string]interface{}
    mutex sync.RWMutex
}

func (a *CachedAgent) Loop(ctx context.Context, goal string) (*state.Trace, error) {
    // 计算任务hash
    hash := calculateHash(goal)
    
    // 检查缓存
    if result, exists := a.getFromCache(hash); exists {
        return result, nil
    }
    
    // 执行任务
    trace, err := a.BaseAgent.Loop(ctx, goal)
    if err == nil {
        a.setCache(hash, trace)
    }
    
    return trace, err
}
```

#### 2. 并发控制

**✅ 推荐做法**：
```go
// 使用工作池限制并发
type WorkerPool struct {
    workerCount int
    taskQueue   chan Task
    resultQueue chan Result
    wg          sync.WaitGroup
}

func (p *WorkerPool) Process(tasks []Task) []Result {
    // 启动工作者
    for i := 0; i < p.workerCount; i++ {
        p.wg.Add(1)
        go p.worker()
    }
    
    // 分发任务
    go func() {
        defer close(p.taskQueue)
        for _, task := range tasks {
            p.taskQueue <- task
        }
    }()
    
    // 收集结果
    go func() {
        p.wg.Wait()
        close(p.resultQueue)
    }()
    
    var results []Result
    for result := range p.resultQueue {
        results = append(results, result)
    }
    
    return results
}
```

#### 3. 资源监控

**✅ 推荐做法**：
```go
// 监控资源使用
func monitorResources(agent Agent) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        var m runtime.MemStats
        runtime.ReadMemStats(&m)
        
        log.Printf("内存使用: %d MB", m.Alloc/1024/1024)
        log.Printf("Goroutine数量: %d", runtime.NumGoroutine())
        
        // 如果内存使用过高，触发垃圾回收
        if m.Alloc > 500*1024*1024 { // 500MB
            runtime.GC()
        }
    }
}
```

## 🔒 安全最佳实践

### 访问控制

#### 1. 网络安全

**✅ 推荐做法**：
```toml
[tools.http]
# 使用白名单策略
allowed_domains = [
    "api.company.com",
    "*.trusted-partner.com",
    "public-api.gov"
]

# 禁止访问内网和敏感服务
blocked_domains = [
    "localhost",
    "127.0.0.1",
    "169.254.169.254",  # AWS metadata service
    "metadata.google.internal",  # GCP metadata service
    "internal.company.com"
]

# 设置合理的超时
timeout = "30s"
max_redirects = 3
```

#### 2. 文件系统安全

**✅ 推荐做法**：
```toml
[tools.filesystem]
# 严格限制访问路径
allowed_paths = [
    "/app/workspace",
    "/app/data",
    "/tmp/openmanus"  # 临时文件目录
]

# 禁止访问系统敏感目录
blocked_paths = [
    "/etc",
    "/sys",
    "/proc",
    "/root",
    "/var/lib",
    "/usr/bin"
]

# 限制文件大小
max_file_size = "100MB"

# 禁用符号链接（防止路径绕过）
enable_symlinks = false
```

#### 3. 数据库安全

**✅ 推荐做法**：
```toml
[tools.database.mysql]
# 使用最小权限用户
dsn = "readonly_user:password@tcp(localhost:3306)/app_db"

# 限制连接数
max_open_conns = 5
max_idle_conns = 2

# 设置连接超时
conn_max_lifetime = "1h"

[tools.database.redis]
# 使用独立数据库
db = 1  # 不使用默认的db 0

# 限制连接池
pool_size = 5
```

### 输入验证

#### 1. 参数验证

**✅ 推荐做法**：
```go
func validateTaskInput(goal string) error {
    // 检查长度
    if len(goal) > 10000 {
        return errors.New("任务描述过长")
    }
    
    // 检查敏感词
    forbidden := []string{"rm -rf", "DROP TABLE", "DELETE FROM"}
    for _, word := range forbidden {
        if strings.Contains(strings.ToLower(goal), strings.ToLower(word)) {
            return fmt.Errorf("任务包含禁止的操作: %s", word)
        }
    }
    
    return nil
}
```

#### 2. 输出过滤

**✅ 推荐做法**：
```go
func sanitizeOutput(output string) string {
    // 移除敏感信息
    patterns := []string{
        `password:\s*\w+`,
        `api_key:\s*\w+`,
        `token:\s*\w+`,
    }
    
    for _, pattern := range patterns {
        re := regexp.MustCompile(pattern)
        output = re.ReplaceAllString(output, "password: [REDACTED]")
    }
    
    return output
}
```

## 🔧 运维最佳实践

### 监控配置

#### 1. 关键指标监控

**✅ 推荐做法**：
```toml
[monitoring]
enabled = true
metrics_port = 9090

# 关键指标
[monitoring.metrics]
# 执行指标
execution_success_rate = true
execution_duration = true
execution_count = true

# 系统指标
memory_usage = true
cpu_usage = true
goroutine_count = true

# 业务指标
tool_usage_stats = true
error_rate_by_type = true
llm_token_usage = true
```

#### 2. 告警设置

**✅ 推荐做法**：
```yaml
# prometheus alerts
groups:
  - name: openmanus
    rules:
      - alert: HighErrorRate
        expr: rate(openmanus_errors_total[5m]) > 0.1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "错误率过高"
          description: "过去5分钟错误率超过10%"
          
      - alert: HighMemoryUsage
        expr: openmanus_memory_usage_bytes > 1073741824  # 1GB
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "内存使用过高"
```

#### 3. 日志管理

**✅ 推荐做法**：
```toml
[logging]
# 结构化日志
format = "json"
level = "info"

# 日志轮转
max_size = "100MB"
max_backups = 10
max_age = "30d"
compress = true

# 敏感信息过滤
redact_patterns = [
    "password",
    "api_key",
    "token",
    "secret"
]
```

### 备份策略

#### 1. 状态备份

**✅ 推荐做法**：
```bash
#!/bin/bash
# backup.sh

# 备份执行轨迹
tar -czf "traces_$(date +%Y%m%d_%H%M%S).tar.gz" data/traces/

# 备份配置文件
cp configs/config.toml "backup/config_$(date +%Y%m%d).toml"

# 上传到云存储
aws s3 cp traces_*.tar.gz s3://backup-bucket/openmanus/
```

#### 2. 数据库备份

**✅ 推荐做法**：
```bash
# Redis备份
redis-cli --rdb redis_$(date +%Y%m%d_%H%M%S).rdb

# MySQL备份
mysqldump -u backup_user -p app_db > mysql_$(date +%Y%m%d_%H%M%S).sql
```

### 容量规划

#### 1. 存储规划

**✅ 推荐做法**：
```bash
# 监控磁盘使用
df -h

# 清理旧的执行轨迹
find data/traces/ -name "*.json" -mtime +30 -delete

# 压缩旧日志
find logs/ -name "*.log" -mtime +7 -exec gzip {} \;
```

#### 2. 性能调优

**✅ 推荐做法**：
```toml
# 根据负载调整配置
[agent]
max_steps = 15        # 平衡质量和速度
max_duration = "10m"  # 防止长时间运行

[tools.http]
timeout = "30s"       # 合理的网络超时
max_redirects = 3     # 限制重定向次数

# 调整资源限制
[performance]
max_memory = "2GB"
max_cpu_percent = 80
gc_percent = 100      # Go垃圾回收调优
```

## 🧪 测试最佳实践

### 单元测试

**✅ 推荐做法**：
```go
func TestAgentExecution(t *testing.T) {
    // 使用Mock LLM客户端
    mockLLM := &MockLLMClient{
        responses: map[string]string{
            "创建文件": `{"action": "fs", "args": {"operation": "write", "path": "test.txt", "content": "hello"}}`,
        },
    }
    
    // 创建测试Agent
    agent := agent.NewBaseAgent(mockLLM, testRegistry, memoryStore)
    
    // 执行测试
    trace, err := agent.Loop(context.Background(), "创建一个测试文件")
    
    // 验证结果
    assert.NoError(t, err)
    assert.Equal(t, state.TraceStatusCompleted, trace.Status)
    assert.FileExists(t, "test.txt")
}
```

### 集成测试

**✅ 推荐做法**：
```go
func TestWorkflowExecution(t *testing.T) {
    if testing.Short() {
        t.Skip("跳过集成测试")
    }
    
    // 设置测试环境
    testConfig := createTestConfig()
    engine := flow.NewEngine(testConfig)
    
    // 创建测试工作流
    workflow := createTestWorkflow()
    
    // 执行测试
    execution, err := engine.Execute(context.Background(), workflow, nil)
    
    // 验证结果
    assert.NoError(t, err)
    assert.Equal(t, flow.StatusCompleted, execution.Status)
    
    // 验证输出文件
    assert.FileExists(t, "output/result.json")
}
```

### 性能测试

**✅ 推荐做法**：
```go
func BenchmarkAgentLoop(b *testing.B) {
    agent := createTestAgent()
    goal := "执行简单的文件操作"
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := agent.Loop(context.Background(), goal)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func TestConcurrentExecution(t *testing.T) {
    agent := createTestAgent()
    concurrency := 10
    
    var wg sync.WaitGroup
    results := make(chan error, concurrency)
    
    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _, err := agent.Loop(context.Background(), "测试并发执行")
            results <- err
        }()
    }
    
    wg.Wait()
    close(results)
    
    for err := range results {
        assert.NoError(t, err)
    }
}
```

## 🚨 常见陷阱和解决方案

### 陷阱1：无限循环

**问题**：Agent 陷入重复执行相同操作的循环

**解决方案**：
```toml
[agent]
max_steps = 15           # 限制最大步数
max_duration = "10m"     # 设置超时时间
reflection_steps = 3     # 启用反思机制
```

### 陷阱2：内存泄漏

**问题**：长时间运行后内存占用过高

**解决方案**：
```go
// 定期清理资源
func cleanupResources() {
    runtime.GC()
    debug.FreeOSMemory()
}

// 设置内存限制
func setMemoryLimit() {
    limit := 1 * 1024 * 1024 * 1024 // 1GB
    debug.SetMemoryLimit(int64(limit))
}
```

### 陷阱3：API 配额耗尽

**问题**：LLM API 调用过于频繁

**解决方案**：
```toml
[llm]
max_tokens = 2000        # 减少单次token使用
temperature = 0.1        # 降低随机性

[agent]
max_steps = 10           # 减少执行步数
reflection_steps = 5     # 增加反思间隔
```

### 陷阱4：文件权限错误

**问题**：无法访问或创建文件

**解决方案**：
```bash
# 确保目录存在且有正确权限
mkdir -p workspace data logs
chmod 755 workspace data logs

# 检查运行用户权限
ls -la workspace/
```

---

遵循这些最佳实践可以帮助您更好地使用 OpenManus-Go，避免常见问题，并获得最佳的性能和稳定性！

**相关文档**: [性能优化](PERFORMANCE.md) → [故障排除](TROUBLESHOOTING.md) → [监控运维](MONITORING.md)
