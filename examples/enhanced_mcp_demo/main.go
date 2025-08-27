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
	"openmanus-go/pkg/logger"
	"openmanus-go/pkg/tool"
	"openmanus-go/pkg/tool/builtin"
)

func main() {
	fmt.Println("🚀 OpenManus-Go Enhanced MCP Demo")
	fmt.Println("=====================================")

	// 初始化日志
	logger.InitWithConfig(logger.Config{
		Level:  "info",
		Output: "console",
	})

	// 加载配置
	cfg, err := config.Load("configs/config.toml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Printf("📋 Loaded configuration with %d MCP servers\n", len(cfg.MCP.Servers))
	for serverName := range cfg.MCP.Servers {
		fmt.Printf("  - %s\n", serverName)
	}

	// 创建 LLM 客户端
	llmClient := llm.NewOpenAIClient(cfg.ToLLMConfig())

	// 创建工具注册表
	toolRegistry := tool.NewRegistry()
	if err := builtin.RegisterBuiltinTools(toolRegistry, cfg); err != nil {
		log.Fatalf("Failed to register builtin tools: %v", err)
	}

	// 创建 Agent 配置
	agentConfig := &agent.Config{
		MaxSteps:        10,
		MaxTokens:       8000,
		MaxDuration:     5 * time.Minute,
		ReflectionSteps: 3,
		MaxRetries:      2,
		RetryBackoff:    time.Second,
	}

	// 创建增强 MCP Agent
	fmt.Println("\n🤖 Creating Enhanced MCP Agent...")
	mcpAgent := agent.NewBaseAgentWithMCP(llmClient, toolRegistry, agentConfig, cfg)

	// 等待 MCP 工具发现完成
	fmt.Println("⏳ Waiting for MCP tool discovery...")
	time.Sleep(3 * time.Second)

	// 测试用例
	testCases := []string{
		"查询苹果公司(AAPL)的股票价格",
		"获取今天的天气信息",
		"搜索最新的人工智能新闻",
	}

	ctx := context.Background()

	for i, testCase := range testCases {
		fmt.Printf("\n🎯 Test Case %d: %s\n", i+1, testCase)
		fmt.Println(strings.Repeat("-", 50))

		startTime := time.Now()
		result, err := mcpAgent.Loop(ctx, testCase)
		duration := time.Since(startTime)

		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		} else {
			fmt.Printf("✅ Result (%.2fs):\n%s\n", duration.Seconds(), result)
		}

		// 等待一下再执行下一个测试
		if i < len(testCases)-1 {
			time.Sleep(2 * time.Second)
		}
	}

	fmt.Println("\n🎉 Enhanced MCP Demo completed!")
	fmt.Println("\n💡 Tips:")
	fmt.Println("  - Check logs to see the intelligent tool selection process")
	fmt.Println("  - Try running with --debug flag for detailed MCP interactions")
	fmt.Println("  - Modify configs/config.toml to add more MCP servers")
}
