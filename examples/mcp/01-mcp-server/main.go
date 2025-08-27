package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"openmanus-go/pkg/mcp"
	"openmanus-go/pkg/tool"
	"openmanus-go/pkg/tool/builtin"
)

// MCP 服务器示例
// 展示如何创建和启动一个简单的 HTTP 服务器，模拟 MCP 服务器的基本功能
// 提供标准化的工具接口供其他 MCP 客户端调用

func main() {
	fmt.Println("🔌 OpenManus-Go MCP Server Example")
	fmt.Println("==================================")
	fmt.Println()

	// 1. 创建工具注册表
	toolRegistry := tool.NewRegistry()
	fmt.Println("✅ 工具注册表已创建")

	// 2. 注册内置工具
	fmt.Println("\n🔧 注册工具到 MCP 服务器...")

	// 2.1 文件系统工具
	fsTool := builtin.NewFileSystemTool(
		[]string{"./workspace", "./examples"},
		[]string{"/etc", "/sys"},
	)
	if err := toolRegistry.Register(fsTool); err != nil {
		log.Fatalf("❌ 注册文件系统工具失败: %v", err)
	}
	fmt.Println("  ✅ 文件系统工具 (fs)")

	// 2.2 HTTP 工具
	httpTool := builtin.NewHTTPTool()
	if err := toolRegistry.Register(httpTool); err != nil {
		log.Fatalf("❌ 注册 HTTP 工具失败: %v", err)
	}
	fmt.Println("  ✅ HTTP 工具 (http)")

	// 2.3 爬虫工具
	crawlerTool := builtin.NewCrawlerTool("OpenManus-Go-MCP-Server/1.0", []string{}, []string{})
	if err := toolRegistry.Register(crawlerTool); err != nil {
		log.Fatalf("❌ 注册爬虫工具失败: %v", err)
	}
	fmt.Println("  ✅ 爬虫工具 (crawler)")

	// 2.4 可选工具（如果服务可用）
	redisTool := builtin.NewRedisTool("localhost:6379", "", 0)
	if err := toolRegistry.Register(redisTool); err != nil {
		fmt.Printf("  ⚠️  Redis 工具注册失败 (可能缺少 Redis 服务): %v\n", err)
	} else {
		fmt.Println("  ✅ Redis 工具 (redis)")
	}

	tools := toolRegistry.List()
	fmt.Printf("\n📊 MCP 服务器将暴露 %d 个工具\n", len(tools))

	// 3. 设置服务器信息
	serverInfo := mcp.ServerInfo{
		Name:    "openmanus-example-server",
		Version: "1.0.0",
	}
	fmt.Printf("✅ 服务器信息已设置: %s v%s\n", serverInfo.Name, serverInfo.Version)

	// 4. 设置服务器端口
	port := "8080"
	if envPort := os.Getenv("MCP_SERVER_PORT"); envPort != "" {
		port = envPort
	}

	// 5. 启动 HTTP 服务器
	mux := http.NewServeMux()

	// 5.1 健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := fmt.Sprintf(`{
			"status": "healthy",
			"server": "%s",
			"version": "%s",
			"tools_count": %d,
			"timestamp": "%s"
		}`, serverInfo.Name, serverInfo.Version, len(tools), time.Now().Format(time.RFC3339))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	})

	// 5.2 工具列表端点
	mux.HandleFunc("/tools", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// 构建 JSON 响应
		response := "["
		for i, tool := range tools {
			if i > 0 {
				response += ","
			}
			response += fmt.Sprintf(`{
				"name": "%s",
				"description": "%s"
			}`, tool.Name(), tool.Description())
		}
		response += "]"

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	})

	// 5.3 工具调用端点
	mux.HandleFunc("/tools/invoke", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "只支持 POST 方法", http.StatusMethodNotAllowed)
			return
		}

		// 这里简化处理，实际实现会解析 JSON 请求体
		w.Header().Set("Content-Type", "application/json")
		response := `{
			"success": true,
			"message": "工具调用端点已就绪，请使用 MCP 客户端进行调用"
		}`
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	})

	// 5.4 MCP 协议端点（JSON-RPC）
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := `{
			"jsonrpc": "2.0",
			"result": {
				"message": "MCP 协议端点已就绪"
			}
		}`
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	})

	// 5.5 根路径信息
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>OpenManus-Go MCP Server</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .header { color: #2c3e50; }
        .info { background: #f8f9fa; padding: 20px; border-radius: 5px; }
        .endpoint { margin: 10px 0; }
        .code { background: #e9ecef; padding: 2px 4px; border-radius: 3px; font-family: monospace; }
    </style>
</head>
<body>
    <h1 class="header">🔌 OpenManus-Go MCP Server</h1>
    <div class="info">
        <p><strong>服务器:</strong> %s v%s</p>
        <p><strong>描述:</strong> OpenManus-Go 示例 MCP 服务器</p>
        <p><strong>工具数量:</strong> %d</p>
        <p><strong>状态:</strong> 运行中 ✅</p>
    </div>
    
    <h2>可用端点</h2>
    <div class="endpoint">🏥 <strong>健康检查:</strong> <span class="code">GET /health</span></div>
    <div class="endpoint">🔧 <strong>工具列表:</strong> <span class="code">GET /tools</span></div>
    <div class="endpoint">⚡ <strong>工具调用:</strong> <span class="code">POST /tools/invoke</span></div>
    <div class="endpoint">🔌 <strong>MCP 协议:</strong> <span class="code">POST /mcp</span></div>
    
    <h2>示例使用</h2>
    <pre><code># 健康检查
curl http://localhost:%s/health

# 获取工具列表
curl http://localhost:%s/tools

# MCP 客户端连接
# 使用 OpenManus-Go MCP 客户端连接到此服务器</code></pre>
</body>
</html>`, serverInfo.Name, serverInfo.Version, len(tools), port, port)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// 8. 启动服务器
	fmt.Printf("\n🚀 启动 MCP 服务器...\n")
	fmt.Printf("📡 监听地址: http://localhost:%s\n", port)
	fmt.Printf("🏥 健康检查: http://localhost:%s/health\n", port)
	fmt.Printf("🔧 工具列表: http://localhost:%s/tools\n", port)
	fmt.Printf("🔌 MCP 协议: http://localhost:%s/mcp\n", port)
	fmt.Printf("🌐 Web 界面: http://localhost:%s/\n", port)
	fmt.Println()

	// 9. 优雅关闭处理
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		fmt.Println("\n🛑 收到关闭信号，正在优雅关闭服务器...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			fmt.Printf("❌ 服务器关闭失败: %v\n", err)
		} else {
			fmt.Println("✅ 服务器已优雅关闭")
		}
	}()

	// 10. 启动服务器
	fmt.Println("🎯 MCP 服务器运行中... (按 Ctrl+C 停止)")
	fmt.Println("📋 可用工具:")
	for _, tool := range tools {
		fmt.Printf("  - %s: %s\n", tool.Name(), tool.Description())
	}
	fmt.Println()

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("❌ HTTP 服务器启动失败: %v", err)
	}

	fmt.Println("👋 MCP 服务器已停止")
}
