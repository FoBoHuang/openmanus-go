package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"openmanus-go/pkg/agent"
	"openmanus-go/pkg/config"
	"openmanus-go/pkg/llm"
	"openmanus-go/pkg/tool"
	"openmanus-go/pkg/tool/builtin"
)

// 基础任务示例
// 展示 OpenManus-Go Agent 执行各种基础任务的能力
// 包括文件操作、网络请求、数据处理等常见场景

func main() {
	fmt.Println("📋 OpenManus-Go 基础任务示例")
	fmt.Println("=" + strings.Repeat("=", 35))
	fmt.Println()

	// 1. 初始化配置和组件
	cfg := setupConfig()
	llmClient := llm.NewOpenAIClient(cfg.ToLLMConfig())
	toolRegistry := setupTools()
	agent := setupAgent(llmClient, toolRegistry)

	hasAPIKey := cfg.LLM.APIKey != "" && cfg.LLM.APIKey != "your-api-key-here"

	if !hasAPIKey {
		fmt.Println("ℹ️  运行在演示模式（未设置 API Key）")
		fmt.Println("   设置 API Key 后可体验完整的智能任务执行")
		fmt.Println()
	}

	// 2. 定义各类基础任务
	taskCategories := []TaskCategory{
		{
			Name:        "文件操作任务",
			Description: "文件和目录的创建、读取、修改操作",
			Tasks: []Task{
				{
					Description: "创建项目目录结构",
					Goal:        "在 workspace 中创建一个名为 'project' 的目录，并在其中创建 'src', 'docs', 'tests' 三个子目录",
					Expected:    "目录结构创建成功",
				},
				{
					Description: "生成配置文件",
					Goal:        "创建一个名为 config.json 的配置文件，包含应用名称、版本和作者信息",
					Expected:    "JSON 配置文件创建完成",
				},
				{
					Description: "文件内容处理",
					Goal:        "读取刚创建的 config.json 文件，验证内容是否正确",
					Expected:    "文件内容验证成功",
				},
			},
		},
		{
			Name:        "网络请求任务",
			Description: "HTTP 请求、API 调用、数据获取",
			Tasks: []Task{
				{
					Description: "获取公共 API 数据",
					Goal:        "从 https://httpbin.org/json 获取示例 JSON 数据",
					Expected:    "成功获取 JSON 响应",
				},
				{
					Description: "检查网站状态",
					Goal:        "检查 https://httpbin.org 网站的可用性和响应时间",
					Expected:    "网站状态检查完成",
				},
				{
					Description: "保存网络数据",
					Goal:        "获取 https://httpbin.org/uuid 的响应并保存到 uuid.txt 文件",
					Expected:    "网络数据保存成功",
				},
			},
		},
		{
			Name:        "数据处理任务",
			Description: "数据格式转换、内容分析、信息提取",
			Tasks: []Task{
				{
					Description: "时间戳处理",
					Goal:        "创建一个包含当前时间戳的报告文件 timestamp_report.txt",
					Expected:    "时间戳报告生成完成",
				},
				{
					Description: "文件清单生成",
					Goal:        "扫描 workspace 目录，生成一个详细的文件清单 file_inventory.txt",
					Expected:    "文件清单生成完成",
				},
				{
					Description: "简单统计分析",
					Goal:        "统计 workspace 目录中的文件数量和总大小，保存到 stats.txt",
					Expected:    "统计分析完成",
				},
			},
		},
	}

	// 3. 执行任务演示
	ctx := context.Background()
	totalTasks := 0
	successTasks := 0

	for categoryIndex, category := range taskCategories {
		fmt.Printf("📂 %d. %s\n", categoryIndex+1, category.Name)
		fmt.Printf("   📝 %s\n", category.Description)
		fmt.Println()

		for taskIndex, task := range category.Tasks {
			totalTasks++
			fmt.Printf("   📋 任务 %d.%d: %s\n", categoryIndex+1, taskIndex+1, task.Description)
			fmt.Printf("   🎯 目标: %s\n", task.Goal)

			if hasAPIKey {
				// 实际执行任务
				fmt.Println("   🔄 执行中...")
				startTime := time.Now()

				result, err := agent.Loop(ctx, task.Goal)
				duration := time.Since(startTime)

				if err != nil {
					fmt.Printf("   ❌ 执行失败: %v\n", err)
				} else {
					successTasks++
					fmt.Printf("   ✅ 执行成功 (耗时: %v)\n", duration.Round(time.Millisecond))

					// 显示执行结果摘要
					if result != "" {
						// 限制输出长度
						if len(result) > 100 {
							fmt.Printf("   📄 结果: %s...\n", result[:100])
						} else {
							fmt.Printf("   📄 结果: %s\n", result)
						}
					}

					// 显示执行轨迹信息
					if trace := agent.GetTrace(); trace != nil && len(trace.Steps) > 0 {
						fmt.Printf("   🔍 执行步骤: %d 步\n", len(trace.Steps))
					}
				}
			} else {
				// 演示模式
				fmt.Println("   🔄 模拟执行...")
				time.Sleep(500 * time.Millisecond) // 模拟执行时间

				successTasks++
				fmt.Printf("   ✅ 模拟完成: %s\n", task.Expected)
				fmt.Println("   💭 在真实模式下，Agent 会智能选择工具完成此任务")
			}

			fmt.Println()
		}

		fmt.Println(strings.Repeat("-", 60))
		fmt.Println()
	}

	// 4. 执行总结
	fmt.Println("📊 执行总结")
	fmt.Println("=" + strings.Repeat("=", 15))
	fmt.Printf("📋 总任务数: %d\n", totalTasks)
	fmt.Printf("✅ 成功任务: %d\n", successTasks)
	fmt.Printf("📈 成功率: %.1f%%\n", float64(successTasks)/float64(totalTasks)*100)
	fmt.Println()

	// 5. 任务类型分析
	fmt.Println("🎯 任务类型分析:")
	for i, category := range taskCategories {
		fmt.Printf("  %d. %s: %d 个任务\n", i+1, category.Name, len(category.Tasks))
	}
	fmt.Println()

	// 6. 框架能力展示
	fmt.Println("🛠️  框架能力展示:")
	fmt.Println("  ✅ 自动任务分解和执行")
	fmt.Println("  ✅ 智能工具选择和调用")
	fmt.Println("  ✅ 错误处理和容错机制")
	fmt.Println("  ✅ 执行轨迹记录和分析")
	fmt.Println("  ✅ 多类型任务统一处理")
	fmt.Println()

	// 7. 下一步建议
	fmt.Println("📚 学习建议:")
	if !hasAPIKey {
		fmt.Println("  1. 设置 LLM API Key 体验完整功能")
		fmt.Println("  2. 观察 Agent 的智能决策过程")
	}
	fmt.Println("  3. 查看 workspace 目录验证任务结果")
	fmt.Println("  4. 尝试修改任务描述测试不同场景")
	fmt.Println("  5. 学习 ../../02-tool-usage/ 了解工具详情")
	fmt.Println()

	fmt.Println("🎉 基础任务示例完成！")
}

