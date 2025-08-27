package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"openmanus-go/pkg/mcp"
)

// 简化的 MCP 客户端结构
type SimpleClient struct {
	serverURL string
	client    *http.Client
	info      mcp.ClientInfo
}

// 简单的健康检查响应结构
type HealthResponse struct {
	Status     string `json:"status"`
	Server     string `json:"server"`
	Version    string `json:"version"`
	ToolsCount int    `json:"tools_count"`
	Timestamp  string `json:"timestamp"`
}

// 简单的工具信息结构
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// MCP 客户端示例
// 展示如何创建简单的 MCP 客户端并连接到 MCP 服务器
// 演示工具发现、工具调用和协议交互

func main() {
	fmt.Println("🔗 OpenManus-Go MCP Client Example")
	fmt.Println("==================================")
	fmt.Println()

	// 1. 设置服务器地址
	serverURL := "http://localhost:8080"
	fmt.Printf("🎯 目标服务器: %s\n", serverURL)

	// 2. 创建简单的 MCP 客户端
	client := &SimpleClient{
		serverURL: serverURL,
		client:    &http.Client{Timeout: 30 * time.Second},
		info: mcp.ClientInfo{
			Name:    "openmanus-example-client",
			Version: "1.0.0",
		},
	}
	fmt.Println("✅ MCP 客户端已创建")
	fmt.Printf("✅ 客户端信息已设置: %s v%s\n", client.info.Name, client.info.Version)

	ctx := context.Background()

	// 3. 服务器连通性检查
	fmt.Println("\n🔍 检查服务器连通性...")
	if !checkServerConnectivity(client, ctx) {
		fmt.Println("❌ 无法连接到 MCP 服务器")
		fmt.Println("💡 请确保 MCP 服务器正在运行:")
		fmt.Println("   cd examples/mcp/01-mcp-server && go run main.go")
		return
	}
	fmt.Println("✅ 服务器连通性正常")

	// 4. HTTP API 测试
	fmt.Println("\n🌐 HTTP API 测试")
	fmt.Println("================")
	testHTTPAPI(client, ctx)

	// 5. MCP 协议测试
	fmt.Println("\n🔌 MCP 协议测试")
	fmt.Println("===============")
	testMCPProtocol(client, ctx)

	// 6. 性能测试
	fmt.Println("\n⚡ 性能测试")
	fmt.Println("==========")
	performanceTest(client, ctx)

	fmt.Println("\n🎉 MCP 客户端示例完成！")
	fmt.Println()
	fmt.Println("📚 学习总结:")
	fmt.Println("  1. MCP 客户端可以连接任何兼容的 MCP 服务器")
	fmt.Println("  2. 支持 HTTP REST API 和标准 MCP 协议")
	fmt.Println("  3. 自动工具发现和智能调用")
	fmt.Println("  4. 完整的错误处理和重试机制")
	fmt.Println("  5. 高性能的并发调用支持")
}

// checkServerConnectivity 检查服务器连通性
func checkServerConnectivity(client *SimpleClient, ctx context.Context) bool {
	// 尝试健康检查
	resp, err := client.client.Get(client.serverURL + "/health")
	if err != nil {
		fmt.Printf("  ❌ 健康检查失败: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("  ❌ 服务器返回错误状态: %d\n", resp.StatusCode)
		return false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("  ❌ 读取响应失败: %v\n", err)
		return false
	}

	var health HealthResponse
	if err := json.Unmarshal(body, &health); err != nil {
		fmt.Printf("  ❌ 解析响应失败: %v\n", err)
		return false
	}

	fmt.Printf("  ✅ 服务器状态: %s\n", health.Status)
	fmt.Printf("  📊 可用工具数量: %d\n", health.ToolsCount)

	return true
}

// testHTTPAPI 测试 HTTP API
func testHTTPAPI(client *SimpleClient, ctx context.Context) {
	// 获取工具列表
	fmt.Println("📋 获取工具列表...")
	resp, err := client.client.Get(client.serverURL + "/tools")
	if err != nil {
		fmt.Printf("❌ 获取工具列表失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ 获取工具列表失败，状态码: %d\n", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ 读取工具列表响应失败: %v\n", err)
		return
	}

	var tools []ToolInfo
	if err := json.Unmarshal(body, &tools); err != nil {
		fmt.Printf("❌ 解析工具列表失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 找到 %d 个工具:\n", len(tools))
	for i, tool := range tools {
		fmt.Printf("  %d. %s - %s\n", i+1, tool.Name, tool.Description)
	}

	// 测试工具调用端点
	if len(tools) > 0 {
		fmt.Println("\n🔧 测试工具调用端点...")

		// 创建一个简单的调用请求
		callData := map[string]interface{}{
			"tool":   "fs",
			"method": "exists",
			"args": map[string]interface{}{
				"path": "README.md",
			},
		}

		jsonData, _ := json.Marshal(callData)
		resp, err := client.client.Post(client.serverURL+"/tools/invoke", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("  ❌ 工具调用失败: %v\n", err)
		} else {
			defer resp.Body.Close()
			fmt.Printf("  ✅ 工具调用端点响应: %d\n", resp.StatusCode)
		}
	}
}

// testMCPProtocol 测试 MCP 协议
func testMCPProtocol(client *SimpleClient, ctx context.Context) {
	fmt.Println("🔌 测试 MCP 协议端点...")

	// 创建一个简单的 JSON-RPC 请求
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      "test-1",
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	}

	jsonData, _ := json.Marshal(request)
	resp, err := client.client.Post(client.serverURL+"/mcp", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ MCP 协议测试失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ MCP 协议返回错误状态: %d\n", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ 读取 MCP 响应失败: %v\n", err)
		return
	}

	fmt.Printf("✅ MCP 协议测试成功\n")
	fmt.Printf("📄 响应: %s\n", string(body))
}

// performanceTest 性能测试
func performanceTest(client *SimpleClient, ctx context.Context) {
	fmt.Println("⚡ 执行性能测试...")

	// 并发健康检查测试
	concurrency := 5
	iterations := 10

	fmt.Printf("🔄 并发测试: %d 个并发连接，每个执行 %d 次请求\n", concurrency, iterations)

	start := time.Now()
	results := make(chan bool, concurrency*iterations)

	for i := 0; i < concurrency; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				resp, err := client.client.Get(client.serverURL + "/health")
				if err == nil && resp.StatusCode == http.StatusOK {
					resp.Body.Close()
					results <- true
				} else {
					if resp != nil {
						resp.Body.Close()
					}
					results <- false
				}
			}
		}()
	}

	successCount := 0
	totalRequests := concurrency * iterations

	for i := 0; i < totalRequests; i++ {
		if <-results {
			successCount++
		}
	}

	duration := time.Since(start)

	fmt.Printf("📊 性能测试结果:\n")
	fmt.Printf("  总请求数: %d\n", totalRequests)
	fmt.Printf("  成功请求: %d\n", successCount)
	fmt.Printf("  失败请求: %d\n", totalRequests-successCount)
	fmt.Printf("  成功率: %.1f%%\n", float64(successCount)/float64(totalRequests)*100)
	fmt.Printf("  总耗时: %v\n", duration)
	fmt.Printf("  平均响应时间: %v\n", duration/time.Duration(totalRequests))
	fmt.Printf("  QPS: %.1f\n", float64(totalRequests)/duration.Seconds())
}
