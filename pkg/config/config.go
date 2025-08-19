package config

import (
	"fmt"
	"time"

	"openmanus-go/pkg/llm"
	"github.com/spf13/viper"
)

// Config 应用程序配置
type Config struct {
	LLM     LLMConfig     `mapstructure:"llm"`
	Agent   AgentConfig   `mapstructure:"agent"`
	RunFlow RunFlowConfig `mapstructure:"runflow"`
	Server  ServerConfig  `mapstructure:"server"`
	Storage StorageConfig `mapstructure:"storage"`
	Tools   ToolsConfig   `mapstructure:"tools"`
}

// LLMConfig LLM 配置
type LLMConfig struct {
	Model       string  `mapstructure:"model"`
	BaseURL     string  `mapstructure:"base_url"`
	APIKey      string  `mapstructure:"api_key"`
	Temperature float64 `mapstructure:"temperature"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	Timeout     int     `mapstructure:"timeout"`
}

// AgentConfig Agent 配置
type AgentConfig struct {
	MaxSteps        int    `mapstructure:"max_steps"`
	MaxTokens       int    `mapstructure:"max_tokens"`
	MaxDuration     string `mapstructure:"max_duration"`
	ReflectionSteps int    `mapstructure:"reflection_steps"`
	MaxRetries      int    `mapstructure:"max_retries"`
	RetryBackoff    string `mapstructure:"retry_backoff"`
}

// RunFlowConfig 流程配置
type RunFlowConfig struct {
	UseDataAnalysisAgent bool `mapstructure:"use_data_analysis_agent"`
	EnableMultiAgent     bool `mapstructure:"enable_multi_agent"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type     string `mapstructure:"type"`      // file, memory, redis, s3
	BasePath string `mapstructure:"base_path"` // for file storage
	Redis    struct {
		Addr     string `mapstructure:"addr"`
		Password string `mapstructure:"password"`
		DB       int    `mapstructure:"db"`
	} `mapstructure:"redis"`
	S3 struct {
		Region    string `mapstructure:"region"`
		Bucket    string `mapstructure:"bucket"`
		AccessKey string `mapstructure:"access_key"`
		SecretKey string `mapstructure:"secret_key"`
	} `mapstructure:"s3"`
}

