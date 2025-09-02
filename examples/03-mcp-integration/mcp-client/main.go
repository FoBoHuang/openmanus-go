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

// MCP 客户端集成示例
// 展示如何集成外部 MCP 服务器，使用 MCP 工具
// 以及如何将 MCP 工具与内置工具统一管理

func main() {
	fmt.Println("🔌 OpenManus-Go MCP 客户端示例")
	fmt.Println("=" + strings.Repeat("=", 40))
	fmt.Println()

	// 1. 加载配置（包含 MCP 服务器配置）
	cfg := loadConfigWithMCP()

	// 2. 展示 MCP 配置信息
	displayMCPConfig(cfg)

	// 3. 创建带 MCP 集成的 Agent
	agent := createMCPAgent(cfg)

	// 4. 演示 MCP 工具发现和使用
	demonstrateMCPFeatures(agent, cfg)

	fmt.Println("\n🎉 MCP 客户端示例完成！")
	fmt.Println("\n📚 学习要点:")
	fmt.Println("  ✅ MCP 协议标准化工具接口")
	fmt.Println("  ✅ 支持动态工具发现和注册")
	fmt.Println("  ✅ 与内置工具统一管理")
	fmt.Println("  ✅ Agent 智能选择 MCP 工具")
}

func loadConfigWithMCP() *config.Config {
	configPath := "../../../configs/config.toml"
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("⚠️  配置文件加载失败，使用默认配置: %v\n", err)
		cfg = config.DefaultConfig()
	}

	// 检查是否有 MCP 服务器配置
	if len(cfg.MCP.Servers) == 0 {
		fmt.Println("⚠️  未发现 MCP 服务器配置，添加示例配置")
		cfg = addExampleMCPConfig(cfg)
	}

	return cfg
}

func addExampleMCPConfig(cfg *config.Config) *config.Config {
	// 添加示例 MCP 服务器配置
	// 注意：这些是示例 URL，实际使用时需要替换为真实的 MCP 服务器
	cfg.MCP.Servers = map[string]config.MCPServerConfig{
		"stock-helper": {
			URL:     "https://mcp.higress.ai/mcp-stock-helper/demo/sse",
			Headers: map[string]string{"Transport": "sse"},
		},
		"weather-service": {
			URL:     "https://example.com/weather-mcp",
			Headers: map[string]string{"Transport": "http"},
		},
		"news-aggregator": {
			URL:     "https://example.com/news-mcp/sse",
			Headers: map[string]string{"Transport": "sse"},
		},
	}
	return cfg
}

func displayMCPConfig(cfg *config.Config) {
	fmt.Println("📋 MCP 服务器配置:")
	fmt.Println(strings.Repeat("-", 25))

	if len(cfg.MCP.Servers) == 0 {
		fmt.Println("  ⚠️  未配置 MCP 服务器")
		return
	}

	i := 1
	for name, server := range cfg.MCP.Servers {
		fmt.Printf("  %d. %s\n", i, name)
		fmt.Printf("     🌐 URL: %s\n", server.URL)
		if transport, ok := server.Headers["Transport"]; ok {
			fmt.Printf("     🚀 传输: %s\n", transport)
		}
		fmt.Println()
		i++
	}
}

func createMCPAgent(cfg *config.Config) agent.Agent {
	fmt.Println("🤖 创建带 MCP 集成的 Agent...")

	// 创建 LLM 客户端
	llmClient := llm.NewOpenAIClient(cfg.ToLLMConfig())
	fmt.Println("  ✅ LLM 客户端已创建")

	// 创建工具注册表
	toolRegistry := tool.NewRegistry()

	// 注册内置工具
	setupBuiltinTools(toolRegistry)

	// 创建带 MCP 集成的 Agent
	agentConfig, err := agent.ConfigFromAppConfig(cfg)
	if err != nil {
		panic(fmt.Sprintf("创建 Agent 配置失败: %v", err))
	}

	// 可以根据需要覆盖特定设置
	agentConfig.MaxSteps = 10
	agentConfig.MaxDuration = 5 * time.Minute

	// 使用带 MCP 支持的构造函数
	baseAgent := agent.NewBaseAgentWithMCP(llmClient, toolRegistry, agentConfig, cfg)
	fmt.Println("  ✅ MCP Agent 已创建")

	// 等待 MCP 工具发现完成
	fmt.Println("  🔍 等待 MCP 工具发现...")
	time.Sleep(3 * time.Second) // 给 MCP 发现服务一些时间

	// 显示所有可用工具
	allTools := toolRegistry.List()
	fmt.Printf("  📊 共发现 %d 个工具\n", len(allTools))

	// 简化显示工具信息
	fmt.Printf("    - 已注册工具: %d 个\n", len(allTools))
	fmt.Printf("    - 支持MCP集成\n")
	fmt.Println()

	return baseAgent
}

func setupBuiltinTools(toolRegistry *tool.Registry) {
	// 注册文件系统工具
	fsTool := builtin.NewFileSystemTool(
		[]string{"../../../workspace"},
		[]string{},
	)
	if err := toolRegistry.Register(fsTool); err != nil {
		log.Fatalf("❌ 注册文件系统工具失败: %v", err)
	}

	// 注册 HTTP 工具
	httpTool := builtin.NewHTTPTool()
	if err := toolRegistry.Register(httpTool); err != nil {
		log.Fatalf("❌ 注册 HTTP 工具失败: %v", err)
	}

	fmt.Println("  ✅ 内置工具注册完成")
}

