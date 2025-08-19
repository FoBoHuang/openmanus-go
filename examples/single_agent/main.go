package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"openmanus-go/pkg/agent"
	"openmanus-go/pkg/config"
	"openmanus-go/pkg/llm"
	"openmanus-go/pkg/state"
	"openmanus-go/pkg/tool"
	"openmanus-go/pkg/tool/builtin"
)

func main() {
	// 示例：单 Agent 执行简单任务
	fmt.Println("🤖 OpenManus-Go Single Agent Example")
	fmt.Println("=====================================")

	// 1. 加载配置
	cfg := config.DefaultConfig()
	cfg.LLM.APIKey = "your-api-key-here" // 在实际使用中设置真实的 API Key
	cfg.Agent.MaxSteps = 5

	// 2. 创建 LLM 客户端
	llmClient := llm.NewOpenAIClient(cfg.ToLLMConfig())

	// 3. 创建工具注册表并注册基础工具
	toolRegistry := tool.NewRegistry()

	// 只注册一些基础工具用于演示
	httpTool := builtin.NewHTTPTool()
	if err := toolRegistry.Register(httpTool); err != nil {
		log.Fatalf("Failed to register HTTP tool: %v", err)
	}

	fsTool := builtin.NewFileSystemTool([]string{"./examples"}, []string{})
	if err := toolRegistry.Register(fsTool); err != nil {
		log.Fatalf("Failed to register FS tool: %v", err)
	}

	// 4. 创建 Agent
	agentConfig := agent.DefaultConfig()
	agentConfig.MaxSteps = 5

	baseAgent := agent.NewBaseAgent(llmClient, toolRegistry, agentConfig)

	// 5. 定义任务目标
	goals := []string{
		"创建一个名为 hello.txt 的文件，内容为 'Hello, OpenManus-Go!'",
		"读取刚才创建的 hello.txt 文件内容",
		"列出当前目录下的所有文件",
	}

	// 6. 执行任务
	ctx := context.Background()
	store := state.NewFileStore("./examples/traces")

	for i, goal := range goals {
		fmt.Printf("\n📋 Task %d: %s\n", i+1, goal)
		fmt.Println(strings.Repeat("-", 50))

		result, err := baseAgent.Loop(ctx, goal)
		if err != nil {
			fmt.Printf("❌ Task failed: %v\n", err)
			continue
		}

		fmt.Printf("✅ Result: %s\n", result)

		// 保存轨迹
		trace := baseAgent.GetTrace()
		if trace != nil {
			if err := store.Save(trace); err != nil {
				fmt.Printf("⚠️  Warning: Failed to save trace: %v\n", err)
			} else {
				fmt.Printf("📝 Trace saved\n")
			}
		}
	}

	fmt.Println("\n🎉 All tasks completed!")
}