// TaskCategory 任务类别
type TaskCategory struct {
	Name        string
	Description string
	Tasks       []Task
}

// Task 单个任务
type Task struct {
	Description string
	Goal        string
	Expected    string
}

// setupConfig 设置配置
func setupConfig() *config.Config {
	configPath := "../../../configs/config.toml"
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("⚠️  配置文件加载失败，使用默认配置: %v\n", err)
		cfg = config.DefaultConfig()
	}

	fmt.Printf("✅ 配置加载完成 (模型: %s)\n", cfg.LLM.Model)
	return cfg
}

// setupTools 设置工具
func setupTools() *tool.Registry {
	toolRegistry := tool.NewRegistry()

	// 注册文件系统工具
	fsTool := builtin.NewFileSystemTool(
		[]string{"../../../workspace"}, // 允许访问 workspace 目录
		[]string{},                     // 无禁止路径
	)
	if err := toolRegistry.Register(fsTool); err != nil {
		log.Fatalf("❌ 注册文件系统工具失败: %v", err)
	}

	// 注册 HTTP 工具
	httpTool := builtin.NewHTTPTool()
	if err := toolRegistry.Register(httpTool); err != nil {
		log.Fatalf("❌ 注册 HTTP 工具失败: %v", err)
	}

	// 注册爬虫工具
	crawlerTool := builtin.NewCrawlerTool("OpenManus-Go-Example/1.0", []string{}, []string{})
	if err := toolRegistry.Register(crawlerTool); err != nil {
		log.Fatalf("❌ 注册爬虫工具失败: %v", err)
	}

	tools := toolRegistry.List()
	fmt.Printf("✅ 工具注册完成 (%d 个工具)\n", len(tools))

	return toolRegistry
}

// setupAgent 设置 Agent
func setupAgent(llmClient llm.Client, toolRegistry *tool.Registry) agent.Agent {
	agentConfig := agent.DefaultConfig()
	agentConfig.MaxSteps = 8 // 适合基础任务的步数
	agentConfig.MaxDuration = 3 * time.Minute
	agentConfig.ReflectionSteps = 3

	baseAgent := agent.NewBaseAgent(llmClient, toolRegistry, agentConfig)
	fmt.Printf("✅ Agent 创建完成 (最大步数: %d)\n", agentConfig.MaxSteps)
	fmt.Println()

	return baseAgent
}