// ToolsConfig 工具配置
type ToolsConfig struct {
	HTTP struct {
		Timeout        int      `mapstructure:"timeout"`
		AllowedDomains []string `mapstructure:"allowed_domains"`
		BlockedDomains []string `mapstructure:"blocked_domains"`
	} `mapstructure:"http"`

	FileSystem struct {
		AllowedPaths []string `mapstructure:"allowed_paths"`
		BlockedPaths []string `mapstructure:"blocked_paths"`
	} `mapstructure:"filesystem"`

	Browser struct {
		Headless  bool   `mapstructure:"headless"`
		Timeout   int    `mapstructure:"timeout"`
		UserAgent string `mapstructure:"user_agent"`
	} `mapstructure:"browser"`

	Database struct {
		MySQL struct {
			DSN string `mapstructure:"dsn"`
		} `mapstructure:"mysql"`

		Redis struct {
			Addr     string `mapstructure:"addr"`
			Password string `mapstructure:"password"`
			DB       int    `mapstructure:"db"`
		} `mapstructure:"redis"`
	} `mapstructure:"database"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		LLM: LLMConfig{
			Model:       "gpt-3.5-turbo",
			BaseURL:     "https://api.openai.com/v1",
			Temperature: 0.1,
			MaxTokens:   4000,
			Timeout:     30,
		},
		Agent: AgentConfig{
			MaxSteps:        10,
			MaxTokens:       8000,
			MaxDuration:     "5m",
			ReflectionSteps: 3,
			MaxRetries:      2,
			RetryBackoff:    "1s",
		},
		RunFlow: RunFlowConfig{
			UseDataAnalysisAgent: false,
			EnableMultiAgent:     false,
		},
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Storage: StorageConfig{
			Type:     "file",
			BasePath: "./data/traces",
		},
		Tools: ToolsConfig{
			HTTP: struct {
				Timeout        int      `mapstructure:"timeout"`
				AllowedDomains []string `mapstructure:"allowed_domains"`
				BlockedDomains []string `mapstructure:"blocked_domains"`
			}{
				Timeout:        30,
				AllowedDomains: []string{},
				BlockedDomains: []string{"localhost", "127.0.0.1"},
			},
			FileSystem: struct {
				AllowedPaths []string `mapstructure:"allowed_paths"`
				BlockedPaths []string `mapstructure:"blocked_paths"`
			}{
				AllowedPaths: []string{"./workspace", "./data"},
				BlockedPaths: []string{"/etc", "/sys", "/proc"},
			},
			Browser: struct {
				Headless  bool   `mapstructure:"headless"`
				Timeout   int    `mapstructure:"timeout"`
				UserAgent string `mapstructure:"user_agent"`
			}{
				Headless:  true,
				Timeout:   30,
				UserAgent: "OpenManus-Go/1.0",
			},
		},
	}
}

// Load 加载配置文件
func Load(configPath string) (*Config, error) {
	// 设置默认值
	config := DefaultConfig()

	// 设置 viper
	viper.SetConfigType("toml")

	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath("./configs")
		viper.AddConfigPath(".")
	}

	// 设置环境变量前缀
	viper.SetEnvPrefix("OPENMANUS")
	viper.AutomaticEnv()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// 配置文件不存在时使用默认配置
	}

	// 解析配置
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证 LLM 配置
	if c.LLM.Model == "" {
		return fmt.Errorf("llm.model is required")
	}
	if c.LLM.BaseURL == "" {
		return fmt.Errorf("llm.base_url is required")
	}
	if c.LLM.APIKey == "" {
		return fmt.Errorf("llm.api_key is required")
	}

	// 验证 Agent 配置
	if c.Agent.MaxSteps <= 0 {
		return fmt.Errorf("agent.max_steps must be positive")
	}

	// 验证存储配置
	if c.Storage.Type == "" {
		return fmt.Errorf("storage.type is required")
	}

	return nil
}

// ToLLMConfig 转换为 LLM 配置
func (c *Config) ToLLMConfig() *llm.Config {
	return &llm.Config{
		Model:       c.LLM.Model,
		BaseURL:     c.LLM.BaseURL,
		APIKey:      c.LLM.APIKey,
		Temperature: c.LLM.Temperature,
		MaxTokens:   c.LLM.MaxTokens,
		Timeout:     c.LLM.Timeout,
	}
}

// GetMaxDuration 获取最大持续时间
func (c *Config) GetMaxDuration() (time.Duration, error) {
	return time.ParseDuration(c.Agent.MaxDuration)
}

// GetRetryBackoff 获取重试退避时间
func (c *Config) GetRetryBackoff() (time.Duration, error) {
	return time.ParseDuration(c.Agent.RetryBackoff)
}

// Save 保存配置到文件
func (c *Config) Save(path string) error {
	viper.SetConfigFile(path)

	// 设置配置值
	viper.Set("llm", c.LLM)
	viper.Set("agent", c.Agent)
	viper.Set("runflow", c.RunFlow)
	viper.Set("server", c.Server)
	viper.Set("storage", c.Storage)
	viper.Set("tools", c.Tools)

	return viper.WriteConfig()
}

// GetConfigTemplate 获取配置模板
func GetConfigTemplate() string {
	return `# OpenManus-Go Configuration

[llm]
model = "gpt-3.5-turbo"
base_url = "https://api.openai.com/v1"
api_key = "your-api-key-here"
temperature = 0.1
max_tokens = 4000
timeout = 30

[agent]
max_steps = 10
max_tokens = 8000
max_duration = "5m"
reflection_steps = 3
max_retries = 2
retry_backoff = "1s"

[runflow]
use_data_analysis_agent = false
enable_multi_agent = false

[server]
host = "localhost"
port = 8080

[storage]
type = "file"  # file, memory, redis, s3
base_path = "./data/traces"

[storage.redis]
addr = "localhost:6379"
password = ""
db = 0

[storage.s3]
region = "us-east-1"
bucket = "openmanus-traces"
access_key = ""
secret_key = ""

[tools.http]
timeout = 30
allowed_domains = []
blocked_domains = ["localhost", "127.0.0.1"]

[tools.filesystem]
allowed_paths = ["./workspace", "./data"]
blocked_paths = ["/etc", "/sys", "/proc"]

[tools.browser]
headless = true
timeout = 30
user_agent = "OpenManus-Go/1.0"

[tools.database.mysql]
dsn = "user:password@tcp(localhost:3306)/dbname"

[tools.database.redis]
addr = "localhost:6379"
password = ""
db = 0`
}
