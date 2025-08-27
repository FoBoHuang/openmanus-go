# 配置管理示例

这个示例展示了 OpenManus-Go 框架中的配置管理系统，包括配置加载、验证和使用的最佳实践。

## 🎯 学习目标

通过这个示例，你将学会：
- 理解配置系统的层次结构
- 掌握配置文件的格式和结构
- 学会使用环境变量覆盖配置
- 了解配置验证的重要性
- 掌握配置管理的最佳实践

## 📋 配置优先级

OpenManus-Go 的配置系统遵循以下优先级顺序：

```
环境变量 > 配置文件 > 默认值
```

1. **默认值** - 框架提供的基础配置
2. **配置文件** - TOML 格式的配置文件
3. **环境变量** - 运行时环境变量（最高优先级）

## 🔧 配置文件结构

### 基本配置文件 (`config.toml`)

```toml
# LLM 配置
[llm]
model = "deepseek-chat"
base_url = "https://api.deepseek.com/v1"
api_key = "your-api-key-here"
temperature = 0.1
max_tokens = 4000

# Agent 配置
[agent]
max_steps = 10
max_tokens = 8000
max_duration = "5m"
reflection_steps = 3
max_retries = 2

# MCP 服务器配置
[[mcp_servers]]
name = "stock-helper"
transport = "sse"
url = "https://mcp.example.com/stock-helper"

[[mcp_servers]]
name = "weather-service"
transport = "http"
url = "http://localhost:8080/weather"

# 工具配置
[tools.filesystem]
allowed_paths = ["./workspace", "./data"]
blocked_paths = ["/etc", "/sys", "/usr"]

[tools.http]
timeout = 30
max_redirects = 5
blocked_domains = ["localhost", "127.0.0.1"]

[tools.redis]
host = "localhost"
port = 6379
password = ""
database = 0

[tools.mysql]
host = "localhost"
port = 3306
username = "user"
password = "password"
database = "openmanus"
```

## 🌍 环境变量

支持的环境变量前缀：`OPENMANUS_`

### LLM 配置
- `OPENMANUS_LLM_MODEL` - LLM 模型名称
- `OPENMANUS_LLM_BASE_URL` - API 基础 URL
- `OPENMANUS_LLM_API_KEY` - API 密钥
- `OPENMANUS_LLM_TEMPERATURE` - 生成温度
- `OPENMANUS_LLM_MAX_TOKENS` - 最大令牌数

### Agent 配置
- `OPENMANUS_AGENT_MAX_STEPS` - 最大执行步数
- `OPENMANUS_AGENT_MAX_TOKENS` - 最大令牌数
- `OPENMANUS_AGENT_MAX_DURATION` - 最大执行时间
- `OPENMANUS_AGENT_REFLECTION_STEPS` - 反思步数间隔

### 示例
```bash
# 设置环境变量
export OPENMANUS_LLM_API_KEY="sk-your-real-api-key"
export OPENMANUS_LLM_MODEL="gpt-4"
export OPENMANUS_AGENT_MAX_STEPS="15"

# 运行程序
go run main.go
```

## 🚀 运行示例

```bash
# 进入示例目录
cd examples/basic/03-configuration

# 运行示例
go run main.go
```

## 📊 预期输出

