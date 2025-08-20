package commands

import (
	"fmt"
	"log"
	"os"

	"openmanus-go/pkg/config"
	"openmanus-go/pkg/mcp"
	"openmanus-go/pkg/tool"
	"openmanus-go/pkg/tool/builtin"

	"github.com/spf13/cobra"
)

// NewMCPCommand 创建 MCP 命令
func NewMCPCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "MCP (Model Context Protocol) 服务器",
		Long: `启动 MCP 服务器，提供标准化的工具接口。

MCP 允许 AI 模型通过标准协议访问工具和服务，实现跨平台和跨语言的工具集成。

示例:
  openmanus mcp --port 8080
  openmanus mcp --host 0.0.0.0 --port 9000`,
		RunE: runMCPServer,
	}

	cmd.Flags().StringP("host", "H", "localhost", "服务器监听地址")
	cmd.Flags().IntP("port", "p", 8080, "服务器监听端口")
	cmd.Flags().BoolP("docs", "", false, "生成 MCP 工具文档")

	return cmd
}

func runMCPServer(cmd *cobra.Command, args []string) error {
	host, _ := cmd.Flags().GetString("host")
	port, _ := cmd.Flags().GetInt("port")
	docs, _ := cmd.Flags().GetBool("docs")

	if docs {
		return generateMCPDocs()
	}

	// 加载配置
	cfg := config.DefaultConfig()

	// 创建工具注册表
	toolRegistry := tool.NewRegistry()

	// 注册内置工具
	if err := builtin.RegisterBuiltinTools(toolRegistry, cfg); err != nil {
		return fmt.Errorf("failed to register builtin tools: %w", err)
	}

	// 创建 MCP 服务器
	mcpServer := mcp.NewServer(toolRegistry)

	// 设置日志器
	logger := log.New(os.Stdout, "[MCP] ", log.LstdFlags)
	mcpServer.SetLogger(logger)

	fmt.Printf("🚀 Starting MCP Server on %s:%d\n", host, port)
	fmt.Println("📋 Available endpoints:")
	fmt.Println("   POST /             - MCP protocol endpoint")
	fmt.Println("   GET  /tools        - List available tools (REST)")
	fmt.Println("   POST /tools/invoke - Invoke a tool (REST)")
	fmt.Println("   GET  /health       - Health check")
	fmt.Printf("🔧 Registered %d tools\n", len(toolRegistry.ListNames()))
	fmt.Println()

	// 启动服务器
	return mcpServer.Start(host, port)
}

func generateMCPDocs() error {
	// 加载配置
	cfg := config.DefaultConfig()

	// 创建工具注册表
	toolRegistry := tool.NewRegistry()

	// 注册内置工具
	if err := builtin.RegisterBuiltinTools(toolRegistry, cfg); err != nil {
		return fmt.Errorf("failed to register builtin tools: %w", err)
	}

	fmt.Println("# MCP Tools Documentation")
	fmt.Println()
	fmt.Println("This document describes the available tools in the OpenManus-Go MCP server.")
	fmt.Println()
	fmt.Printf("**Total Tools**: %d\n", len(toolRegistry.ListNames()))
	fmt.Println()

	fmt.Println("## Available Tools")
	fmt.Println()

	// 生成实际的工具文档
	for _, toolName := range toolRegistry.ListNames() {
		t, err := toolRegistry.Get(toolName)
		if err != nil {
			continue
		}

		fmt.Printf("### %s\n", t.Name())
		fmt.Printf("- **Description**: %s\n", t.Description())

		// 输入 Schema
		if inputSchema := t.InputSchema(); inputSchema != nil {
			fmt.Println("- **Input Schema**:")
			if properties, ok := inputSchema["properties"].(map[string]interface{}); ok {
				for propName, propDef := range properties {
					if prop, ok := propDef.(map[string]interface{}); ok {
						propType := "unknown"
						if t, ok := prop["type"].(string); ok {
							propType = t
						}
						propDesc := ""
						if d, ok := prop["description"].(string); ok {
							propDesc = d
						}
						fmt.Printf("  - `%s` (%s): %s\n", propName, propType, propDesc)
					}
				}
			}
		}

		// 输出 Schema
		if outputSchema := t.OutputSchema(); outputSchema != nil {
			fmt.Println("- **Output Schema**:")
			if properties, ok := outputSchema["properties"].(map[string]interface{}); ok {
				for propName, propDef := range properties {
					if prop, ok := propDef.(map[string]interface{}); ok {
						propType := "unknown"
						if t, ok := prop["type"].(string); ok {
							propType = t
						}
						propDesc := ""
						if d, ok := prop["description"].(string); ok {
							propDesc = d
						}
						fmt.Printf("  - `%s` (%s): %s\n", propName, propType, propDesc)
					}
				}
			}
		}

		fmt.Println()
	}

	fmt.Println("## Usage Examples")
	fmt.Println()
	fmt.Println("### MCP Protocol (JSON-RPC)")
	fmt.Println("```json")
	fmt.Println(`{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "tools/call",
  "params": {
    "name": "http",
    "arguments": {
      "url": "https://api.example.com/data",
      "method": "GET"
    }
  }
}`)
	fmt.Println("```")
	fmt.Println()

	fmt.Println("### REST API")
	fmt.Println("```bash")
	fmt.Println("# List tools")
	fmt.Println("curl http://localhost:8080/tools")
	fmt.Println()
	fmt.Println("# Invoke tool")
	fmt.Println(`curl -X POST http://localhost:8080/tools/invoke \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "http",
    "args": {
      "url": "https://api.example.com/data",
      "method": "GET"
    }
  }'`)
	fmt.Println("```")

	return nil
}
