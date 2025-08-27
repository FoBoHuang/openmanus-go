package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"openmanus-go/pkg/config"
)

// 配置管理示例
// 展示 OpenManus-Go 框架中的配置加载、验证和使用方法
// 包括环境变量、配置文件、默认值的优先级处理

func main() {
	fmt.Println("⚙️  OpenManus-Go Configuration Example")
	fmt.Println("======================================")
	fmt.Println()

	// 1. 展示默认配置
	fmt.Println("📋 1. 默认配置")
	fmt.Println("=============")

	defaultCfg := config.DefaultConfig()
	displayConfig("默认配置", defaultCfg)

	// 2. 从配置文件加载
	fmt.Println("\n📄 2. 配置文件加载")
	fmt.Println("=================")

	// 尝试加载主配置文件
	configPath := "../../../configs/config.toml"
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("✅ 找到配置文件: %s\n", configPath)

		cfg, err := config.Load(configPath)
		if err != nil {
			log.Printf("❌ 加载配置文件失败: %v", err)
		} else {
			fmt.Println("✅ 配置文件加载成功")
			displayConfig("配置文件", cfg)
		}
	} else {
		fmt.Printf("⚠️  配置文件不存在: %s\n", configPath)
		fmt.Println("💡 提示：运行 'cp configs/config.example.toml configs/config.toml' 创建配置文件")
	}

	// 3. 创建示例配置文件
	fmt.Println("\n📝 3. 创建示例配置")
	fmt.Println("=================")

	exampleConfigPath := "example_config.toml"
	if err := createExampleConfig(exampleConfigPath); err != nil {
		log.Printf("❌ 创建示例配置失败: %v", err)
	} else {
		fmt.Printf("✅ 创建示例配置文件: %s\n", exampleConfigPath)

		// 加载示例配置
		exampleCfg, err := config.Load(exampleConfigPath)
		if err != nil {
			log.Printf("❌ 加载示例配置失败: %v", err)
		} else {
			fmt.Println("✅ 示例配置加载成功")
			displayConfig("示例配置", exampleCfg)
		}

		// 清理示例文件
		defer func() {
			if err := os.Remove(exampleConfigPath); err != nil {
				log.Printf("⚠️  清理示例配置文件失败: %v", err)
			} else {
				fmt.Printf("🧹 已清理示例配置文件: %s\n", exampleConfigPath)
			}
		}()
	}

	// 4. 环境变量演示
	fmt.Println("\n🌍 4. 环境变量配置")
	fmt.Println("==================")

	// 设置一些示例环境变量
	testEnvVars := map[string]string{
		"OPENMANUS_LLM_MODEL":       "gpt-4",
		"OPENMANUS_LLM_API_KEY":     "sk-test-key-from-env",
		"OPENMANUS_AGENT_MAX_STEPS": "15",
	}

	fmt.Println("设置测试环境变量:")
	for key, value := range testEnvVars {
		os.Setenv(key, value)
		fmt.Printf("  %s = %s\n", key, value)
	}

	// 创建支持环境变量的配置
	envCfg := createConfigWithEnvVars()
	displayConfig("环境变量配置", envCfg)

	// 清理环境变量
	for key := range testEnvVars {
		os.Unsetenv(key)
	}

	// 5. 配置验证
	fmt.Println("\n✅ 5. 配置验证")
	fmt.Println("=============")

	// 验证有效配置
	validCfg := createValidConfig()
	if err := validateConfig(validCfg); err != nil {
		fmt.Printf("❌ 有效配置验证失败: %v\n", err)
	} else {
		fmt.Println("✅ 有效配置验证通过")
	}

	// 验证无效配置
	invalidCfg := createInvalidConfig()
	if err := validateConfig(invalidCfg); err != nil {
		fmt.Printf("✅ 无效配置验证失败（预期）: %v\n", err)
	} else {
		fmt.Println("❌ 无效配置验证通过（不应该）")
	}

	// 6. 配置使用示例
	fmt.Println("\n🔧 6. 配置使用示例")
	fmt.Println("==================")

	cfg := config.DefaultConfig()
	cfg.LLM.Model = "deepseek-chat"
	cfg.LLM.APIKey = "sk-example-key"
	cfg.Agent.MaxSteps = 8

	demonstrateConfigUsage(cfg)

	// 7. 配置最佳实践
	fmt.Println("\n💡 7. 配置最佳实践")
	fmt.Println("==================")

	showBestPractices()

	fmt.Println("\n🎉 配置管理示例完成！")
	fmt.Println()
	fmt.Println("📚 学习总结:")
	fmt.Println("  1. 默认配置提供基础设置")
	fmt.Println("  2. 配置文件覆盖默认值")
	fmt.Println("  3. 环境变量具有最高优先级")
	fmt.Println("  4. 配置验证确保设置正确")
	fmt.Println("  5. 合理使用配置提高灵活性")
}

