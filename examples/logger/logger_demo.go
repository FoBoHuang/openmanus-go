package examples

import (
	"openmanus-go/pkg/logger"
	"time"
)

// 简单演示如何调用全局 logger
func Demo() {
	// 默认从环境变量 OPENMANUS_LOG_LEVEL / OUTPUT / FILEPATH 读取，未设置则使用配置或默认
	logger.Init(logger.GetLevelFromEnv())

	defer logger.Sync()

	// 便捷方法
	logger.Info("hello logger (Info)")
	logger.Infof("hello %s (%s)", "logger", "Infof")
	logger.Infow("hello with fields", "feature", "demo", "ts", time.Now().Format(time.RFC3339))

	logger.Debug("this is a debug message")
	logger.Warn("this is a warning")
	logger.Error("this is an error")
}
