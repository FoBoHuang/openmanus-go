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

// 文件系统工具使用示例
// 展示文件系统工具的各种功能和使用场景
// 包括文件读写、目录操作、权限管理等

func main() {
	fmt.Println("📁 OpenManus-Go 文件系统工具示例")
	fmt.Println("=" + strings.Repeat("=", 40))
	fmt.Println()

	// 1. 初始化
	ctx := context.Background()
	cfg := loadConfig()
	toolRegistry := setupFileSystemTools()

	hasAPIKey := cfg.LLM.APIKey != "" && cfg.LLM.APIKey != "your-api-key-here"

	// 2. 展示工具信息
	fmt.Println("🔧 文件系统工具详情:")
	fsTool, err := toolRegistry.Get("fs")
	if err != nil {
		log.Fatalf("❌ 获取文件系统工具失败: %v", err)
	}

	fmt.Printf("  📝 名称: %s\n", fsTool.Name())
	fmt.Printf("  📄 描述: %s\n", fsTool.Description())

	// 展示支持的操作
	schema := fsTool.InputSchema()
	if properties, ok := schema["properties"].(map[string]any); ok {
		if operation, ok := properties["operation"].(map[string]any); ok {
			if desc, ok := operation["description"].(string); ok {
				fmt.Printf("  ⚙️  支持操作: %s\n", desc)
			}
		}
	}
	fmt.Println()

	// 3. 直接工具调用演示
	fmt.Println("🧪 直接工具调用演示")
	fmt.Println(strings.Repeat("-", 30))

	demonstrateDirectToolUsage(ctx, fsTool)

	// 4. Agent 智能调用演示
	if hasAPIKey {
		fmt.Println("\n🤖 Agent 智能调用演示")
		fmt.Println(strings.Repeat("-", 30))

		llmClient := llm.NewOpenAIClient(cfg.ToLLMConfig())
		agentConfig := agent.DefaultConfig()
		agentConfig.MaxSteps = 5

		baseAgent := agent.NewBaseAgent(llmClient, toolRegistry, agentConfig)

		demonstrateAgentUsage(ctx, baseAgent)
	} else {
		fmt.Println("\n💡 设置 API Key 后可体验 Agent 智能文件操作")
	}

	// 5. 安全特性演示
	fmt.Println("\n🔒 安全特性演示")
	fmt.Println(strings.Repeat("-", 20))

	demonstrateSecurity(ctx, fsTool)

	// 6. 性能测试
	fmt.Println("\n⚡ 性能测试")
	fmt.Println(strings.Repeat("-", 15))

	performanceTest(ctx, fsTool)

	fmt.Println("\n🎉 文件系统工具示例完成！")
	fmt.Println("\n📚 学习要点:")
	fmt.Println("  ✅ 文件系统工具支持多种操作类型")
	fmt.Println("  ✅ 内置安全限制防止误操作")
	fmt.Println("  ✅ Agent 能智能选择合适的文件操作")
	fmt.Println("  ✅ 支持高性能的文件处理")
}

func loadConfig() *config.Config {
	configPath := "../../../configs/config.toml"
	cfg, err := config.Load(configPath)
	if err != nil {
		cfg = config.DefaultConfig()
	}
	return cfg
}

func setupFileSystemTools() *tool.Registry {
	toolRegistry := tool.NewRegistry()

	// 创建更宽松的文件系统工具用于演示
	fsTool := builtin.NewFileSystemTool(
		[]string{
			"../../../workspace",
			"../../../examples",
			"/tmp", // 用于临时文件演示
		},
		[]string{
			"/etc",
			"/sys",
			"/proc",
			"/root",
		},
	)

	if err := toolRegistry.Register(fsTool); err != nil {
		log.Fatalf("❌ 注册文件系统工具失败: %v", err)
	}

	fmt.Println("✅ 文件系统工具注册成功")
	return toolRegistry
}

