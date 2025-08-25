package builtin

import (
	"fmt"
	"time"

	"openmanus-go/pkg/config"
	"openmanus-go/pkg/tool"
)

// RegisterBuiltinTools 注册所有内置工具
func RegisterBuiltinTools(registry *tool.Registry, cfg *config.Config) error {
	// 注册 HTTP 工具
	httpTool := NewHTTPTool()
	if err := registry.Register(httpTool); err != nil {
		return fmt.Errorf("failed to register http tool: %w", err)
	}

	// 注册高级 HTTP 客户端工具
	httpClientTool := NewHTTPClientTool()
	if err := registry.Register(httpClientTool); err != nil {
		return fmt.Errorf("failed to register http_client tool: %w", err)
	}

	// 注册文件系统工具
	fsTool := NewFileSystemTool(
		cfg.Tools.FileSystem.AllowedPaths,
		cfg.Tools.FileSystem.BlockedPaths,
	)
	if err := registry.Register(fsTool); err != nil {
		return fmt.Errorf("failed to register fs tool: %w", err)
	}

	// 注册文件复制工具
	fileCopyTool := NewFileCopyTool()
	if err := registry.Register(fileCopyTool); err != nil {
		return fmt.Errorf("failed to register file_copy tool: %w", err)
	}

	// 注册 Redis 工具（如果配置了）
	if cfg.Tools.Database.Redis.Addr != "" {
		redisTool := NewRedisTool(
			cfg.Tools.Database.Redis.Addr,
			cfg.Tools.Database.Redis.Password,
			cfg.Tools.Database.Redis.DB,
		)
		if err := registry.Register(redisTool); err != nil {
			return fmt.Errorf("failed to register redis tool: %w", err)
		}
	}

	// 注册 MySQL 工具（如果配置了）
	if cfg.Tools.Database.MySQL.DSN != "" {
		mysqlTool, err := NewMySQLTool(cfg.Tools.Database.MySQL.DSN)
		if err != nil {
			return fmt.Errorf("failed to create mysql tool: %w", err)
		}
		if err := registry.Register(mysqlTool); err != nil {
			return fmt.Errorf("failed to register mysql tool: %w", err)
		}
	}

	// 注册浏览器工具
	timeout := time.Duration(cfg.Tools.Browser.Timeout) * time.Second
	browserTool, err := NewBrowserTool(cfg.Tools.Browser.Headless, timeout)
	if err != nil {
		return fmt.Errorf("failed to create browser tool: %w", err)
	}
	if err := registry.Register(browserTool); err != nil {
		return fmt.Errorf("failed to register browser tool: %w", err)
	}

	// 注册爬虫工具
	crawlerTool := NewCrawlerTool(
		cfg.Tools.Browser.UserAgent,
		cfg.Tools.HTTP.AllowedDomains,
		cfg.Tools.HTTP.BlockedDomains,
	)
	if err := registry.Register(crawlerTool); err != nil {
		return fmt.Errorf("failed to register crawler tool: %w", err)
	}

	// 注册 MCP 工具（桥接外部 MCP server）
	if err := registry.Register(NewMCPListToolsTool(cfg)); err != nil {
		return fmt.Errorf("failed to register mcp_list_tools tool: %w", err)
	}
	if err := registry.Register(NewMCPCallTool(cfg)); err != nil {
		return fmt.Errorf("failed to register mcp_call tool: %w", err)
	}
	if err := registry.Register(NewMCPAutoTool(cfg)); err != nil {
		return fmt.Errorf("failed to register mcp_auto tool: %w", err)
	}

	return nil
}

// GetBuiltinToolsList 获取内置工具列表
func GetBuiltinToolsList() []string {
	return []string{
		"http",
		"http_client",
		"fs",
		"file_copy",
		"redis",
		"mysql",
		"browser",
		"crawler",
	}
}

// CreateToolFromConfig 根据配置创建特定工具
func CreateToolFromConfig(toolName string, cfg *config.Config) (tool.Tool, error) {
	switch toolName {
	case "http":
		return NewHTTPTool(), nil
	case "http_client":
		return NewHTTPClientTool(), nil
	case "fs":
		return NewFileSystemTool(
			cfg.Tools.FileSystem.AllowedPaths,
			cfg.Tools.FileSystem.BlockedPaths,
		), nil
	case "file_copy":
		return NewFileCopyTool(), nil
	case "redis":
		if cfg.Tools.Database.Redis.Addr == "" {
			return nil, fmt.Errorf("redis configuration is missing")
		}
		return NewRedisTool(
			cfg.Tools.Database.Redis.Addr,
			cfg.Tools.Database.Redis.Password,
			cfg.Tools.Database.Redis.DB,
		), nil
	case "mysql":
		if cfg.Tools.Database.MySQL.DSN == "" {
			return nil, fmt.Errorf("mysql configuration is missing")
		}
		return NewMySQLTool(cfg.Tools.Database.MySQL.DSN)
	case "browser":
		timeout := time.Duration(cfg.Tools.Browser.Timeout) * time.Second
		return NewBrowserTool(cfg.Tools.Browser.Headless, timeout)
	case "crawler":
		return NewCrawlerTool(
			cfg.Tools.Browser.UserAgent,
			cfg.Tools.HTTP.AllowedDomains,
			cfg.Tools.HTTP.BlockedDomains,
		), nil
	default:
		return nil, fmt.Errorf("unknown builtin tool: %s", toolName)
	}
}

// ValidateToolConfig 验证工具配置
func ValidateToolConfig(toolName string, cfg *config.Config) error {
	switch toolName {
	case "redis":
		if cfg.Tools.Database.Redis.Addr == "" {
			return fmt.Errorf("redis.addr is required")
		}
	case "mysql":
		if cfg.Tools.Database.MySQL.DSN == "" {
			return fmt.Errorf("mysql.dsn is required")
		}
	case "http", "http_client", "fs", "file_copy", "browser", "crawler":
		// 这些工具有默认配置，无需特殊验证
		return nil
	default:
		return fmt.Errorf("unknown tool: %s", toolName)
	}
	return nil
}
