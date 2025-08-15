package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type OpenAI struct {
	APIKey         string  `mapstructure:"api_key"`
	BaseURL        string  `mapstructure:"base_url"`
	Model          string  `mapstructure:"model"`
	Temperature    float64 `mapstructure:"temperature"`
	TimeoutSeconds int     `mapstructure:"timeout_seconds"`
}

type OTel struct {
	ServiceName string `mapstructure:"service_name"`
	Stdout      bool   `mapstructure:"stdout"`
}

type Log struct {
	Level string `mapstructure:"level"`
}

type Config struct {
	OpenAI OpenAI `mapstructure:"openai"`
	OTel   OTel   `mapstructure:"otel"`
	Log    Log    `mapstructure:"log"`
}

func defaultConfig() *Config {
	return &Config{
		OpenAI: OpenAI{
			BaseURL:        "https://api.openai.com/v1",
			Model:          "gpt-4o-mini",
			Temperature:    0.2,
			TimeoutSeconds: 60,
		},
		OTel: OTel{
			ServiceName: "openmanus-go",
			Stdout:      true,
		},
		Log: Log{
			Level: "info",
		},
	}
}

func InitLogger(level string) {
	lvl, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Level(lvl).With().Timestamp().Logger()
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigName("config")
	v.AddConfigPath("config")
	if home, _ := os.UserHomeDir(); home != "" {
		v.AddConfigPath(filepath.Join(home, ".openmanus"))
	}
	v.SetEnvPrefix("OPENMANUS")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	def := defaultConfig()
	v.SetDefault("openai.base_url", def.OpenAI.BaseURL)
	v.SetDefault("openai.model", def.OpenAI.Model)
	v.SetDefault("openai.temperature", def.OpenAI.Temperature)
	v.SetDefault("openai.timeout_seconds", def.OpenAI.TimeoutSeconds)
	v.SetDefault("otel.service_name", def.OTel.ServiceName)
	v.SetDefault("otel.stdout", def.OTel.Stdout)
	v.SetDefault("log.level", def.Log.Level)

	_ = v.ReadInConfig()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// prefer env key OPENAI_API_KEY
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		cfg.OpenAI.APIKey = key
	}
	if key := os.Getenv("OPENMANUS_OPENAI_API_KEY"); key != "" {
		cfg.OpenAI.APIKey = key
	}

	InitLogger(cfg.Log.Level)
	return &cfg, nil
}
