package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"openmanus-go/pkg/config"
	"github.com/spf13/cobra"
)

// NewConfigCommand 创建配置命令
func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "配置管理命令",
		Long: `管理 OpenManus-Go 的配置文件。

子命令:
  show     - 显示当前配置
  init     - 初始化配置文件
  validate - 验证配置文件`,
	}

	cmd.AddCommand(newConfigShowCommand())
	cmd.AddCommand(newConfigInitCommand())
	cmd.AddCommand(newConfigValidateCommand())

	return cmd
}

func newConfigShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "显示当前配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Flags().GetString("config")

			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			fmt.Printf("Configuration loaded from: %s\n\n", getConfigPath(configPath))

			fmt.Printf("LLM Configuration:\n")
			fmt.Printf("  Model: %s\n", cfg.LLM.Model)
			fmt.Printf("  Base URL: %s\n", cfg.LLM.BaseURL)
			fmt.Printf("  API Key: %s\n", maskAPIKey(cfg.LLM.APIKey))
			fmt.Printf("  Temperature: %.2f\n", cfg.LLM.Temperature)
			fmt.Printf("  Max Tokens: %d\n", cfg.LLM.MaxTokens)
			fmt.Printf("  Timeout: %ds\n\n", cfg.LLM.Timeout)

			fmt.Printf("Agent Configuration:\n")
			fmt.Printf("  Max Steps: %d\n", cfg.Agent.MaxSteps)
			fmt.Printf("  Max Tokens: %d\n", cfg.Agent.MaxTokens)
			fmt.Printf("  Max Duration: %s\n", cfg.Agent.MaxDuration)
			fmt.Printf("  Reflection Steps: %d\n", cfg.Agent.ReflectionSteps)
			fmt.Printf("  Max Retries: %d\n", cfg.Agent.MaxRetries)
			fmt.Printf("  Retry Backoff: %s\n\n", cfg.Agent.RetryBackoff)

			fmt.Printf("RunFlow Configuration:\n")
			fmt.Printf("  Use Data Analysis Agent: %t\n", cfg.RunFlow.UseDataAnalysisAgent)
			fmt.Printf("  Enable Multi Agent: %t\n\n", cfg.RunFlow.EnableMultiAgent)

			fmt.Printf("Storage Configuration:\n")
			fmt.Printf("  Type: %s\n", cfg.Storage.Type)
			fmt.Printf("  Base Path: %s\n\n", cfg.Storage.BasePath)

			fmt.Printf("Tools Configuration:\n")
			fmt.Printf("  HTTP Timeout: %ds\n", cfg.Tools.HTTP.Timeout)
			fmt.Printf("  Allowed Domains: %v\n", cfg.Tools.HTTP.AllowedDomains)
			fmt.Printf("  Blocked Domains: %v\n", cfg.Tools.HTTP.BlockedDomains)
			fmt.Printf("  FileSystem Allowed Paths: %v\n", cfg.Tools.FileSystem.AllowedPaths)
			fmt.Printf("  FileSystem Blocked Paths: %v\n", cfg.Tools.FileSystem.BlockedPaths)
			fmt.Printf("  Browser Headless: %t\n", cfg.Tools.Browser.Headless)

			return nil
		},
	}
}

func newConfigInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "初始化配置文件",
		Long: `创建一个新的配置文件模板。

如果未指定路径，将在当前目录创建 config.toml 文件。`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var configPath string
			if len(args) > 0 {
				configPath = args[0]
			} else {
				configPath = "config.toml"
			}

			// 检查文件是否已存在
			if _, err := os.Stat(configPath); err == nil {
				overwrite, _ := cmd.Flags().GetBool("force")
				if !overwrite {
					return fmt.Errorf("config file already exists: %s (use --force to overwrite)", configPath)
				}
			}

			// 确保目录存在
			dir := filepath.Dir(configPath)
			if dir != "." {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return fmt.Errorf("failed to create directory: %w", err)
				}
			}

			// 创建配置文件
			template := config.GetConfigTemplate()
			if err := os.WriteFile(configPath, []byte(template), 0644); err != nil {
				return fmt.Errorf("failed to write config file: %w", err)
			}

			fmt.Printf("✅ Configuration file created: %s\n", configPath)
			fmt.Println("\n📝 Please edit the configuration file and set your API key:")
			fmt.Printf("   api_key = \"your-api-key-here\"\n")
			fmt.Println("\n🔧 You can also customize other settings as needed.")

			return nil
		},
	}

	cmd.Flags().BoolP("force", "f", false, "覆盖已存在的配置文件")

	return cmd
}

func newConfigValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "验证配置文件",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Flags().GetString("config")

			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if err := cfg.Validate(); err != nil {
				fmt.Printf("❌ Configuration validation failed:\n")
				fmt.Printf("   %v\n", err)
				return err
			}

			fmt.Printf("✅ Configuration is valid\n")
			fmt.Printf("   Config path: %s\n", getConfigPath(configPath))
			fmt.Printf("   LLM Model: %s\n", cfg.LLM.Model)
			fmt.Printf("   Max Steps: %d\n", cfg.Agent.MaxSteps)
			fmt.Printf("   Storage Type: %s\n", cfg.Storage.Type)

			return nil
		},
	}
}

func getConfigPath(configPath string) string {
	if configPath != "" {
		return configPath
	}

	// 检查常见的配置文件位置
	candidates := []string{
		"./config.toml",
		"./configs/config.toml",
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return "default configuration"
}

func maskAPIKey(apiKey string) string {
	if apiKey == "" {
		return "(not set)"
	}

	if len(apiKey) <= 8 {
		return "***"
	}

	return apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
}
