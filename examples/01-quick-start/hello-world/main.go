package main

import (
	"context"
	"fmt"
	"log"
	"strings"

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
	fmt.Println("🚀 OpenManus-Go Hello World 示例")
	fmt.Println("=" + strings.Repeat("=", 30))
	fmt.Println()

	// 1. 检查配置文件
	configPath := "../../../configs/config.toml"
	fmt.Printf("📁 加载配置文件: %s\n", configPath)

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("⚠️  配置文件加载失败，使用默认配置: %v\n", err)
		cfg = config.DefaultConfig()
	}

	// 检查 API Key
	if cfg.LLM.APIKey == "" || cfg.LLM.APIKey == "your-api-key-here" {
		fmt.Println()
		fmt.Println("⚠️  未设置 LLM API Key")
		fmt.Println("请在 configs/config.toml 中设置正确的 api_key")
		fmt.Println()
		fmt.Println("示例配置：")
		fmt.Println("[llm]")
		fmt.Println(`model = "deepseek-chat"`)
		fmt.Println(`base_url = "https://api.deepseek.com/v1"`)
		fmt.Println(`api_key = "your-actual-api-key"`)
		fmt.Println()
		fmt.Println("📝 继续演示框架结构（模拟模式）...")
		fmt.Println()
	}

	// 2. 创建 LLM 客户端
	fmt.Println("🤖 创建 LLM 客户端...")
	llmClient := llm.NewOpenAIClient(cfg.ToLLMConfig())
	fmt.Printf("✅ LLM 客户端已创建 (模型: %s)\n", cfg.LLM.Model)

	// 3. 创建工具注册表并注册基础工具
	fmt.Println("\n🔧 注册基础工具...")
	toolRegistry := tool.NewRegistry()

	// 注册文件系统工具
	fsTool := builtin.NewFileSystemTool(
		[]string{"../../../workspace"}, // 允许访问 workspace 目录
		[]string{},                     // 无禁止路径
	)
	if err := toolRegistry.Register(fsTool); err != nil {
		log.Fatalf("❌ 注册文件系统工具失败: %v", err)
	}
	fmt.Println("  ✅ 文件系统工具 (fs)")

	// 注册 HTTP 工具
	httpTool := builtin.NewHTTPTool()
	if err := toolRegistry.Register(httpTool); err != nil {
		log.Fatalf("❌ 注册 HTTP 工具失败: %v", err)
	}
	fmt.Println("  ✅ HTTP 工具 (http)")

	tools := toolRegistry.List()
	fmt.Printf("📊 共注册 %d 个工具\n", len(tools))

	// 4. 创建 Agent
	fmt.Println("\n🧠 创建 Agent...")
	agentConfig, err := agent.ConfigFromAppConfig(cfg)
	if err != nil {
		fmt.Printf("❌ 创建 Agent 配置失败: %v\n", err)
		return
	}
	agentConfig.MaxSteps = 5 // 限制步数，适合简单演示

	baseAgent := agent.NewBaseAgent(llmClient, toolRegistry, agentConfig)
	fmt.Println("✅ Agent 已创建")

	// 5. 展示 Agent 配置信息
	fmt.Println("\n📋 Agent 配置信息:")
	fmt.Printf("  - 最大步数: %d\n", agentConfig.MaxSteps)
	fmt.Printf("  - 最大执行时间: %v\n", agentConfig.MaxDuration)
	fmt.Printf("  - 反思间隔: %d 步\n", agentConfig.ReflectionSteps)

	// 6. 定义简单任务
	tasks := []string{
		"在 workspace 目录创建一个名为 hello.txt 的文件，内容为 'Hello, OpenManus-Go!'",
		"检查 workspace 目录下的文件列表",
		"获取 https://httpbin.org/json 的响应内容",
	}

	// 7. 执行任务演示
	ctx := context.Background()
	hasAPIKey := cfg.LLM.APIKey != "" && cfg.LLM.APIKey != "your-api-key-here"

	for i, task := range tasks {
		fmt.Printf("\n📋 任务 %d: %s\n", i+1, task)
		fmt.Println(strings.Repeat("-", 50))

		if !hasAPIKey {
			// 模拟模式 - 展示框架结构
			fmt.Println("🔄 模拟执行中...")
			fmt.Println("💭 Agent 分析任务...")

			switch i {
			case 0:
				fmt.Println("🔧 选择工具: fs (文件系统)")
				fmt.Println("📝 执行操作: 写入文件")
				fmt.Println("✅ 模拟结果: 文件创建成功")
			case 1:
				fmt.Println("🔧 选择工具: fs (文件系统)")
				fmt.Println("📝 执行操作: 列出目录")
				fmt.Println("✅ 模拟结果: 找到 2 个文件")
			case 2:
				fmt.Println("🔧 选择工具: http (HTTP 客户端)")
				fmt.Println("📝 执行操作: GET 请求")
				fmt.Println("✅ 模拟结果: 获取 JSON 数据成功")
			}
		} else {
			// 实际执行模式
			fmt.Println("🔄 正在执行...")
			result, err := baseAgent.Loop(ctx, task)
			if err != nil {
				fmt.Printf("❌ 任务失败: %v\n", err)
				continue
			}
			fmt.Printf("✅ 执行结果:\n%s\n", result)

			// 显示执行轨迹
			if trace := baseAgent.GetTrace(); trace != nil && len(trace.Steps) > 0 {
				fmt.Printf("🔍 执行了 %d 个步骤\n", len(trace.Steps))
			}
		}
	}

	// 8. 展示工具能力
	fmt.Println("\n🛠️  可用工具详情:")
	fmt.Println("=" + strings.Repeat("=", 20))
	for i, tool := range tools {
		fmt.Printf("%d. %s\n", i+1, tool.Name())
		fmt.Printf("   📝 描述: %s\n", tool.Description())

		// 展示工具参数
		schema := tool.InputSchema()
		if properties, ok := schema["properties"].(map[string]any); ok {
			fmt.Printf("   ⚙️  参数: ")
			var params []string
			for param := range properties {
				params = append(params, param)
			}
			fmt.Println(strings.Join(params, ", "))
		}
		fmt.Println()
	}

	// 9. 总结和下一步建议
	fmt.Println("🎉 Hello World 示例完成！")
	fmt.Println()

	if !hasAPIKey {
		fmt.Println("💡 下一步:")
		fmt.Println("  1. 在 configs/config.toml 中设置真实的 API Key")
		fmt.Println("  2. 重新运行此示例体验完整功能")
		fmt.Println()
	}

	fmt.Println("📚 继续学习:")
	fmt.Println("  1. 查看 ../basic-tasks/ 学习更多任务类型")
	fmt.Println("  2. 查看 ../configuration/ 学习配置管理")
	fmt.Println("  3. 查看 ../../02-tool-usage/ 学习工具使用")
	fmt.Println()

	fmt.Println("💡 提示:")
	fmt.Println("  - 运行 'make build' 构建完整项目")
	fmt.Println("  - 运行 '../../../bin/openmanus run \"你的任务\"' 使用 CLI")
	fmt.Println("  - 查看 workspace 目录查看文件操作结果")
}
