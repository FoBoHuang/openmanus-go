package state

import (
	"fmt"
	"openmanus-go/pkg/config"
)

// NewStore 根据配置创建存储实例
func NewStore(cfg *config.StorageConfig) (Store, error) {
	if cfg == nil {
		return nil, fmt.Errorf("storage config is required")
	}

	switch cfg.Type {
	case "file":
		if cfg.BasePath == "" {
			cfg.BasePath = "./data/traces"
		}
		return NewFileStore(cfg.BasePath), nil

	case "memory":
		return NewMemoryStore(), nil

	case "redis":
		// TODO: 实现 Redis 存储
		return nil, fmt.Errorf("redis storage not implemented yet")

	case "s3":
		// TODO: 实现 S3 存储
		return nil, fmt.Errorf("s3 storage not implemented yet")

	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Type)
	}
}

// NewDefaultStore 创建默认的文件存储实例
func NewDefaultStore() Store {
	return NewFileStore("./data/traces")
}
