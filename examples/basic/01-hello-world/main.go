package main

import (
	"context"
	"fmt"
	"log"

	"openmanus-go/pkg/agent"
	"openmanus-go/pkg/config"
	"openmanus-go/pkg/llm"
	"openmanus-go/pkg/tool"
	"openmanus-go/pkg/tool/builtin"
)

// Hello World 示例
// 这是 OpenManus-Go 框架的最简单使用示例
// 展示如何创建一个基础的 Agent 并执行简单任务

func main() {
	fmt.Println("🚀 OpenManus-Go Hello World Example")
	fmt.Println("====================================")
	fmt.Println()

	// 1. 加载配置
	// 使用默认配置，在实际使用中应该从配置文件加载
	cfg := config.DefaultConfig()

	// 注意：在实际使用中，请在 configs/config.toml 中设置真实的 API Key
	// 这里使用占位符只是为了演示代码结构
	if cfg.LLM.APIKey == "" || cfg.LLM.APIKey == "your-api-key-here" {
		fmt.Println("⚠️  警告：未设置 LLM API Key")
		fmt.Println("请在 configs/config.toml 中设置正确的 api_key")
		fmt.Println()
		fmt.Println("示例配置：")
		fmt.Println("[llm]")
		fmt.Println(`model = "deepseek-chat"`)
		fmt.Println(`base_url = "https://api.deepseek.com/v1"`)
		fmt.Println(`api_key = "your-actual-api-key"`)
		fmt.Println()

		// 在没有真实 API Key 的情况下，我们仍然可以展示框架的基本结构
		fmt.Println("📝 继续演示框架结构（不会进行实际的 LLM 调用）...")
		fmt.Println()
	}

	// 2. 创建 LLM 客户端
	// 这是与大语言模型通信的客户端
	llmClient := llm.NewOpenAIClient(cfg.ToLLMConfig())
	fmt.Println("✅ LLM 客户端已创建")

	// 3. 创建工具注册表
	// 工具注册表管理所有可用的工具
	toolRegistry := tool.NewRegistry()
	fmt.Println("✅ 工具注册表已创建")

	// 4. 注册基础工具
	// 注册一个简单的文件系统工具用于演示
	fsTool := builtin.NewFileSystemTool(
		[]string{"./workspace"}, // 允许访问的路径
		[]string{},              // 禁止访问的路径
	)

	if err := toolRegistry.Register(fsTool); err != nil {
		log.Fatalf("❌ 注册文件系统工具失败: %v", err)
	}
	fmt.Println("✅ 文件系统工具已注册")

	// 5. 创建 Agent
	// Agent 是执行任务的核心组件
	agentConfig := agent.DefaultConfig()
	agentConfig.MaxSteps = 3 // 限制最大步数，适合简单任务

	baseAgent := agent.NewBaseAgent(llmClient, toolRegistry, agentConfig)
	fmt.Println("✅ Agent 已创建")
	fmt.Println()

	// 6. 定义简单任务
	// 这里定义一些不需要复杂 LLM 调用的简单任务
	tasks := []string{
		"创建一个名为 hello.txt 的文件，内容为 'Hello, OpenManus-Go!'",
		"检查 hello.txt 文件是否存在",
	}

	// 7. 执行任务演示
	ctx := context.Background()

	for i, task := range tasks {
		fmt.Printf("📋 任务 %d: %s\n", i+1, task)
		fmt.Println("------------------------------------")

		// 在没有真实 API Key 的情况下，我们模拟任务执行
		if cfg.LLM.APIKey == "" || cfg.LLM.APIKey == "your-api-key-here" {
			fmt.Println("🔄 模拟执行中...")
			fmt.Println("💭 Agent 思考：需要使用文件系统工具")
			fmt.Println("🔧 工具调用：fs(operation='write', path='workspace/hello.txt', content='Hello, OpenManus-Go!')")
			fmt.Println("✅ 模拟结果：文件创建成功")
		} else {
			// 实际执行任务
			result, err := baseAgent.Loop(ctx, task)
			if err != nil {
				fmt.Printf("❌ 任务失败: %v\n", err)
				continue
			}
			fmt.Printf("✅ 执行结果: %s\n", result)
		}

		fmt.Println()
	}

	// 8. 展示工具信息
	fmt.Println("📊 框架信息总览")
	fmt.Println("================")
	tools := toolRegistry.List()
	fmt.Printf("🔧 已注册工具数量: %d\n", len(tools))
	fmt.Printf("⚙️  Agent 配置 - 最大步数: %d\n", agentConfig.MaxSteps)
	fmt.Printf("🤖 LLM 模型: %s\n", cfg.LLM.Model)
	fmt.Println()

	// 9. 展示已注册的工具列表
	fmt.Println("📋 可用工具列表:")
	for _, tool := range tools {
		fmt.Printf("  - %s: %s\n", tool.Name(), tool.Description())
	}
	fmt.Println()

	fmt.Println("🎉 Hello World 示例完成！")
	fmt.Println()
	fmt.Println("📚 下一步学习建议：")
	fmt.Println("  1. 查看 ../02-tool-usage/ 学习工具使用")
	fmt.Println("  2. 查看 ../03-configuration/ 学习配置管理")
	fmt.Println("  3. 设置真实的 API Key 体验完整功能")
	fmt.Println()
	fmt.Println("💡 提示：运行 'make build' 构建完整项目")
	fmt.Println("💡 提示：运行 './bin/openmanus run --help' 查看 CLI 帮助")
}