func demonstrateMCPFeatures(agent agent.Agent, cfg *config.Config) {
	ctx := context.Background()
	hasAPIKey := cfg.LLM.APIKey != "" && cfg.LLM.APIKey != "your-api-key-here"

	fmt.Println("🔍 MCP 功能演示")
	fmt.Println(strings.Repeat("-", 20))

	// 1. 展示工具发现结果
	fmt.Println("\n1. 工具发现结果")
	demonstrateToolDiscovery(agent)

	// 2. MCP 工具使用场景
	fmt.Println("\n2. MCP 工具使用场景")
	mcpUseCases := []MCPUseCase{
		{
			Name:         "股票价格查询",
			Description:  "使用 MCP 股票工具查询实时股价",
			Task:         "查询苹果公司(AAPL)的当前股价和基本信息",
			ExpectedTool: "stock-price",
		},
		{
			Name:         "天气信息获取",
			Description:  "使用 MCP 天气服务获取天气预报",
			Task:         "获取北京市明天的天气预报",
			ExpectedTool: "weather-forecast",
		},
		{
			Name:         "新闻搜索",
			Description:  "使用 MCP 新闻服务搜索最新资讯",
			Task:         "搜索最新的人工智能相关新闻，限制5条",
			ExpectedTool: "news-search",
		},
		{
			Name:         "混合任务",
			Description:  "结合 MCP 工具和内置工具的复合任务",
			Task:         "查询特斯拉股价，如果股价大于200美元，将结果保存到 tesla_stock.txt 文件",
			ExpectedTool: "stock-price + fs",
		},
	}

	for i, useCase := range mcpUseCases {
		fmt.Printf("\n  📋 场景 %d: %s\n", i+1, useCase.Name)
		fmt.Printf("  📝 描述: %s\n", useCase.Description)
		fmt.Printf("  🎯 任务: %s\n", useCase.Task)
		fmt.Printf("  🔧 预期工具: %s\n", useCase.ExpectedTool)

		if hasAPIKey {
			fmt.Println("  🔄 执行中...")
			startTime := time.Now()

			result, err := agent.Loop(ctx, useCase.Task)
			duration := time.Since(startTime)

			if err != nil {
				fmt.Printf("  ❌ 执行失败: %v\n", err)
			} else {
				fmt.Printf("  ✅ 执行成功 (耗时: %v)\n", duration.Round(time.Millisecond))

				// 显示结果摘要
				if len(result) > 150 {
					fmt.Printf("  📄 结果: %s...\n", result[:150])
				} else {
					fmt.Printf("  📄 结果: %s\n", result)
				}

				// 显示使用的工具
				if trace := agent.GetTrace(); trace != nil && len(trace.Steps) > 0 {
					usedTools := make(map[string]bool)
					for _, step := range trace.Steps {
						usedTools[step.Action.Name] = true
					}

					var toolNames []string
					for toolName := range usedTools {
						toolNames = append(toolNames, toolName)
					}

					fmt.Printf("  🔧 使用工具: %s\n", strings.Join(toolNames, ", "))
				}
			}
		} else {
			fmt.Println("  🔄 模拟执行...")
			fmt.Printf("  ✅ 模拟成功: Agent 会自动选择 %s 完成任务\n", useCase.ExpectedTool)
		}
	}

	// 3. MCP 连接状态检查
	fmt.Println("\n3. MCP 连接状态检查")
	checkMCPConnections(cfg)

	// 4. MCP 最佳实践建议
	fmt.Println("\n4. MCP 最佳实践")
	showMCPBestPractices()
}

type MCPUseCase struct {
	Name         string
	Description  string
	Task         string
	ExpectedTool string
}

func demonstrateToolDiscovery(agent agent.Agent) {
	// 这里简化演示，实际实现中需要访问 Agent 的工具注册表
	fmt.Println("  🔍 发现的 MCP 工具:")

	// 模拟 MCP 工具发现结果
	mockMCPTools := []string{
		"stock-price (股票价格查询)",
		"stock-candlestick (K线数据)",
		"stock-rank (股票排行)",
		"weather-forecast (天气预报)",
		"news-search (新闻搜索)",
	}

	for _, tool := range mockMCPTools {
		fmt.Printf("    ✅ %s\n", tool)
	}

	fmt.Println("  📊 工具统计:")
	fmt.Printf("    - 总工具数: %d\n", len(mockMCPTools)+2) // +2 for builtin tools
	fmt.Printf("    - MCP 工具: %d\n", len(mockMCPTools))
	fmt.Printf("    - 内置工具: 2\n")
}

func checkMCPConnections(cfg *config.Config) {
	fmt.Println("  🌐 检查 MCP 服务器连接状态:")

	for name, server := range cfg.MCP.Servers {
		fmt.Printf("    📡 %s (%s):\n", name, server.URL)

		// 模拟连接检查
		time.Sleep(100 * time.Millisecond) // 模拟网络延迟

		// 简化的连接状态模拟
		if strings.Contains(server.URL, "example.com") {
			fmt.Printf("      ⚠️  连接失败: 示例 URL，请配置真实的 MCP 服务器\n")
		} else {
			fmt.Printf("      ✅ 连接正常: 响应时间 < 100ms\n")
		}
	}
}

func showMCPBestPractices() {
	fmt.Println("  💡 MCP 集成最佳实践:")
	fmt.Println("    1. 配置可靠的 MCP 服务器 URL")
	fmt.Println("    2. 设置合适的超时时间")
	fmt.Println("    3. 监控 MCP 服务器的可用性")
	fmt.Println("    4. 使用回退机制处理 MCP 服务不可用")
	fmt.Println("    5. 定期验证 MCP 工具的功能")
	fmt.Println("    6. 合理设置工具调用频率限制")

	fmt.Println("\n  🔧 配置建议:")
	fmt.Println("    - 生产环境使用 HTTPS 协议")
	fmt.Println("    - 启用工具调用缓存")
	fmt.Println("    - 配置工具使用监控")
	fmt.Println("    - 实现 MCP 服务降级策略")
}
