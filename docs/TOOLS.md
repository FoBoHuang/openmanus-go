# 工具开发指南

本文档介绍如何在 OpenManus-Go 中开发和使用工具。

## 工具概述

工具是 OpenManus-Go 中 Agent 与外部世界交互的接口。通过工具，Agent 可以：

- 访问网络资源（HTTP 请求）
- 操作文件系统
- 查询数据库
- 控制浏览器
- 执行系统命令
- 处理数据分析任务

## 工具接口

所有工具都必须实现 `Tool` 接口：

```go
type Tool interface {
    Name() string
    Description() string
    InputSchema() map[string]any
    OutputSchema() map[string]any
    Invoke(ctx context.Context, args map[string]any) (map[string]any, error)
}
```

## 内置工具详解

### 1. HTTP 工具

**功能**：发送 HTTP 请求

**使用示例**：
```json
{
  "name": "http",
  "args": {
    "url": "https://api.github.com/users/octocat",
    "method": "GET",
    "headers": {
      "User-Agent": "OpenManus-Go"
    }
  }
}
```

**输出**：
```json
{
  "status_code": 200,
  "headers": {...},
  "body": "...",
  "content_type": "application/json"
}
```

### 2. 文件系统工具

**功能**：文件和目录操作

**支持操作**：
- `read`: 读取文件
- `write`: 写入文件
- `list`: 列出目录
- `delete`: 删除文件
- `mkdir`: 创建目录
- `exists`: 检查文件存在
- `stat`: 获取文件信息

**使用示例**：
```json
{
  "name": "fs",
  "args": {
    "operation": "write",
    "path": "./output.txt",
    "content": "Hello, World!"
  }
}
```

### 3. Redis 工具

**功能**：Redis 数据库操作

**支持操作**：
- 字符串：`get`, `set`, `del`
- 哈希：`hget`, `hset`, `hdel`
- 列表：`lpush`, `rpop`
- 集合：`sadd`, `srem`
- 有序集合：`zadd`, `zrange`

**使用示例**：
```json
{
  "name": "redis",
  "args": {
    "operation": "set",
    "key": "user:123",
    "value": "john_doe",
    "ttl": 3600
  }
}
```

### 4. MySQL 工具

**功能**：MySQL 数据库操作

**支持操作**：
- `query`: 查询数据
- `execute`: 执行 SQL
- `insert`: 插入数据
- `update`: 更新数据
- `delete`: 删除数据
- `describe`: 描述表结构
- `show_tables`: 显示所有表

**使用示例**：
```json
{
  "name": "mysql",
  "args": {
    "operation": "query",
    "sql": "SELECT * FROM users WHERE age > ?",
    "params": [18]
  }
}
```

### 5. 浏览器工具

**功能**：网页浏览器自动化

**支持操作**：
- `navigate`: 导航到页面
- `click`: 点击元素
- `type`: 输入文本
- `get_text`: 获取文本
- `get_html`: 获取 HTML
- `screenshot`: 截图
- `wait_for_element`: 等待元素
- `execute_js`: 执行 JavaScript

**使用示例**：
```json
{
  "name": "browser",
  "args": {
    "operation": "navigate",
    "url": "https://example.com"
  }
}
```

### 6. 爬虫工具

**功能**：网页内容抓取

**支持操作**：
- `scrape`: 抓取页面内容
- `crawl`: 批量爬取
- `extract_links`: 提取链接
- `extract_text`: 提取文本
- `extract_images`: 提取图片

**使用示例**：
```json
{
  "name": "crawler",
  "args": {
    "operation": "scrape",
    "url": "https://news.ycombinator.com",
    "selector": ".storylink"
  }
}
```

## 开发自定义工具

### 1. 基础结构

