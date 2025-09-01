# 使用示例

本文档提供了丰富的 OpenManus-Go 使用示例，从基础操作到复杂的实际应用场景，帮助您快速掌握框架的使用方法。

## 📋 目录

- [基础示例](#基础示例)
- [文件操作示例](#文件操作示例)
- [网络操作示例](#网络操作示例)
- [数据处理示例](#数据处理示例)
- [自动化任务示例](#自动化任务示例)
- [集成示例](#集成示例)
- [实际应用场景](#实际应用场景)

## 🚀 基础示例

### Hello World

最简单的入门示例：

```bash
# 交互模式
./bin/openmanus run --config configs/config.toml --interactive

# 输入任务
🎯 Goal: 创建一个包含"Hello, World!"的测试文件
```

程序化使用：

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "openmanus-go/pkg/agent"
    "openmanus-go/pkg/config"
    "openmanus-go/pkg/llm"
    "openmanus-go/pkg/tool"
)

func main() {
    // 加载配置
    cfg, err := config.LoadConfig("configs/config.toml")
    if err != nil {
        log.Fatal(err)
    }
    
    // 创建组件
    llmClient := llm.NewOpenAIClient(cfg.ToLLMConfig())
    toolRegistry := tool.DefaultRegistry
    
    // 创建 Agent
    agent := agent.NewBaseAgent(llmClient, toolRegistry, nil)
    
    // 执行任务
    ctx := context.Background()
    trace, err := agent.Loop(ctx, "创建一个包含当前时间的hello.txt文件")
    if err != nil {
        log.Fatalf("执行失败: %v", err)
    }
    
    fmt.Printf("任务完成: %s\n", trace.Status)
}
```

### 配置验证

```bash
# 验证配置
./bin/openmanus config validate --config configs/config.toml

# 测试 LLM 连接
./bin/openmanus config test-llm --config configs/config.toml

# 查看可用工具
./bin/openmanus tools list --config configs/config.toml
```

## 📁 文件操作示例

### 文件创建和管理

```bash
# 创建文件
🎯 Goal: 在workspace目录创建一个名为report.txt的文件，内容为今天的日期和时间

# 批量文件操作
🎯 Goal: 创建5个测试文件，文件名为test1.txt到test5.txt，每个文件包含不同的内容

# 文件备份
🎯 Goal: 备份workspace目录下的所有.txt文件到backup文件夹

# 文件清理
🎯 Goal: 删除workspace目录下所有空文件和临时文件
```

### 目录结构分析

```bash
# 目录分析
🎯 Goal: 分析当前项目目录结构，生成一个详细的文件清单报告

# 大文件查找
🎯 Goal: 找出workspace目录下所有大于1MB的文件，并生成统计报告

# 重复文件检测
🎯 Goal: 检查workspace目录中是否有重复的文件，如果有则列出来
```

### 配置文件处理

```go
// 示例：批量处理配置文件
func processConfigFiles() {
    goal := `
    检查configs目录下的所有.toml文件：
    1. 验证TOML语法正确性
    2. 检查必需的配置项是否存在
    3. 生成配置文件状态报告
    `
    
    agent.Loop(context.Background(), goal)
}
```

## 🌐 网络操作示例

### HTTP 请求

```bash
# 简单 API 调用
🎯 Goal: 获取https://httpbin.org/json的内容并保存到data.json文件

# 多个 API 调用
🎯 Goal: 分别调用httpbin.org的/json、/ip、/user-agent接口，将结果合并保存到api_results.json

# API 数据处理
🎯 Goal: 从GitHub API获取golang/go仓库的信息，提取star数、fork数等关键信息保存到文件
```

### 网页数据抓取

```bash
# 网页标题抓取
🎯 Goal: 抓取news.ycombinator.com首页的所有文章标题，保存到hacker_news.txt

# 电商数据抓取
🎯 Goal: 从电商网站抓取特定商品的价格信息，生成价格监控报告

# 新闻聚合
🎯 Goal: 从多个新闻网站抓取今日头条，汇总成新闻摘要
```

### REST API 集成

```go
// 示例：集成外部API服务
func integrateWeatherAPI() {
    goal := `
    集成天气API服务：
    1. 调用天气API获取北京今日天气
    2. 解析JSON响应数据
    3. 格式化天气信息
    4. 保存到weather_report.txt文件
    `
    
    agent.Loop(context.Background(), goal)
}
```

## 📊 数据处理示例

### CSV 数据分析

```bash
# CSV 文件分析
🎯 Goal: 分析workspace/sales.csv文件，计算总销售额、平均值、最高和最低销售额

# 数据清洗
🎯 Goal: 清理customers.csv文件中的重复数据和无效数据，生成干净的数据文件

# 数据聚合
🎯 Goal: 将多个月度销售CSV文件合并，按产品类别生成年度销售报告
```

### JSON 数据处理

```bash
# JSON 转换
🎯 Goal: 将products.json文件转换为CSV格式，保持所有字段信息

# 数据提取
🎯 Goal: 从复杂的JSON文件中提取特定字段，创建简化版本

# 数据验证
🎯 Goal: 验证JSON文件的数据完整性，检查必需字段是否存在
```

### 日志分析

```bash
# 日志统计
🎯 Goal: 分析nginx访问日志，统计访问量最高的10个IP地址

# 错误日志分析
🎯 Goal: 从应用日志中提取所有错误信息，按错误类型分类统计

# 性能分析
🎯 Goal: 分析API响应时间日志，找出响应最慢的10个接口
```

## 🤖 自动化任务示例

### 定期报告生成

```go
// 示例：自动生成日报
func generateDailyReport() {
    goal := `
    生成今日工作报告：
    1. 检查workspace目录中今日创建的文件
    2. 统计代码提交次数（如果有git仓库）
    3. 收集系统性能数据
    4. 生成格式化的日报文件
    `
    
    agent.Loop(context.Background(), goal)
}
```

### 系统监控

```bash
# 磁盘空间检查
🎯 Goal: 检查当前磁盘使用情况，如果使用率超过80%则生成警报报告

# 服务状态监控
🎯 Goal: 检查Redis和MySQL服务是否正常运行，生成服务状态报告

# 文件同步
🎯 Goal: 将workspace目录的文件同步到backup目录，只同步修改过的文件
```

### 批量操作

```bash
# 图片处理
🎯 Goal: 批量压缩images目录下的所有图片，将压缩后的图片保存到optimized目录

# 文档转换
🎯 Goal: 将docs目录下的所有Markdown文件转换为HTML格式

# 批量重命名
🎯 Goal: 将music目录下的所有音乐文件重命名为"艺术家-歌曲名"格式
```

## 🔗 集成示例

### 数据库操作

#### Redis 操作

```bash
# 缓存管理
🎯 Goal: 将user_data.json文件的内容存储到Redis中，设置1小时过期时间

# 数据统计
🎯 Goal: 从Redis中获取所有用户会话数据，统计在线用户数量

# 缓存清理
🎯 Goal: 清理Redis中所有过期的缓存键，生成清理报告
```

#### MySQL 操作

```bash
# 数据导入
🎯 Goal: 将products.csv文件的数据导入到MySQL的products表中

# 报表生成
🎯 Goal: 从MySQL数据库查询销售数据，生成月度销售报表

# 数据备份
🎯 Goal: 导出MySQL中的用户表数据到CSV文件进行备份
```

### 浏览器自动化

```bash
# 表单自动填写
🎯 Goal: 打开指定网站的注册页面，自动填写测试数据并截图

# 页面监控
🎯 Goal: 定期检查网站首页是否正常加载，记录加载时间

# 数据抓取
🎯 Goal: 自动登录后台管理系统，抓取最新的统计数据
```

### MCP 服务集成

```go
// 示例：集成股票查询服务
func queryStockPrices() {
    goal := `
    查询股票信息：
    1. 连接股票查询MCP服务
    2. 查询苹果公司(AAPL)的实时股价
    3. 获取最近5日的价格走势
    4. 生成股价分析报告
    `
    
    agent.Loop(context.Background(), goal)
}
```

## 🏗️ 实际应用场景

### 场景1：数据收集和分析流水线

```bash
# 完整的数据流水线
🎯 Goal: 
执行完整的数据分析流程：
1. 从API获取销售数据
2. 清洗和验证数据
3. 与历史数据进行对比
4. 生成趋势分析图表
5. 创建详细的分析报告
6. 将结果发送到指定邮箱
```

### 场景2：内容管理系统

```bash
# 博客内容管理
🎯 Goal:
管理博客内容：
1. 扫描content目录中的新Markdown文件
2. 提取文章标题、标签和摘要
3. 生成文章索引
4. 检查图片链接的有效性
5. 生成静态网站文件
6. 更新RSS订阅源
```

### 场景3：系统运维自动化

```bash
# 服务器维护
🎯 Goal:
执行服务器日常维护：
1. 检查系统资源使用情况
2. 清理临时文件和日志
3. 更新软件包（模拟）
4. 备份重要配置文件
5. 运行健康检查
6. 生成维护报告
```

### 场景4：电商数据监控

```go
// 示例：电商价格监控系统
func ecommerceMonitoring() {
    goal := `
    电商价格监控系统：
    1. 爬取竞争对手网站的产品价格
    2. 与我们的产品价格进行对比
    3. 识别价格变化趋势
    4. 生成价格调整建议
    5. 创建竞争分析报告
    6. 如果发现显著价格变化，生成警报
    `
    
    agent.Loop(context.Background(), goal)
}
```

### 场景5：社交媒体管理

```bash
# 社媒内容分析
🎯 Goal:
社交媒体内容管理：
1. 分析最近发布的社媒内容表现
2. 统计点赞、分享、评论数据
3. 识别表现最好的内容类型
4. 生成内容策略建议
5. 规划下周的发布内容
6. 创建内容日历
```

### 场景6：学术研究辅助

```bash
# 论文资料整理
🎯 Goal:
学术研究辅助：
1. 搜索和下载相关论文PDF
2. 提取论文标题、作者、摘要
3. 分析研究趋势和热点
4. 生成文献综述大纲
5. 创建参考文献格式化列表
6. 建立研究资料数据库
```

### 场景7：财务报表自动化

```go
// 示例：财务报表生成
func generateFinancialReport() {
    goal := `
    财务报表自动化：
    1. 从ERP系统导出财务数据
    2. 验证数据完整性和准确性
    3. 计算关键财务指标
    4. 生成利润表和资产负债表
    5. 创建图表和可视化分析
    6. 生成管理层报告
    7. 自动发送给相关人员
    `
    
    agent.Loop(context.Background(), goal)
}
```

## 🎛️ 高级使用技巧

### 链式任务执行

```bash
# 多步骤复杂任务
🎯 Goal:
执行复杂的数据处理链：
第一步：从多个源收集数据（API、文件、数据库）
第二步：数据清洗和标准化
第三步：数据分析和统计
第四步：生成可视化图表
第五步：创建综合报告
最后：分发报告给相关人员
```

### 条件逻辑处理

```bash
# 条件执行
🎯 Goal:
智能文件处理：
1. 检查workspace目录中的文件
2. 如果发现CSV文件，进行数据分析
3. 如果发现图片文件，进行压缩处理
4. 如果发现日志文件，进行错误分析
5. 根据文件类型生成不同的处理报告
```

### 错误处理和重试

```bash
# 健壮性处理
🎯 Goal:
网络数据获取（带错误处理）：
1. 尝试从主API获取数据
2. 如果失败，尝试备用API
3. 如果都失败，使用本地缓存数据
4. 记录所有尝试过程和错误信息
5. 生成数据获取状态报告
```

### 并行任务处理

```bash
# 并行执行
🎯 Goal:
并行处理多个数据源：
1. 同时从3个不同API获取数据
2. 并行处理多个CSV文件
3. 同步将结果汇总
4. 生成综合分析报告
```

## 📝 示例运行指南

### 准备工作

1. **配置环境**：
   ```bash
   # 复制配置文件
   cp configs/config.example.toml configs/config.toml
   
   # 设置API Key
   export OPENMANUS_API_KEY="your-api-key"
   ```

2. **准备测试数据**：
   ```bash
   # 创建测试目录
   mkdir -p workspace/test-data
   
   # 创建示例CSV文件
   echo "name,age,city" > workspace/test-data/users.csv
   echo "张三,25,北京" >> workspace/test-data/users.csv
   echo "李四,30,上海" >> workspace/test-data/users.csv
   ```

### 运行示例

1. **交互模式**（推荐新手）：
   ```bash
   ./bin/openmanus run --config configs/config.toml --interactive
   ```

2. **单命令模式**：
   ```bash
   ./bin/openmanus run --config configs/config.toml "你的任务描述"
   ```

3. **程序集成**：
   ```go
   // 在你的Go程序中使用
   trace, err := agent.Loop(ctx, "任务描述")
   ```

### 查看结果

```bash
# 查看生成的文件
ls -la workspace/

# 查看执行轨迹
cat data/traces/latest.json

# 查看日志
tail -f logs/openmanus.log
```

### 性能监控

```bash
# 查看Agent状态
./bin/openmanus tools test --config configs/config.toml

# 监控资源使用
top -p $(pgrep openmanus)

# 查看网络连接
netstat -tlnp | grep openmanus
```

## 🔧 故障排除

### 常见问题

1. **任务执行失败**：
   - 检查API Key是否正确
   - 验证网络连接
   - 查看详细日志

2. **文件权限错误**：
   - 检查文件路径配置
   - 确认目录访问权限
   - 查看文件系统工具配置

3. **工具调用失败**：
   - 验证工具配置
   - 检查依赖服务状态
   - 查看工具列表

### 调试技巧

```bash
# 启用详细日志
./bin/openmanus run --config configs/config.toml --verbose --debug "任务"

# 查看工具状态
./bin/openmanus tools list --config configs/config.toml

# 测试特定工具
./bin/openmanus tools test --name fs --config configs/config.toml
```

---

这些示例涵盖了 OpenManus-Go 的各种使用场景，从简单的文件操作到复杂的业务流程自动化。通过学习这些示例，您可以快速掌握如何利用 AI Agent 来自动化各种任务！

**相关文档**: [快速入门](QUICK_START.md) → [核心概念](CONCEPTS.md) → [最佳实践](BEST_PRACTICES.md)
