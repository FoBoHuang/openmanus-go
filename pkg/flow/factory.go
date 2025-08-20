package flow

import (
	"fmt"

	"openmanus-go/pkg/agent"
	"openmanus-go/pkg/llm"
	"openmanus-go/pkg/tool"
)

// DefaultAgentFactory 默认 Agent 工厂
type DefaultAgentFactory struct {
	llmClient    llm.Client
	toolRegistry *tool.Registry
}

// NewDefaultAgentFactory 创建默认 Agent 工厂
func NewDefaultAgentFactory(llmClient llm.Client, toolRegistry *tool.Registry) *DefaultAgentFactory {
	return &DefaultAgentFactory{
		llmClient:    llmClient,
		toolRegistry: toolRegistry,
	}
}

// CreateAgent 创建 Agent
func (f *DefaultAgentFactory) CreateAgent(agentType string, config map[string]interface{}) (agent.Agent, error) {
	switch agentType {
	case "general", "base", "":
		return f.createGeneralAgent(config)
	case "data_analysis":
		return f.createDataAnalysisAgent(config)
	case "web_scraper":
		return f.createWebScraperAgent(config)
	case "file_processor":
		return f.createFileProcessorAgent(config)
	default:
		return nil, fmt.Errorf("unsupported agent type: %s", agentType)
	}
}

// GetSupportedTypes 获取支持的 Agent 类型
func (f *DefaultAgentFactory) GetSupportedTypes() []string {
	return []string{
		"general",
		"base",
		"data_analysis",
		"web_scraper",
		"file_processor",
	}
}

// createGeneralAgent 创建通用 Agent
func (f *DefaultAgentFactory) createGeneralAgent(config map[string]interface{}) (agent.Agent, error) {
	agentConfig := agent.DefaultConfig()

	// 从配置中覆盖默认值
	if maxSteps, ok := config["max_steps"].(float64); ok {
		agentConfig.MaxSteps = int(maxSteps)
	}
	if temperature, ok := config["temperature"].(float64); ok {
		agentConfig.Temperature = temperature
	}
	if maxTokens, ok := config["max_tokens"].(float64); ok {
		agentConfig.MaxTokens = int(maxTokens)
	}

	return agent.NewBaseAgent(f.llmClient, f.toolRegistry, agentConfig), nil
}

// createDataAnalysisAgent 创建数据分析 Agent
func (f *DefaultAgentFactory) createDataAnalysisAgent(config map[string]interface{}) (agent.Agent, error) {
	agentConfig := agent.DefaultConfig()

	// 数据分析 Agent 的特殊配置
	agentConfig.MaxSteps = 15     // 数据分析可能需要更多步骤
	agentConfig.Temperature = 0.1 // 更低的温度确保准确性

	// 从配置中覆盖默认值
	if maxSteps, ok := config["max_steps"].(float64); ok {
		agentConfig.MaxSteps = int(maxSteps)
	}
	if temperature, ok := config["temperature"].(float64); ok {
		agentConfig.Temperature = temperature
	}

	// 创建专门的工具注册表，只包含数据分析相关工具
	dataToolRegistry := tool.NewRegistry()

	// 注册数据分析相关工具
	for _, toolName := range []string{"http", "fs", "mysql", "redis"} {
		if t, err := f.toolRegistry.Get(toolName); err == nil {
			dataToolRegistry.Register(t)
		}
	}

	return agent.NewBaseAgent(f.llmClient, dataToolRegistry, agentConfig), nil
}

// createWebScraperAgent 创建网页爬虫 Agent
func (f *DefaultAgentFactory) createWebScraperAgent(config map[string]interface{}) (agent.Agent, error) {
	agentConfig := agent.DefaultConfig()

	// 网页爬虫 Agent 的特殊配置
	agentConfig.MaxSteps = 10
	agentConfig.Temperature = 0.2

	// 从配置中覆盖默认值
	if maxSteps, ok := config["max_steps"].(float64); ok {
		agentConfig.MaxSteps = int(maxSteps)
	}
	if temperature, ok := config["temperature"].(float64); ok {
		agentConfig.Temperature = temperature
	}

	// 创建专门的工具注册表，只包含网页相关工具
	webToolRegistry := tool.NewRegistry()

	// 注册网页相关工具
	for _, toolName := range []string{"http", "browser", "crawler", "fs"} {
		if t, err := f.toolRegistry.Get(toolName); err == nil {
			webToolRegistry.Register(t)
		}
	}

	return agent.NewBaseAgent(f.llmClient, webToolRegistry, agentConfig), nil
}

