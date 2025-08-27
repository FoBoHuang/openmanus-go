package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"openmanus-go/pkg/agent"
	"openmanus-go/pkg/config"
	"openmanus-go/pkg/llm"
	"openmanus-go/pkg/state"
	"openmanus-go/pkg/tool"
	"openmanus-go/pkg/tool/builtin"
)

// 工具使用示例
// 展示 OpenManus-Go 框架中各种内置工具的使用方法
// 包括文件系统、HTTP、数据库等工具的注册和调用

func main() {
	fmt.Println("🔧 OpenManus-Go Tool Usage Example")
	fmt.Println("==================================")
	fmt.Println()

	// 1. 加载配置
	cfg := config.DefaultConfig()

	// 检查 API Key 配置
	hasAPIKey := cfg.LLM.APIKey != "" && cfg.LLM.APIKey != "your-api-key-here"
	if !hasAPIKey {
		fmt.Println("⚠️  未设置 LLM API Key，将演示工具注册和基本调用")
		fmt.Println()
	}

	// 2. 创建基础组件
	llmClient := llm.NewOpenAIClient(cfg.ToLLMConfig())
	toolRegistry := tool.NewRegistry()

	fmt.Println("✅ 基础组件已创建")

	// 3. 注册多种内置工具
	fmt.Println("\n🔧 注册内置工具...")

	// 3.1 文件系统工具
	fsTool := builtin.NewFileSystemTool(
		[]string{"./workspace", "./examples"}, // 允许访问的路径
		[]string{"/etc", "/sys"},              // 禁止访问的路径
	)
	if err := toolRegistry.Register(fsTool); err != nil {
		log.Fatalf("❌ 注册文件系统工具失败: %v", err)
	}
	fmt.Println("  ✅ 文件系统工具 (fs)")

	// 3.2 HTTP 工具
	httpTool := builtin.NewHTTPTool()
	if err := toolRegistry.Register(httpTool); err != nil {
		log.Fatalf("❌ 注册 HTTP 工具失败: %v", err)
	}
	fmt.Println("  ✅ HTTP 工具 (http)")

	// 3.3 浏览器工具（可选，需要 Chrome）
	browserTool, err := builtin.NewBrowserTool(true, 30*time.Second) // headless=true, timeout=30s
	if err != nil {
		fmt.Printf("  ⚠️  浏览器工具创建失败 (可能缺少 Chrome): %v\n", err)
	} else if err := toolRegistry.Register(browserTool); err != nil {
		fmt.Printf("  ⚠️  浏览器工具注册失败: %v\n", err)
	} else {
		fmt.Println("  ✅ 浏览器工具 (browser)")
	}

	// 3.4 爬虫工具
	crawlerTool := builtin.NewCrawlerTool("OpenManus-Go-Example/1.0", []string{}, []string{})
	if err := toolRegistry.Register(crawlerTool); err != nil {
		log.Fatalf("❌ 注册爬虫工具失败: %v", err)
	}
	fmt.Println("  ✅ 爬虫工具 (crawler)")

	// 3.5 Redis 工具（可选，需要 Redis 服务）
	redisTool := builtin.NewRedisTool("localhost:6379", "", 0)
	if err := toolRegistry.Register(redisTool); err != nil {
		fmt.Printf("  ⚠️  Redis 工具注册失败 (可能缺少 Redis 服务): %v\n", err)
	} else {
		fmt.Println("  ✅ Redis 工具 (redis)")
	}

	// 3.6 MySQL 工具（可选，需要 MySQL 服务）
	mysqlTool, err := builtin.NewMySQLTool("user:password@tcp(localhost:3306)/database")
	if err != nil {
		fmt.Printf("  ⚠️  MySQL 工具创建失败 (可能缺少 MySQL 服务): %v\n", err)
	} else if err := toolRegistry.Register(mysqlTool); err != nil {
		fmt.Printf("  ⚠️  MySQL 工具注册失败: %v\n", err)
	} else {
		fmt.Println("  ✅ MySQL 工具 (mysql)")
	}

	tools := toolRegistry.List()
	fmt.Printf("\n📊 总计注册了 %d 个工具\n", len(tools))

	// 4. 展示工具信息
	fmt.Println("\n📋 工具详细信息:")
	fmt.Println("================")

	for i, tool := range tools {
		fmt.Printf("%d. %s\n", i+1, tool.Name())
		fmt.Printf("   描述: %s\n", tool.Description())

		// 展示工具 Schema（简化版）
		schema := tool.InputSchema()
		if properties, ok := schema["properties"].(map[string]any); ok {
			fmt.Printf("   参数: ")
			var params []string
			for param := range properties {
				params = append(params, param)
			}
			if len(params) > 0 {
				for j, param := range params {
					if j > 0 {
						fmt.Printf(", ")
					}
					fmt.Printf("%s", param)
				}
			}
			fmt.Println()
		}
		fmt.Println()
	}

	// 5. 演示直接工具调用
	fmt.Println("🧪 直接工具调用演示")
	fmt.Println("==================")

	ctx := context.Background()

	// 5.1 文件系统工具演示
	fmt.Println("\n📁 文件系统工具演示:")
	if fsTool, err := toolRegistry.Get("fs"); err == nil {
		// 创建测试文件
		result, err := fsTool.Invoke(ctx, map[string]any{
			"operation": "write",
			"path":      "workspace/tool_test.txt",
			"content":   fmt.Sprintf("Tool test at %s", time.Now().Format("2006-01-02 15:04:05")),
		})
		if err != nil {
			fmt.Printf("  ❌ 写文件失败: %v\n", err)
		} else {
			fmt.Printf("  ✅ 写文件成功: %v\n", result["success"])
		}

		// 读取文件
		result, err = fsTool.Invoke(ctx, map[string]any{
			"operation": "read",
			"path":      "workspace/tool_test.txt",
		})
		if err != nil {
			fmt.Printf("  ❌ 读文件失败: %v\n", err)
		} else if success, ok := result["success"].(bool); ok && success {
			fmt.Printf("  ✅ 读文件成功，内容: %s\n", result["content"])
		}

		// 列出目录
		result, err = fsTool.Invoke(ctx, map[string]any{
			"operation": "list",
			"path":      "workspace",
		})
		if err != nil {
			fmt.Printf("  ❌ 列出目录失败: %v\n", err)
		} else if success, ok := result["success"].(bool); ok && success {
			if files, ok := result["files"].([]any); ok {
				fmt.Printf("  ✅ 目录列表 (%d 个文件):\n", len(files))
				for _, file := range files {
					if fileInfo, ok := file.(map[string]any); ok {
						fmt.Printf("    - %s (%s)\n", fileInfo["name"], fileInfo["type"])
					}
				}
			}
		}
	}

	// 5.2 HTTP 工具演示
	fmt.Println("\n🌐 HTTP 工具演示:")
	if httpTool, err := toolRegistry.Get("http"); err == nil {
		result, err := httpTool.Invoke(ctx, map[string]any{
			"url":    "https://httpbin.org/json",
			"method": "GET",
		})
		if err != nil {
			fmt.Printf("  ❌ HTTP 请求失败: %v\n", err)
		} else if success, ok := result["success"].(bool); ok && success {
			fmt.Printf("  ✅ HTTP 请求成功\n")
			if output, ok := result["output"].(map[string]any); ok {
				fmt.Printf("    状态码: %v\n", output["status_code"])
				fmt.Printf("    内容类型: %v\n", output["content_type"])
				if body, ok := output["body"].(string); ok && len(body) > 0 {
					if len(body) > 100 {
						fmt.Printf("    响应体: %s...\n", body[:100])
					} else {
						fmt.Printf("    响应体: %s\n", body)
					}
				}
			}
		}
	}

	// 6. 使用 Agent 执行工具相关任务
	if hasAPIKey {
		fmt.Println("\n🤖 Agent 工具使用演示")
		fmt.Println("=====================")

		// 创建 Agent
		agentConfig := agent.DefaultConfig()
		agentConfig.MaxSteps = 5
		baseAgent := agent.NewBaseAgent(llmClient, toolRegistry, agentConfig)

		// 创建状态存储
		store := state.NewFileStore("./workspace/traces")

		// 定义需要使用工具的任务
		tasks := []string{
			"检查 workspace 目录下有哪些文件",
			"获取 https://httpbin.org/ip 的响应内容",
			"创建一个名为 agent_test.txt 的文件，写入当前时间",
		}

		for i, task := range tasks {
			fmt.Printf("\n📋 Agent 任务 %d: %s\n", i+1, task)
			fmt.Println("------------------------------------")

			result, err := baseAgent.Loop(ctx, task)
			if err != nil {
				fmt.Printf("❌ 任务失败: %v\n", err)
				continue
			}

			fmt.Printf("✅ 任务完成: %s\n", result)

			// 保存执行轨迹
			if trace := baseAgent.GetTrace(); trace != nil {
				if err := store.Save(trace); err != nil {
					fmt.Printf("⚠️  保存轨迹失败: %v\n", err)
				} else {
					fmt.Printf("📝 执行轨迹已保存\n")
				}
			}
		}
	} else {
		fmt.Println("\n💡 提示：设置 API Key 后可以看到 Agent 智能选择和使用工具的完整过程")
	}

	// 7. 工具使用统计
	fmt.Println("\n📊 工具使用总结")
	fmt.Println("===============")
	fmt.Printf("🔧 可用工具数量: %d\n", len(tools))
	toolNames := make([]string, len(tools))
	for i, tool := range tools {
		toolNames[i] = tool.Name()
	}
	fmt.Printf("✅ 成功注册的工具: %v\n", toolNames)
	fmt.Println()

	// 8. 工具使用最佳实践提示
	fmt.Println("💡 工具使用最佳实践:")
	fmt.Println("1. 根据需求选择合适的工具")
	fmt.Println("2. 注意工具的依赖服务（如 Redis、MySQL）")
	fmt.Println("3. 合理设置工具的访问权限和路径限制")
	fmt.Println("4. 使用 Agent 让 LLM 智能选择工具")
	fmt.Println("5. 定期保存和分析执行轨迹")
	fmt.Println()

	fmt.Println("🎉 工具使用示例完成！")
	fmt.Println()
	fmt.Println("📚 下一步学习建议：")
	fmt.Println("  1. 查看 ../03-configuration/ 学习配置管理")
	fmt.Println("  2. 查看 ../../mcp/ 学习 MCP 工具集成")
	fmt.Println("  3. 查看 ../../applications/ 学习实际应用场景")
}
