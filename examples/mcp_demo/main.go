package main

import (
	"context"
	"fmt"
	"log"

	"openmanus-go/pkg/mcp"
)

func main() {
	fmt.Println("🔗 OpenManus-Go MCP Client Demo")
	fmt.Println("===============================")

	// 创建 MCP 客户端
	client := mcp.NewClient("http://localhost:8080")

	// 设置客户端信息
	client.SetClientInfo(mcp.ClientInfo{
		Name:    "openmanus-demo-client",
		Version: "1.0.0",
	})

	ctx := context.Background()

	// 测试健康检查
	fmt.Println("🏥 Testing health check...")
	health, err := client.HealthCheck(ctx)
	if err != nil {
		log.Printf("Health check failed: %v", err)
		fmt.Println("⚠️  Make sure MCP server is running: ./bin/openmanus mcp")
		return
	}
	fmt.Printf("✅ Server is healthy: %v\n", health["status"])
	fmt.Printf("📊 Tools available: %v\n", health["tools_count"])
	fmt.Println()

	// 获取工具列表
	fmt.Println("🔧 Fetching available tools...")
	tools, err := client.ListToolsHTTP(ctx)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	fmt.Printf("📋 Found %d tools:\n", len(tools))
	for i, tool := range tools {
		fmt.Printf("  %d. %s - %s\n", i+1, tool["name"], tool["description"])
	}
	fmt.Println()

	// 测试 HTTP 工具
	fmt.Println("🌐 Testing HTTP tool...")
	httpResult, err := client.CallToolHTTP(ctx, "http", map[string]interface{}{
		"url":    "https://httpbin.org/json",
		"method": "GET",
	})
	if err != nil {
		log.Printf("HTTP tool call failed: %v", err)
	} else {
		fmt.Printf("✅ HTTP tool result: %v\n", httpResult["success"])
		if output, ok := httpResult["output"].(map[string]interface{}); ok {
			fmt.Printf("📊 Status Code: %v\n", output["status_code"])
			fmt.Printf("📄 Content Type: %v\n", output["content_type"])
		}
	}
	fmt.Println()

	// 测试文件系统工具
	fmt.Println("📁 Testing File System tool...")
	fsResult, err := client.CallToolHTTP(ctx, "fs", map[string]interface{}{
		"operation": "list",
		"path":      ".",
	})
	if err != nil {
		log.Printf("FS tool call failed: %v", err)
	} else {
		fmt.Printf("✅ FS tool result: %v\n", fsResult["success"])
		if output, ok := fsResult["output"].(map[string]interface{}); ok {
			if files, ok := output["files"].([]interface{}); ok {
				fmt.Printf("📂 Found %d files/directories\n", len(files))
			}
		}
	}
	fmt.Println()

	// 测试 MCP 协议（JSON-RPC）
	fmt.Println("🔌 Testing MCP Protocol (JSON-RPC)...")

	// 初始化连接
	err = client.Initialize(ctx)
	if err != nil {
		log.Printf("MCP initialization failed: %v", err)
	} else {
		fmt.Println("✅ MCP client initialized")

		// 获取服务器信息
		if serverInfo := client.GetServerInfo(); serverInfo != nil {
			fmt.Printf("🖥️  Server: %s v%s\n", serverInfo.Name, serverInfo.Version)
		}

		// 通过 MCP 协议获取工具列表
		mcpTools, err := client.ListTools(ctx)
		if err != nil {
			log.Printf("MCP list tools failed: %v", err)
		} else {
			fmt.Printf("📋 MCP Protocol found %d tools\n", len(mcpTools))
		}

		// 通过 MCP 协议调用工具
		toolResult, err := client.CallTool(ctx, "fs", map[string]interface{}{
			"operation": "exists",
			"path":      "README.md",
		})
		if err != nil {
			log.Printf("MCP tool call failed: %v", err)
		} else {
			fmt.Printf("✅ MCP tool call successful: %d content items\n", len(toolResult.Content))
			if len(toolResult.Content) > 0 {
				fmt.Printf("📄 Result: %s\n", toolResult.Content[0].Text)
			}
		}
	}

	fmt.Println("\n🎉 MCP Demo completed!")
}