```go
package main

import (
    "context"
    "github.com/openmanus/openmanus-go/pkg/tool"
)

// CustomTool 自定义工具
type CustomTool struct {
    *tool.BaseTool
}

// NewCustomTool 创建自定义工具
func NewCustomTool() *CustomTool {
    inputSchema := tool.CreateJSONSchema("object", map[string]any{
        "param1": tool.StringProperty("参数1描述"),
        "param2": tool.NumberProperty("参数2描述"),
    }, []string{"param1"})

    outputSchema := tool.CreateJSONSchema("object", map[string]any{
        "success": tool.BooleanProperty("操作是否成功"),
        "result":  tool.StringProperty("操作结果"),
    }, []string{"success"})

    baseTool := tool.NewBaseTool(
        "custom_tool",
        "自定义工具描述",
        inputSchema,
        outputSchema,
    )

    return &CustomTool{
        BaseTool: baseTool,
    }
}

// Invoke 实现工具逻辑
func (c *CustomTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
    // 获取参数
    param1, _ := args["param1"].(string)
    param2, _ := args["param2"].(float64)

    // 执行业务逻辑
    result := performCustomOperation(param1, param2)

    // 返回结果
    return map[string]any{
        "success": true,
        "result":  result,
    }, nil
}

func performCustomOperation(param1 string, param2 float64) string {
    // 自定义业务逻辑
    return "操作完成"
}
```

### 2. 注册工具

```go
// 注册到默认注册表
tool.Register(NewCustomTool())

// 或注册到自定义注册表
registry := tool.NewRegistry()
registry.Register(NewCustomTool())
```

### 3. 工具配置

如果工具需要配置，可以扩展配置结构：

```go
// 在 config/config.go 中添加
type ToolsConfig struct {
    // ... 现有配置
    
    Custom struct {
        APIKey  string `mapstructure:"api_key"`
        BaseURL string `mapstructure:"base_url"`
        Timeout int    `mapstructure:"timeout"`
    } `mapstructure:"custom"`
}
```

配置文件：
```toml
[tools.custom]
api_key = "your-api-key"
base_url = "https://api.example.com"
timeout = 30
```

### 4. 错误处理

```go
func (c *CustomTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
    // 参数验证
    if err := c.ValidateInput(args); err != nil {
        return c.errorResult(err.Error()), nil
    }

    // 业务逻辑
    result, err := c.performOperation(ctx, args)
    if err != nil {
        return c.errorResult(fmt.Sprintf("操作失败: %v", err)), nil
    }

    return map[string]any{
        "success": true,
        "result":  result,
    }, nil
}

func (c *CustomTool) errorResult(message string) map[string]any {
    return map[string]any{
        "success": false,
        "error":   message,
    }
}
```

## 工具最佳实践

### 1. 设计原则

- **单一职责**：每个工具专注于特定领域
- **幂等性**：相同输入应产生相同输出
- **错误处理**：优雅处理各种异常情况
- **参数验证**：严格验证输入参数
- **文档完善**：提供清晰的描述和示例

### 2. 性能优化

```go
// 使用连接池
type DatabaseTool struct {
    *tool.BaseTool
    pool *sql.DB // 连接池
}

// 支持超时控制
func (d *DatabaseTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
    // 创建带超时的上下文
    timeout := 30 * time.Second
    if t, ok := args["timeout"].(float64); ok {
        timeout = time.Duration(t) * time.Second
    }
    
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    // 执行操作
    return d.executeWithTimeout(ctx, args)
}

// 使用缓存
type CachedTool struct {
    *tool.BaseTool
    cache map[string]any
    mu    sync.RWMutex
}
```

### 3. 安全考虑

```go
// 输入验证
func (t *CustomTool) validateInput(args map[string]any) error {
    // 检查必需参数
    if _, ok := args["required_param"]; !ok {
        return fmt.Errorf("missing required parameter")
    }
    
    // 检查参数类型
    if _, ok := args["string_param"].(string); !ok {
        return fmt.Errorf("invalid parameter type")
    }
    
    // 检查参数范围
    if num, ok := args["number_param"].(float64); ok {
        if num < 0 || num > 100 {
            return fmt.Errorf("parameter out of range")
        }
    }
    
    return nil
}

// 访问控制
func (t *FileSystemTool) checkAccess(path string) error {
    absPath, err := filepath.Abs(path)
    if err != nil {
        return err
    }
    
    // 检查是否在允许的路径中
    for _, allowed := range t.allowedPaths {
        if strings.HasPrefix(absPath, allowed) {
            return nil
        }
    }
    
    return fmt.Errorf("access denied: %s", path)
}
```

