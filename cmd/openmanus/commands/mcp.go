package commands

import (
	"fmt"

	"openmanus-go/pkg/config"
	"openmanus-go/pkg/logger"
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

	// 初始化 logger 输出位置（遵守配置）
	logger.InitWithConfig(logger.Config{Level: cfg.Logging.Level, Output: cfg.Logging.Output, FilePath: cfg.Logging.FilePath})

	// 创建工具注册表
	toolRegistry := tool.NewRegistry()

	// 注册内置工具
	if err := builtin.RegisterBuiltinTools(toolRegistry, cfg); err != nil {
		return fmt.Errorf("failed to register builtin tools: %w", err)
	}

	// 创建 MCP 服务器
	mcpServer := mcp.NewServer(toolRegistry)

	// 使用全局 zap logger 输出信息
	logger.Get().Sugar().Infof("🚀 Starting MCP Server on %s:%d", host, port)
	logger.Get().Sugar().Infow("📋 Available endpoints",
		"endpoints", []string{"POST /", "GET /tools", "POST /tools/invoke", "GET /health"},
	)
	logger.Get().Sugar().Infof("🔧 Registered %d tools", len(toolRegistry.ListNames()))

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

	logger.Get().Sugar().Infof("# MCP Tools Documentation")
	logger.Get().Sugar().Infof("This document describes the available tools in the OpenManus-Go MCP server.")
	logger.Get().Sugar().Infof("**Total Tools**: %d", len(toolRegistry.ListNames()))
	logger.Get().Sugar().Infof("## Available Tools")

	// 生成实际的工具文档
	for _, toolName := range toolRegistry.ListNames() {
		t, err := toolRegistry.Get(toolName)
		if err != nil {
			continue
		}

		logger.Get().Sugar().Infof("### %s", t.Name())
		logger.Get().Sugar().Infof("- **Description**: %s", t.Description())

		// 输入 Schema
		if inputSchema := t.InputSchema(); inputSchema != nil {
			logger.Get().Sugar().Infof("- **Input Schema**:")
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
						logger.Get().Sugar().Infof("  - `%s` (%s): %s", propName, propType, propDesc)
					}
				}
			}
		}

		// 输出 Schema
		if outputSchema := t.OutputSchema(); outputSchema != nil {
			logger.Get().Sugar().Infof("- **Output Schema**:")
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
						logger.Get().Sugar().Infof("  - `%s` (%s): %s", propName, propType, propDesc)
					}
				}
			}
		}

		logger.Get().Sugar().Info("")
	}

	logger.Get().Sugar().Infof("## Usage Examples")
	logger.Get().Sugar().Infof("### MCP Protocol (JSON-RPC)")
	logger.Get().Sugar().Infof("```json")
	logger.Get().Sugar().Infof(`{
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
	logger.Get().Sugar().Infof("```")

	logger.Get().Sugar().Infof("### REST API")
	logger.Get().Sugar().Infof("```bash")
	logger.Get().Sugar().Infof("# List tools")
	logger.Get().Sugar().Infof("curl http://localhost:8080/tools")
	logger.Get().Sugar().Infof("# Invoke tool")
	logger.Get().Sugar().Infof(`curl -X POST http://localhost:8080/tools/invoke \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "http",
    "args": {
      "url": "https://api.example.com/data",
      "method": "GET"
    }
  }'`)
	logger.Get().Sugar().Infof("`")

	return nil
}