func demonstrateDirectToolUsage(ctx context.Context, fsTool tool.Tool) {
	workspaceDir := "../../../workspace"

	fmt.Println("📝 1. 文件写入操作")
	result, err := fsTool.Invoke(ctx, map[string]any{
		"operation": "write",
		"path":      workspaceDir + "/fs_demo.txt",
		"content":   fmt.Sprintf("文件系统工具演示\n创建时间: %s\n", time.Now().Format("2006-01-02 15:04:05")),
	})
	if err != nil {
		fmt.Printf("  ❌ 写入失败: %v\n", err)
	} else if success, _ := result["success"].(bool); success {
		fmt.Printf("  ✅ 写入成功: %s\n", result["result"])
	}

	fmt.Println("\n📖 2. 文件读取操作")
	result, err = fsTool.Invoke(ctx, map[string]any{
		"operation": "read",
		"path":      workspaceDir + "/fs_demo.txt",
	})
	if err != nil {
		fmt.Printf("  ❌ 读取失败: %v\n", err)
	} else if success, _ := result["success"].(bool); success {
		content := result["content"].(string)
		fmt.Printf("  ✅ 读取成功，内容:\n%s", content)
	}

	fmt.Println("\n📂 3. 目录创建操作")
	result, err = fsTool.Invoke(ctx, map[string]any{
		"operation": "mkdir",
		"path":      workspaceDir + "/demo_dir",
		"recursive": true,
	})
	if err != nil {
		fmt.Printf("  ❌ 创建目录失败: %v\n", err)
	} else if success, _ := result["success"].(bool); success {
		fmt.Printf("  ✅ 目录创建成功: %s\n", result["result"])
	}

	fmt.Println("\n📋 4. 目录列表操作")
	result, err = fsTool.Invoke(ctx, map[string]any{
		"operation": "list",
		"path":      workspaceDir,
	})
	if err != nil {
		fmt.Printf("  ❌ 列表失败: %v\n", err)
	} else if success, _ := result["success"].(bool); success {
		if files, ok := result["files"].([]any); ok {
			fmt.Printf("  ✅ 找到 %d 个文件/目录:\n", len(files))
			for i, file := range files {
				if i >= 5 { // 限制显示数量
					fmt.Printf("    ... 还有 %d 个文件\n", len(files)-5)
					break
				}
				if fileInfo, ok := file.(map[string]any); ok {
					fmt.Printf("    - %s (%s)\n", fileInfo["name"], fileInfo["type"])
				}
			}
		}
	}

	fmt.Println("\n📊 5. 文件状态查询")
	result, err = fsTool.Invoke(ctx, map[string]any{
		"operation": "stat",
		"path":      workspaceDir + "/fs_demo.txt",
	})
	if err != nil {
		fmt.Printf("  ❌ 状态查询失败: %v\n", err)
	} else if success, _ := result["success"].(bool); success {
		fmt.Printf("  ✅ 文件信息:\n")
		fmt.Printf("    大小: %v 字节\n", result["size"])
		fmt.Printf("    类型: %v\n", result["is_dir"])
	}

	fmt.Println("\n✅ 6. 文件存在性检查")
	result, err = fsTool.Invoke(ctx, map[string]any{
		"operation": "exists",
		"path":      workspaceDir + "/fs_demo.txt",
	})
	if err != nil {
		fmt.Printf("  ❌ 检查失败: %v\n", err)
	} else if success, _ := result["success"].(bool); success {
		exists, _ := result["exists"].(bool)
		fmt.Printf("  ✅ 文件存在: %v\n", exists)
	}
}