```
⚙️  OpenManus-Go Configuration Example
======================================

📋 1. 默认配置
=============

--- 默认配置 ---
LLM 配置:
  模型: deepseek-chat
  基础URL: https://api.deepseek.com/v1
  API密钥: your-api-key-here
  温度: 0.1
  最大令牌: 4000
Agent 配置:
  最大步数: 10
  最大令牌: 8000
  最大持续时间: 5m0s
  反思步数: 3
MCP 服务器数量: 0

📄 2. 配置文件加载
=================
✅ 找到配置文件: ../../../configs/config.toml
✅ 配置文件加载成功

--- 配置文件 ---
LLM 配置:
  模型: deepseek-chat
  基础URL: https://api.deepseek.com/v1
  API密钥: sk-***-key
  温度: 0.1
  最大令牌: 4000
Agent 配置:
  最大步数: 10
  最大令牌: 8000
  最大持续时间: 5m0s
  反思步数: 3
MCP 服务器数量: 2
MCP 服务器:
  1. stock-helper (sse)
  2. weather-service (http)

📝 3. 创建示例配置
=================
✅ 创建示例配置文件: example_config.toml
✅ 示例配置加载成功

--- 示例配置 ---
LLM 配置:
  模型: deepseek-chat
  基础URL: https://api.deepseek.com/v1
  API密钥: your-api-key-here
  温度: 0.2
  最大令牌: 4000
Agent 配置:
  最大步数: 12
  最大令牌: 10000
  最大持续时间: 8m0s
  反思步数: 4
MCP 服务器数量: 2
MCP 服务器:
  1. example-server (sse)
  2. local-server (http)

🌍 4. 环境变量配置
==================
设置测试环境变量:
  OPENMANUS_LLM_MODEL = gpt-4
  OPENMANUS_LLM_API_KEY = sk-test-key-from-env
  OPENMANUS_AGENT_MAX_STEPS = 15

--- 环境变量配置 ---
LLM 配置:
  模型: gpt-4
  基础URL: https://api.deepseek.com/v1
  API密钥: sk-t***-env
  温度: 0.1
  最大令牌: 4000
Agent 配置:
  最大步数: 15
  最大令牌: 8000
  最大持续时间: 5m0s
  反思步数: 3
MCP 服务器数量: 0

✅ 5. 配置验证
=============
✅ 有效配置验证通过
✅ 无效配置验证失败（预期）: 必须设置有效的 API Key

🔧 6. 配置使用示例
==================
配置使用示例:
  LLM 配置转换: deepseek-chat (温度: 0.1)
  工作目录: ./workspace
  MCP 服务器: 未配置
  预估令牌消耗: 8000 (基于最大步数)

💡 7. 配置最佳实践
==================
配置管理最佳实践:
  1. 🔐 永远不要在代码中硬编码 API Key
  2. 📄 使用配置文件管理复杂设置
  3. 🌍 使用环境变量处理敏感信息
  4. ✅ 启动时验证配置的完整性
  5. 📝 为配置项提供清晰的注释
  6. 🔄 支持配置热重载（生产环境）
  7. 🎯 根据环境（开发/测试/生产）使用不同配置
  8. 📊 监控配置变更和使用情况
  9. 🛡️  限制配置文件的访问权限
  10. 📋 提供配置模板和示例

🧹 已清理示例配置文件: example_config.toml

🎉 配置管理示例完成！

📚 学习总结:
  1. 默认配置提供基础设置
  2. 配置文件覆盖默认值
  3. 环境变量具有最高优先级
  4. 配置验证确保设置正确
  5. 合理使用配置提高灵活性
```

## 🔧 配置项详解

### LLM 配置
| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `model` | string | "deepseek-chat" | LLM 模型名称 |
| `base_url` | string | "https://api.deepseek.com/v1" | API 基础 URL |
| `api_key` | string | "your-api-key-here" | API 密钥 |
| `temperature` | float | 0.1 | 生成温度 (0.0-2.0) |
| `max_tokens` | int | 4000 | 最大令牌数 |

### Agent 配置
| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `max_steps` | int | 10 | 最大执行步数 |
| `max_tokens` | int | 8000 | 最大令牌数 |
| `max_duration` | string | "5m" | 最大执行时间 |
| `reflection_steps` | int | 3 | 反思步数间隔 |
| `max_retries` | int | 2 | 最大重试次数 |

### MCP 服务器配置
| 配置项 | 类型 | 说明 |
|--------|------|------|
| `name` | string | 服务器名称 |
| `transport` | string | 传输协议 (sse/http) |
| `url` | string | 服务器 URL |

## 📁 配置文件位置

推荐的配置文件位置：

```
# 开发环境
./configs/config.toml

# 生产环境
/etc/openmanus/config.toml
~/.config/openmanus/config.toml

# 容器环境
/app/configs/config.toml
```

## 🔍 配置验证

框架会在启动时验证配置的有效性：

### 必需配置
- `llm.api_key` - 必须设置有效的 API Key
- `llm.model` - 必须指定模型名称
- `agent.max_steps` - 必须大于 0

### 可选配置
- 所有工具配置都是可选的
- MCP 服务器配置是可选的
- 大部分参数都有合理的默认值

## 🐛 故障排除

**Q: 配置文件加载失败**
A: 检查文件路径和格式，确保是有效的 TOML 格式。

**Q: 环境变量不生效**
A: 确保环境变量名称正确，使用 `OPENMANUS_` 前缀。

**Q: API Key 验证失败**
A: 检查 API Key 是否正确，是否有访问权限。

**Q: 配置项不生效**
A: 检查配置优先级，环境变量会覆盖配置文件。

## 🔒 安全注意事项

1. **API Key 保护**
   ```bash
   # 使用环境变量
   export OPENMANUS_LLM_API_KEY="sk-your-key"
   
   # 设置文件权限
   chmod 600 configs/config.toml
   ```

2. **敏感信息处理**
   ```toml
   # 不要在配置文件中存储敏感信息
   [llm]
   api_key = "${OPENMANUS_LLM_API_KEY}"  # 使用环境变量引用
   ```

3. **生产环境配置**
   ```bash
   # 使用专用的配置目录
   mkdir -p /etc/openmanus
   chown openmanus:openmanus /etc/openmanus
   chmod 750 /etc/openmanus
   ```

## 📚 相关文档

- [MCP 集成示例](../../mcp/README.md)
- [多 Agent 示例](../../multi-agent/README.md)
- [配置参考文档](../../../docs/CONFIGURATION.md)
- [安全指南](../../../docs/SECURITY.md)

---

配置管理是使用 OpenManus-Go 的基础。掌握了配置系统，你就可以灵活地适应各种部署环境和使用场景！
