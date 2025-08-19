package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"openmanus-go/pkg/config"
	"openmanus-go/pkg/tool"
	"openmanus-go/pkg/tool/builtin"
	"github.com/spf13/cobra"
)

// NewToolsCommand 创建工具命令
func NewToolsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tools",
		Short: "工具管理命令",
		Long: `管理和查看可用的工具。

子命令:
  list     - 列出所有可用工具
  info     - 显示特定工具的详细信息
  test     - 测试工具连接`,
	}

	cmd.AddCommand(newToolsListCommand())
	cmd.AddCommand(newToolsInfoCommand())
	cmd.AddCommand(newToolsTestCommand())

	return cmd
}

func newToolsListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出所有可用工具",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			format, _ := cmd.Flags().GetString("format")

			// 加载配置
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// 创建工具注册表
			registry := tool.NewRegistry()
			if err := builtin.RegisterBuiltinTools(registry, cfg); err != nil {
				return fmt.Errorf("failed to register tools: %w", err)
			}

			// 获取工具清单
			manifest := registry.GetToolsManifest()

			switch format {
			case "json":
				return outputJSON(manifest)
			case "table":
				return outputTable(manifest)
			default:
				return outputDefault(manifest)
			}
		},
	}

	cmd.Flags().StringP("format", "f", "default", "输出格式 (default, table, json)")

	return cmd
}

func newToolsInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info <tool-name>",
		Short: "显示特定工具的详细信息",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			toolName := args[0]
			configPath, _ := cmd.Flags().GetString("config")

			// 加载配置
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// 创建工具注册表
			registry := tool.NewRegistry()
			if err := builtin.RegisterBuiltinTools(registry, cfg); err != nil {
				return fmt.Errorf("failed to register tools: %w", err)
			}

			// 获取工具信息
			toolInstance, err := registry.Get(toolName)
			if err != nil {
				return fmt.Errorf("tool not found: %s", toolName)
			}

			// 显示详细信息
			fmt.Printf("Tool: %s\n", toolInstance.Name())
			fmt.Printf("Description: %s\n\n", toolInstance.Description())

			fmt.Println("Input Schema:")
			inputJSON, _ := json.MarshalIndent(toolInstance.InputSchema(), "", "  ")
			fmt.Printf("%s\n\n", inputJSON)

			fmt.Println("Output Schema:")
			outputJSON, _ := json.MarshalIndent(toolInstance.OutputSchema(), "", "  ")
			fmt.Printf("%s\n", outputJSON)

			return nil
		},
	}
}

func newToolsTestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "test [tool-name]",
		Short: "测试工具连接",
		Long: `测试工具的连接和基本功能。

如果不指定工具名称，将测试所有支持测试的工具。`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Flags().GetString("config")

			// 加载配置
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if len(args) > 0 {
				return testSingleTool(args[0], cfg)
			} else {
				return testAllTools(cfg)
			}
		},
	}
}

func outputDefault(manifest []tool.ToolInfo) error {
	fmt.Printf("Available Tools (%d):\n\n", len(manifest))

	for _, toolInfo := range manifest {
		fmt.Printf("📋 %s\n", toolInfo.Name)
		fmt.Printf("   %s\n\n", toolInfo.Description)
	}

	return nil
}

func outputTable(manifest []tool.ToolInfo) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tDESCRIPTION")
	fmt.Fprintln(w, "----\t-----------")

	for _, toolInfo := range manifest {
		description := toolInfo.Description
		if len(description) > 60 {
			description = description[:57] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\n", toolInfo.Name, description)
	}

	return w.Flush()
}

func outputJSON(manifest []tool.ToolInfo) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(manifest)
}

func testSingleTool(toolName string, cfg *config.Config) error {
	fmt.Printf("Testing tool: %s\n", toolName)

	// 验证工具配置
	if err := builtin.ValidateToolConfig(toolName, cfg); err != nil {
		fmt.Printf("❌ Configuration validation failed: %v\n", err)
		return err
	}

	// 创建工具实例
	toolInstance, err := builtin.CreateToolFromConfig(toolName, cfg)
	if err != nil {
		fmt.Printf("❌ Failed to create tool: %v\n", err)
		return err
	}

	// 执行特定的测试
	switch toolName {
	case "redis":
		return testRedisTool(toolInstance)
	case "mysql":
		return testMySQLTool(toolInstance)
	case "browser":
		return testBrowserTool(toolInstance)
	default:
		fmt.Printf("✅ Tool '%s' created successfully\n", toolName)
		fmt.Printf("   Type: %T\n", toolInstance)
		fmt.Printf("   Description: %s\n", toolInstance.Description())
		return nil
	}
}

func testAllTools(cfg *config.Config) error {
	fmt.Println("Testing all available tools...\n")

	toolNames := builtin.GetBuiltinToolsList()
	successCount := 0

	for _, toolName := range toolNames {
		fmt.Printf("🔧 Testing %s... ", toolName)

		if err := builtin.ValidateToolConfig(toolName, cfg); err != nil {
			fmt.Printf("❌ Config invalid: %v\n", err)
			continue
		}

		if _, err := builtin.CreateToolFromConfig(toolName, cfg); err != nil {
			fmt.Printf("❌ Failed: %v\n", err)
			continue
		}

		fmt.Printf("✅ OK\n")
		successCount++
	}

	fmt.Printf("\nSummary: %d/%d tools passed basic tests\n", successCount, len(toolNames))

	if successCount < len(toolNames) {
		fmt.Println("\n💡 Tip: Use 'openmanus tools info <tool-name>' for detailed information")
		fmt.Println("   Some tools may require additional configuration (Redis, MySQL, etc.)")
	}

	return nil
}

func testRedisTool(toolInstance tool.Tool) error {
	if redisTool, ok := toolInstance.(*builtin.RedisTool); ok {
		if err := redisTool.Ping(nil); err != nil {
			fmt.Printf("❌ Redis connection failed: %v\n", err)
			return err
		}
		fmt.Printf("✅ Redis connection successful\n")
	}
	return nil
}

func testMySQLTool(toolInstance tool.Tool) error {
	if mysqlTool, ok := toolInstance.(*builtin.MySQLTool); ok {
		if err := mysqlTool.Ping(nil); err != nil {
			fmt.Printf("❌ MySQL connection failed: %v\n", err)
			return err
		}
		fmt.Printf("✅ MySQL connection successful\n")
	}
	return nil
}

func testBrowserTool(toolInstance tool.Tool) error {
	fmt.Printf("✅ Browser tool created (headless mode)\n")
	fmt.Printf("   Note: Browser functionality requires system dependencies\n")
	return nil
}