// createFileProcessorAgent 创建文件处理 Agent
func (f *DefaultAgentFactory) createFileProcessorAgent(config map[string]interface{}) (agent.Agent, error) {
	agentConfig := agent.DefaultConfig()

	// 文件处理 Agent 的特殊配置
	agentConfig.MaxSteps = 8
	agentConfig.Temperature = 0.1

	// 从配置中覆盖默认值
	if maxSteps, ok := config["max_steps"].(float64); ok {
		agentConfig.MaxSteps = int(maxSteps)
	}
	if temperature, ok := config["temperature"].(float64); ok {
		agentConfig.Temperature = temperature
	}

	// 创建专门的工具注册表，只包含文件相关工具
	fileToolRegistry := tool.NewRegistry()

	// 注册文件相关工具
	for _, toolName := range []string{"fs", "http"} {
		if t, err := f.toolRegistry.Get(toolName); err == nil {
			fileToolRegistry.Register(t)
		}
	}

	return agent.NewBaseAgent(f.llmClient, fileToolRegistry, agentConfig), nil
}

// ValidateAgentConfig 验证 Agent 配置
func (f *DefaultAgentFactory) ValidateAgentConfig(agentType string, config map[string]interface{}) error {
	supportedTypes := f.GetSupportedTypes()
	found := false
	for _, t := range supportedTypes {
		if t == agentType {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("unsupported agent type: %s, supported types: %v", agentType, supportedTypes)
	}

	// 验证配置参数
	if maxSteps, ok := config["max_steps"]; ok {
		if steps, ok := maxSteps.(float64); ok {
			if steps <= 0 || steps > 100 {
				return fmt.Errorf("max_steps must be between 1 and 100, got: %v", steps)
			}
		} else {
			return fmt.Errorf("max_steps must be a number, got: %T", maxSteps)
		}
	}

	if temperature, ok := config["temperature"]; ok {
		if temp, ok := temperature.(float64); ok {
			if temp < 0 || temp > 2 {
				return fmt.Errorf("temperature must be between 0 and 2, got: %v", temp)
			}
		} else {
			return fmt.Errorf("temperature must be a number, got: %T", temperature)
		}
	}

	if maxTokens, ok := config["max_tokens"]; ok {
		if tokens, ok := maxTokens.(float64); ok {
			if tokens <= 0 || tokens > 100000 {
				return fmt.Errorf("max_tokens must be between 1 and 100000, got: %v", tokens)
			}
		} else {
			return fmt.Errorf("max_tokens must be a number, got: %T", maxTokens)
		}
	}

	return nil
}

// GetAgentDescription 获取 Agent 类型描述
func (f *DefaultAgentFactory) GetAgentDescription(agentType string) string {
	descriptions := map[string]string{
		"general":        "通用 Agent，可以处理各种任务",
		"base":           "基础 Agent，与 general 相同",
		"data_analysis":  "数据分析 Agent，专门用于数据处理和分析任务",
		"web_scraper":    "网页爬虫 Agent，专门用于网页抓取和数据提取",
		"file_processor": "文件处理 Agent，专门用于文件操作和处理",
	}

	if desc, ok := descriptions[agentType]; ok {
		return desc
	}
	return "未知 Agent 类型"
}

// GetDefaultConfig 获取 Agent 类型的默认配置
func (f *DefaultAgentFactory) GetDefaultConfig(agentType string) map[string]interface{} {
	configs := map[string]map[string]interface{}{
		"general": {
			"max_steps":   10,
			"temperature": 0.1,
			"max_tokens":  8000,
		},
		"base": {
			"max_steps":   10,
			"temperature": 0.1,
			"max_tokens":  8000,
		},
		"data_analysis": {
			"max_steps":   15,
			"temperature": 0.1,
			"max_tokens":  10000,
		},
		"web_scraper": {
			"max_steps":   10,
			"temperature": 0.2,
			"max_tokens":  8000,
		},
		"file_processor": {
			"max_steps":   8,
			"temperature": 0.1,
			"max_tokens":  6000,
		},
	}

	if config, ok := configs[agentType]; ok {
		return config
	}
	return configs["general"]
}