func demonstrateAgentUsage(ctx context.Context, agent agent.Agent) {
	tasks := []string{
		"在 workspace 目录创建一个名为 'agent_test.txt' 的文件，内容包含当前时间和一段欢迎信息",
		"读取刚创建的 agent_test.txt 文件并验证内容",
		"在 workspace 目录创建一个子目录 'agent_files'，并在其中创建3个示例文件",
		"列出 workspace 目录的所有文件，并生成一个文件清单保存到 'file_list.txt'",
	}

	for i, task := range tasks {
		fmt.Printf("\n📋 Agent 任务 %d: %s\n", i+1, task)
		fmt.Println(strings.Repeat("-", 50))

		startTime := time.Now()
		result, err := agent.Loop(ctx, task)
		duration := time.Since(startTime)

		if err != nil {
			fmt.Printf("❌ 任务失败: %v\n", err)
		} else {
			fmt.Printf("✅ 任务完成 (耗时: %v)\n", duration.Round(time.Millisecond))

			// 显示结果摘要
			if len(result) > 200 {
				fmt.Printf("📄 结果摘要: %s...\n", result[:200])
			} else {
				fmt.Printf("📄 结果: %s\n", result)
			}

			// 显示执行步骤
			if trace := agent.GetTrace(); trace != nil && len(trace.Steps) > 0 {
				fmt.Printf("🔍 执行步骤: %d 步\n", len(trace.Steps))
			}
		}
	}
}

func demonstrateSecurity(ctx context.Context, fsTool tool.Tool) {
	fmt.Println("🚫 尝试访问禁止路径")

	// 尝试访问系统目录
	result, err := fsTool.Invoke(ctx, map[string]any{
		"operation": "list",
		"path":      "/etc",
	})
	if err != nil {
		fmt.Printf("  ✅ 安全限制生效: %v\n", err)
	} else if success, _ := result["success"].(bool); !success {
		fmt.Printf("  ✅ 安全限制生效: %s\n", result["error"])
	} else {
		fmt.Printf("  ⚠️  安全限制可能失效\n")
	}

	fmt.Println("\n🔍 尝试访问允许路径")
	result, err = fsTool.Invoke(ctx, map[string]any{
		"operation": "exists",
		"path":      "../../../workspace",
	})
	if err != nil {
		fmt.Printf("  ❌ 访问失败: %v\n", err)
	} else if success, _ := result["success"].(bool); success {
		fmt.Printf("  ✅ 允许路径访问成功\n")
	}
}

func performanceTest(ctx context.Context, fsTool tool.Tool) {
	fmt.Println("📈 文件操作性能测试")

	workspaceDir := "../../../workspace"
	testCount := 10

	// 批量文件写入测试
	fmt.Printf("🔥 批量创建 %d 个文件...\n", testCount)
	startTime := time.Now()

	for i := 0; i < testCount; i++ {
		_, err := fsTool.Invoke(ctx, map[string]any{
			"operation": "write",
			"path":      fmt.Sprintf("%s/perf_test_%d.txt", workspaceDir, i),
			"content":   fmt.Sprintf("Performance test file %d\nCreated at: %s", i, time.Now().Format(time.RFC3339)),
		})
		if err != nil {
			fmt.Printf("  ❌ 文件 %d 创建失败: %v\n", i, err)
		}
	}

	duration := time.Since(startTime)
	fmt.Printf("  ✅ %d 个文件创建完成，耗时: %v (平均: %v/文件)\n",
		testCount, duration.Round(time.Millisecond),
		(duration / time.Duration(testCount)).Round(time.Millisecond))

	// 批量文件读取测试
	fmt.Printf("\n📖 批量读取 %d 个文件...\n", testCount)
	startTime = time.Now()

	successCount := 0
	for i := 0; i < testCount; i++ {
		result, err := fsTool.Invoke(ctx, map[string]any{
			"operation": "read",
			"path":      fmt.Sprintf("%s/perf_test_%d.txt", workspaceDir, i),
		})
		if err == nil {
			if success, _ := result["success"].(bool); success {
				successCount++
			}
		}
	}

	duration = time.Since(startTime)
	fmt.Printf("  ✅ %d/%d 个文件读取成功，耗时: %v (平均: %v/文件)\n",
		successCount, testCount, duration.Round(time.Millisecond),
		(duration / time.Duration(testCount)).Round(time.Millisecond))

	// 清理测试文件
	fmt.Println("\n🧹 清理测试文件...")
	for i := 0; i < testCount; i++ {
		fsTool.Invoke(ctx, map[string]any{
			"operation": "delete",
			"path":      fmt.Sprintf("%s/perf_test_%d.txt", workspaceDir, i),
		})
	}
	fmt.Println("  ✅ 测试文件清理完成")
}
