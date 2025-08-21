package main

import (
	"fmt"
	"os"

	"openmanus-go/cmd/openmanus/commands"
	"openmanus-go/pkg/logger"

	"github.com/spf13/cobra"
)

var (
	version = "1.0.0"
	commit  = "dev"
	date    = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "openmanus",
		Short: "OpenManus-Go: A generalist AI Agent framework",
		Long: `OpenManus-Go is a generalist AI Agent framework that helps users accomplish their goals
through a Plan -> Tool Use -> Observation -> Reflection -> Next Action loop.

It supports:
- Multiple LLM providers (OpenAI compatible)
- Extensible tool system
- MCP (Model Context Protocol) integration
- Multi-agent workflows
- Data analysis capabilities`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	}

	// 添加全局标志
	rootCmd.PersistentFlags().StringP("config", "c", "", "配置文件路径")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "详细输出")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "调试模式")

	// 添加子命令
	rootCmd.AddCommand(commands.NewRunCommand())
	rootCmd.AddCommand(commands.NewMCPCommand())
	rootCmd.AddCommand(commands.NewFlowCommand())
	rootCmd.AddCommand(commands.NewConfigCommand())
	rootCmd.AddCommand(commands.NewToolsCommand())

	// 执行命令
	if err := rootCmd.Execute(); err != nil {
		logger.Errorw("command execution error", "error", err)
		os.Exit(1)
	}
}