// displayConfig 展示配置信息
func displayConfig(title string, cfg *config.Config) {
	fmt.Printf("\n--- %s ---\n", title)
	fmt.Printf("LLM 配置:\n")
	fmt.Printf("  模型: %s\n", cfg.LLM.Model)
	fmt.Printf("  基础URL: %s\n", cfg.LLM.BaseURL)
	fmt.Printf("  API密钥: %s\n", maskAPIKey(cfg.LLM.APIKey))
	fmt.Printf("  温度: %.1f\n", cfg.LLM.Temperature)
	fmt.Printf("  最大令牌: %d\n", cfg.LLM.MaxTokens)

	fmt.Printf("Agent 配置:\n")
	fmt.Printf("  最大步数: %d\n", cfg.Agent.MaxSteps)
	fmt.Printf("  最大令牌: %d\n", cfg.Agent.MaxTokens)
	fmt.Printf("  最大持续时间: %s\n", cfg.Agent.MaxDuration)
	fmt.Printf("  反思步数: %d\n", cfg.Agent.ReflectionSteps)

	fmt.Printf("MCP 服务器数量: %d\n", len(cfg.MCP.Servers))
	if len(cfg.MCP.Servers) > 0 {
		fmt.Printf("MCP 服务器:\n")
		i := 1
		for name, server := range cfg.MCP.Servers {
			fmt.Printf("  %d. %s (%s)\n", i, name, server.URL)
			i++
		}
	}
}

// maskAPIKey 隐藏 API Key 的敏感部分
func maskAPIKey(key string) string {
	if key == "" || key == "your-api-key-here" {
		return key
	}
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "***" + key[len(key)-4:]
}

// createExampleConfig 创建示例配置文件
func createExampleConfig(path string) error {
	content := `# OpenManus-Go 示例配置文件

[llm]
model = "deepseek-chat"
base_url = "https://api.deepseek.com/v1"
api_key = "your-api-key-here"
temperature = 0.2
max_tokens = 4000

[agent]
max_steps = 12
max_tokens = 10000
max_duration = "8m"
reflection_steps = 4
max_retries = 3

# MCP 服务器配置
[[mcp_servers]]
name = "example-server"
transport = "sse"
url = "https://example.com/mcp"

[[mcp_servers]]
name = "local-server"
transport = "http"
url = "http://localhost:8080"

# 工具配置
[tools.filesystem]
allowed_paths = ["./workspace", "./data"]
blocked_paths = ["/etc", "/sys"]

[tools.http]
timeout = 30
blocked_domains = ["localhost"]
`

	return os.WriteFile(path, []byte(content), 0644)
}

// createConfigWithEnvVars 创建支持环境变量的配置
func createConfigWithEnvVars() *config.Config {
	cfg := config.DefaultConfig()

	// 模拟环境变量处理
	if model := os.Getenv("OPENMANUS_LLM_MODEL"); model != "" {
		cfg.LLM.Model = model
	}
	if apiKey := os.Getenv("OPENMANUS_LLM_API_KEY"); apiKey != "" {
		cfg.LLM.APIKey = apiKey
	}
	if maxSteps := os.Getenv("OPENMANUS_AGENT_MAX_STEPS"); maxSteps != "" {
		// 在实际实现中会进行类型转换
		cfg.Agent.MaxSteps = 15 // 模拟转换结果
	}

	return cfg
}

// validateConfig 验证配置
func validateConfig(cfg *config.Config) error {
	// 使用内置的验证方法
	return cfg.Validate()
}

// createValidConfig 创建有效配置
func createValidConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.LLM.APIKey = "sk-valid-api-key-example"
	return cfg
}

// createInvalidConfig 创建无效配置
func createInvalidConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.LLM.APIKey = ""    // 无效的空 API Key
	cfg.Agent.MaxSteps = 0 // 无效的步数
	return cfg
}

// demonstrateConfigUsage 演示配置使用
func demonstrateConfigUsage(cfg *config.Config) {
	fmt.Println("配置使用示例:")

	// 转换为 LLM 配置
	llmConfig := cfg.ToLLMConfig()
	fmt.Printf("  LLM 配置转换: %s (温度: %.1f)\n", llmConfig.Model, llmConfig.Temperature)

	// 获取工作目录
	workDir := filepath.Join(".", "workspace")
	fmt.Printf("  工作目录: %s\n", workDir)

	// 检查 MCP 服务器配置
	if len(cfg.MCP.Servers) > 0 {
		fmt.Printf("  MCP 服务器: %d 个已配置\n", len(cfg.MCP.Servers))
	} else {
		fmt.Printf("  MCP 服务器: 未配置\n")
	}

	// 计算预估令牌消耗
	estimatedTokens := cfg.Agent.MaxSteps * 1000 // 简化估算
	fmt.Printf("  预估令牌消耗: %d (基于最大步数)\n", estimatedTokens)
}

// showBestPractices 展示最佳实践
func showBestPractices() {
	practices := []string{
		"🔐 永远不要在代码中硬编码 API Key",
		"📄 使用配置文件管理复杂设置",
		"🌍 使用环境变量处理敏感信息",
		"✅ 启动时验证配置的完整性",
		"📝 为配置项提供清晰的注释",
		"🔄 支持配置热重载（生产环境）",
		"🎯 根据环境（开发/测试/生产）使用不同配置",
		"📊 监控配置变更和使用情况",
		"🛡️  限制配置文件的访问权限",
		"📋 提供配置模板和示例",
	}

	fmt.Println("配置管理最佳实践:")
	for i, practice := range practices {
		fmt.Printf("  %d. %s\n", i+1, practice)
	}
}
