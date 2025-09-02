# 轨迹管理功能

OpenManus-Go 提供了完整的执行轨迹保存和管理功能，帮助用户跟踪、分析和管理 Agent 的执行历史。

## 功能概述

轨迹（Trace）记录了 Agent 执行任务的完整过程，包括：
- 目标（Goal）
- 执行步骤（Steps）
- 每个步骤的动作（Action）和观测结果（Observation）
- 预算使用情况（Budget）
- 执行状态和时间信息

## 轨迹保存

### 自动保存

在运行 Agent 任务时，可以通过 `--save-trace` 参数启用轨迹保存：

```bash
# 保存轨迹（默认启用）
openmanus run "你的任务目标" --save-trace=true

# 禁用轨迹保存
openmanus run "你的任务目标" --save-trace=false
```

### 自定义保存路径

可以通过 `--trace-path` 参数指定轨迹保存路径：

```bash
# 保存到自定义路径
openmanus run "你的任务目标" --trace-path="./my-traces"
```

### 配置文件设置

在 `config.toml` 中配置轨迹存储：

```toml
[storage]
type = "file"  # 存储类型：file, memory, redis, s3
base_path = "./data/traces"  # 文件存储路径

# Redis 存储配置（可选）
[storage.redis]
addr = "localhost:6379"
password = ""
db = 0

# S3 存储配置（可选）
[storage.s3]
region = "us-east-1"
bucket = "openmanus-traces"
access_key = ""
secret_key = ""
```

## 轨迹管理命令

### 列出轨迹

```bash
# 列出所有轨迹
openmanus trace list

# 显示详细信息
openmanus trace list --verbose

# 限制显示数量
openmanus trace list --limit 5
```

### 查看轨迹详情

```bash
# 基本信息
openmanus trace show <trace-id>

# 显示步骤详情
openmanus trace show <trace-id> --steps

# 显示观测结果
openmanus trace show <trace-id> --observations

# 显示完整信息
openmanus trace show <trace-id> --steps --observations
```

### 删除轨迹

```bash
# 删除指定轨迹（会询问确认）
openmanus trace delete <trace-id>

# 强制删除
openmanus trace delete <trace-id> --force
```

### 清理旧轨迹

```bash
# 清理30天前的轨迹
openmanus trace clean --days 30

# 预览将被删除的轨迹
openmanus trace clean --days 30 --dry-run

# 清理7天前的轨迹
openmanus trace clean --days 7
```

## 轨迹文件格式

轨迹以 JSON 格式保存，包含以下主要字段：

```json
{
  "goal": "任务目标",
  "steps": [
    {
      "index": 0,
      "action": {
        "name": "工具名称",
        "args": {
          "参数": "值"
        },
        "reason": "执行原因"
      },
      "observation": {
        "tool": "工具名称",
        "output": {
          "结果": "值"
        },
        "err_msg": "错误信息（如果有）",
        "latency_ms": 100
      },
      "timestamp": "2025-09-02T18:16:29Z"
    }
  ],
  "reflections": [
    {
      "step_index": 2,
      "result": {
        "revise_plan": false,
        "next_action_hint": "建议",
        "should_stop": false,
        "reason": "反思原因",
        "confidence": 0.8
      },
      "timestamp": "2025-09-02T18:16:30Z"
    }
  ],
  "budget": {
    "max_steps": 30,
    "max_tokens": 8000,
    "max_duration": 600000000000,
    "used_steps": 1,
    "used_tokens": 150,
    "start_time": "2025-09-02T18:16:21Z"
  },
  "status": "completed",
  "created_at": "2025-09-02T18:16:21Z",
  "updated_at": "2025-09-02T18:16:29Z"
}
```

## 文件命名规则

轨迹文件按以下格式命名：
```
trace_<时间戳>_<目标摘要>.json
```

例如：
- `trace_20250902_181629_计算_3+5_等于多少.json`
- `trace_20250902_182030_创建文件.json`

## 存储类型

### 文件存储（File）
- **优点**：简单、可直接查看、易于备份
- **缺点**：大量轨迹时性能较低
- **适用场景**：开发、测试、小规模使用

### 内存存储（Memory）
- **优点**：速度快
- **缺点**：重启后丢失、不持久化
- **适用场景**：临时测试、性能测试

### Redis 存储（Redis）
- **优点**：高性能、支持集群
- **缺点**：需要额外的 Redis 服务
- **适用场景**：生产环境、高并发场景

### S3 存储（S3）
- **优点**：无限容量、高可用性
- **缺点**：网络延迟、成本
- **适用场景**：大规模部署、长期存档

## 最佳实践

### 1. 轨迹管理
- 定期清理旧轨迹以节省存储空间
- 为重要任务使用描述性的目标文本
- 在生产环境中使用 Redis 或 S3 存储

### 2. 性能优化
- 对于高频任务，考虑禁用轨迹保存
- 使用 `--limit` 参数限制列表显示数量
- 定期备份重要的轨迹文件

### 3. 故障排查
- 使用 `--observations` 查看工具执行的详细结果
- 检查失败步骤的错误信息
- 分析反思记录了解 Agent 的决策过程

### 4. 安全注意事项
- 轨迹文件可能包含敏感信息，注意访问权限
- 在共享环境中使用独立的存储路径
- 定期清理包含敏感数据的轨迹

## 示例用法

```bash
# 1. 运行任务并保存轨迹
openmanus run "分析sales.csv文件并生成报告" --save-trace=true

# 2. 查看所有轨迹
openmanus trace list --verbose

# 3. 查看特定轨迹的详细信息
openmanus trace show trace_20250902_181629_分析sales_csv文件 --steps --observations

# 4. 清理一周前的轨迹
openmanus trace clean --days 7

# 5. 交互模式（自动保存轨迹）
openmanus run --interactive --save-trace=true
```

## 故障排查

### 轨迹保存失败
1. 检查存储路径权限
2. 确认磁盘空间充足
3. 验证配置文件格式正确

### 无法找到轨迹
1. 确认轨迹ID正确（可通过 `trace list` 查看）
2. 检查存储配置是否一致
3. 验证文件路径是否存在

### 性能问题
1. 考虑使用 Redis 存储
2. 定期清理旧轨迹
3. 调整列表显示限制

通过这些功能，您可以有效地管理和分析 OpenManus-Go Agent 的执行历史，提高调试效率和系统可观测性。
