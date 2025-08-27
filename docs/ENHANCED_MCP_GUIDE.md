# OpenManus-Go 增强 MCP 集成指南

## 概述

OpenManus-Go 现在提供了完全重新设计的 MCP (Model Context Protocol) 集成系统，参考了 [OpenManus](https://github.com/FoundationAgents/OpenManus) 项目的设计理念，实现了智能工具发现、选择和执行的完整流程。

## 核心特性

### 1. 智能工具发现 (Intelligent Tool Discovery)
- **自动发现**: 自动从配置的 MCP 服务器发现所有可用工具
- **实时更新**: 定期刷新工具列表，确保始终获得最新的工具信息
- **冲突处理**: 自动处理同名工具冲突，使用 `server.tool` 格式区分

### 2. 智能工具选择 (Smart Tool Selection)
- **语义匹配**: 基于用户请求与工具描述的语义相似度选择最佳工具
- **LLM 辅助**: 使用 LLM 在多个候选工具中进行智能选择
- **上下文感知**: 考虑执行历史和上下文信息优化选择

### 3. 自动参数生成 (Automatic Parameter Generation)
- **Schema 驱动**: 根据工具的 JSON Schema 自动生成参数
- **智能推断**: 从用户请求中智能提取参数值
- **类型转换**: 自动处理参数类型转换和验证

### 4. 执行监控和统计 (Execution Monitoring)
- **成功率跟踪**: 记录每个工具的执行成功率
- **性能监控**: 监控工具执行延迟和性能指标
- **错误处理**: 完善的错误处理和重试机制

## 架构设计

```
用户请求 → Agent → Planner → MCP Tool Selector → MCP Executor → MCP Server
                        ↓              ↓              ↓
                 MCP Discovery ← Tool Selection ← Parameter Generation
```

### 核心组件

1. **MCPDiscoveryService**: 负责从 MCP 服务器发现和管理工具
2. **MCPToolSelector**: 智能选择最合适的工具并生成参数
3. **MCPExecutor**: 执行 MCP 工具调用并收集统计信息
4. **Enhanced Planner**: 集成 MCP 功能的增强规划器

## 配置说明

### 基本配置

```toml
[mcp.servers]

# 基本服务器配置
[mcp.servers.my-service]
url = "https://api.example.com/mcp"
headers.X-API-Key = "your-api-key"

# 支持多种认证方式
[mcp.servers.auth-service]
url = "https://secure.example.com/mcp"
headers.Authorization = "Bearer your-token"
headers.X-Custom-Header = "custom-value"
```

### 高级配置

查看 `configs/config.enhanced-mcp.toml` 获取完整的配置示例，包括：
- 股票信息服务
- 天气信息服务
- 新闻搜索服务
- 数据分析服务
- 文档搜索服务
- 代码生成服务
- 翻译服务

## 使用方法

### 1. 基本使用

```bash
# 使用增强 MCP 配置运行
openmanus run --config configs/config.enhanced-mcp.toml "查询苹果公司最新股价"

# 交互模式
openmanus run --config configs/config.enhanced-mcp.toml --interactive
```

### 2. 智能工具选择示例

```bash
# Agent 会自动选择最合适的 MCP 工具
openmanus run "获取今天北京的天气情况"
openmanus run "搜索最新的 AI 技术新闻"
openmanus run "分析这个 CSV 文件的数据趋势"
```

### 3. 调试和监控

```bash
# 启用详细日志查看 MCP 交互过程
openmanus run --debug "你的请求"

# 查看工具执行统计
openmanus run --verbose "你的请求"
```

## 工作流程详解

### 1. 工具发现阶段
```
启动时 → 连接所有配置的 MCP 服务器 → 调用 tools/list → 解析工具信息 → 存储到本地缓存
```

### 2. 智能选择阶段
```
用户请求 → 关键词分析 → 候选工具搜索 → LLM 评估选择 → 返回最佳工具
```

### 3. 参数生成阶段
```
选定工具 → 获取 Input Schema → LLM 参数生成 → 类型验证 → 参数优化
```

### 4. 执行阶段
```
工具调用 → MCP 协议通信 → 结果解析 → 统计更新 → 返回结果
```

## 优势特性

### 相比原有 MCP 实现的改进

1. **更智能的选择**: 不再需要手动指定工具名，Agent 自动选择最合适的工具
2. **更好的参数处理**: 自动从用户请求中提取和生成参数
3. **更强的容错性**: 完善的错误处理和重试机制
4. **更丰富的监控**: 详细的执行统计和性能监控
5. **更好的扩展性**: 支持动态添加和移除 MCP 服务器

### 与 OpenManus 项目的对比

| 特性 | OpenManus (Python) | OpenManus-Go (增强版) |
|------|-------------------|----------------------|
| 工具发现 | 手动配置 | 自动发现 + 定期刷新 |
| 工具选择 | 基础匹配 | 语义匹配 + LLM 辅助 |
| 参数生成 | 简单提取 | 智能推断 + Schema 驱动 |
| 错误处理 | 基础重试 | 智能重试 + 统计分析 |
| 性能监控 | 无 | 详细统计 + 成功率跟踪 |

## 故障排除

### 常见问题

1. **工具未发现**
   - 检查 MCP 服务器 URL 是否正确
   - 验证认证信息是否有效
   - 查看日志中的连接错误信息

2. **参数生成失败**
   - 确保用户请求包含足够的信息
   - 检查工具的 Input Schema 是否正确
   - 启用 debug 日志查看详细过程

3. **工具执行超时**
   - 增加 Agent 的 max_duration 配置
   - 检查 MCP 服务器的响应时间
   - 考虑使用重试机制

### 调试技巧

```bash
# 查看详细的 MCP 交互日志
export OPENMANUS_LOG_LEVEL=debug
openmanus run "你的请求"

# 查看工具发现过程
openmanus run --verbose "你的请求"
```

## 扩展开发

### 添加自定义 MCP 服务器

1. 在配置文件中添加服务器信息
2. 实现符合 MCP 协议的服务器端点
3. 重启 Agent 自动发现新工具

### 自定义工具选择逻辑

可以通过继承 `MCPToolSelector` 类并重写相关方法来自定义工具选择逻辑。

### 监控和统计扩展

`MCPExecutor` 提供了丰富的统计信息接口，可以集成到监控系统中。

## 最佳实践

1. **配置多个 MCP 服务器**: 提供不同类型的工具以增加覆盖面
2. **合理设置超时时间**: 平衡响应速度和成功率
3. **监控工具性能**: 定期检查工具执行统计，优化配置
4. **使用描述性的服务器名称**: 便于调试和维护
5. **定期更新工具列表**: 确保获得最新的工具能力

## 贡献指南

欢迎提交 Issue 和 Pull Request 来改进 MCP 集成功能。在提交前请确保：

1. 代码通过所有测试
2. 添加适当的日志和错误处理
3. 更新相关文档
4. 遵循项目的编码规范

## 参考资源

- [MCP 协议规范](https://spec.modelcontextprotocol.io/)
- [OpenManus 项目](https://github.com/FoundationAgents/OpenManus)
- [项目架构文档](./ARCHITECTURE.md)
- [工具开发指南](./TOOLS.md)