### 4. 测试

```go
func TestCustomTool(t *testing.T) {
    tool := NewCustomTool()
    
    tests := []struct {
        name     string
        args     map[string]any
        expected map[string]any
        wantErr  bool
    }{
        {
            name: "valid input",
            args: map[string]any{
                "param1": "test",
                "param2": 42.0,
            },
            expected: map[string]any{
                "success": true,
                "result":  "操作完成",
            },
            wantErr: false,
        },
        // 更多测试用例...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := tool.Invoke(context.Background(), tt.args)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("Invoke() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if !reflect.DeepEqual(result, tt.expected) {
                t.Errorf("Invoke() = %v, want %v", result, tt.expected)
            }
        })
    }
}
```

## 工具集成

### 1. 工具发现

Agent 通过工具注册表发现可用工具：

```go
// 获取所有工具
tools := registry.List()

// 获取工具清单（用于 LLM）
manifest := registry.GetToolsManifest()
```

### 2. 工具调用

```go
// 直接调用
result, err := registry.Invoke(ctx, "tool_name", args)

// 通过执行器调用（支持重试等）
executor := tool.NewExecutor(registry, timeout)
observation, err := executor.Execute(ctx, action)
```

### 3. 批量执行

```go
actions := []state.Action{
    {Name: "http", Args: map[string]any{"url": "https://api1.com"}},
    {Name: "http", Args: map[string]any{"url": "https://api2.com"}},
}

observations, err := executor.BatchExecute(ctx, actions)
```

## 工具生态

### 1. 社区工具

鼓励社区开发和分享工具：

- 标准化接口
- 文档规范
- 测试要求
- 安全审查

### 2. 工具市场

未来计划：

- 工具包管理
- 版本控制
- 依赖管理
- 自动更新

### 3. 工具组合

支持工具组合和流水线：

```go
// 工具链
pipeline := []string{"crawler", "data_analysis", "visualization"}

// 数据流转
result := input
for _, toolName := range pipeline {
    result, err = registry.Invoke(ctx, toolName, map[string]any{
        "input": result,
    })
}
```

## 故障排除

### 1. 常见问题

- **工具未注册**：检查工具是否正确注册到注册表
- **参数错误**：验证参数类型和必需字段
- **权限问题**：检查访问权限和安全设置
- **超时错误**：调整超时设置或优化工具性能

### 2. 调试技巧

```go
// 启用详细日志
logger.SetLevel(zap.DebugLevel)

// 工具执行追踪
func (t *CustomTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
    logger.Debug("tool invoked", 
        zap.String("tool", t.Name()),
        zap.Any("args", args))
    
    start := time.Now()
    defer func() {
        logger.Debug("tool completed",
            zap.Duration("duration", time.Since(start)))
    }()
    
    // 工具逻辑...
}
```

### 3. 性能监控

```go
// 添加指标收集
func (t *CustomTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
    // 记录调用次数
    toolInvocationCounter.WithLabelValues(t.Name()).Inc()
    
    // 记录执行时间
    timer := prometheus.NewTimer(toolDurationHistogram.WithLabelValues(t.Name()))
    defer timer.ObserveDuration()
    
    // 执行工具逻辑...
    result, err := t.execute(ctx, args)
    
    // 记录成功/失败
    if err != nil {
        toolErrorCounter.WithLabelValues(t.Name()).Inc()
    } else {
        toolSuccessCounter.WithLabelValues(t.Name()).Inc()
    }
    
    return result, err
}
```
