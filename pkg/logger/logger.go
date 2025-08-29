package logger

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// globalLogger 是全局可用的 zap.Logger 指针
var globalLogger *zap.Logger

// once 确保只初始化一次
var initOnce sync.Once

// Config 定义 logger 初始化配置
type Config struct {
	Level    string
	Output   string // console | file | both
	FilePath string // 当 Output 包含 file 时生效
}

// Get 返回全局 logger。如果未初始化，则使用默认 info 级别初始化。
func Get() *zap.Logger {
	if globalLogger == nil {
		Init(GetLevelFromEnv())
	}
	return globalLogger
}

// Sugar 返回全局 SugaredLogger
func Sugar() *zap.SugaredLogger {
	return Get().Sugar()
}

// GetLevelFromEnv 读取环境变量 OPENMANUS_LOG_LEVEL 来确定日志级别，默认 info。
func GetLevelFromEnv() string {
	level := os.Getenv("OPENMANUS_LOG_LEVEL")
	if level == "" {
		return "info"
	}
	return level
}

// parseLevel 将字符串日志级别转换为 zapcore.Level。
func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// buildCore 根据配置构建 zap core（支持 console/file/both）
func buildCore(cfg Config) zapcore.Core {
	level := parseLevel(cfg.Level)
	var cores []zapcore.Core

	out := strings.ToLower(cfg.Output)
	if out == "" {
		out = "console"
	}

	// 控制台输出使用友好格式
	if out == "console" || out == "both" {
		consoleEncoderCfg := zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,        // 彩色级别
			EncodeTime:     zapcore.TimeEncoderOfLayout("15:04:05"), // 简化时间格式
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}
		consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderCfg)
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level))
	}

	// 文件输出使用JSON格式
	if out == "file" || out == "both" {
		fileEncoderCfg := zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}
		fileEncoder := zapcore.NewJSONEncoder(fileEncoderCfg)

		path := cfg.FilePath
		if path == "" {
			path = "./log/openmanus.log"
		}
		_ = os.MkdirAll(filepath.Dir(path), 0755)
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			cores = append(cores, zapcore.NewCore(fileEncoder, zapcore.AddSync(f), level))
		} else {
			// 回退到控制台
			cores = append(cores, zapcore.NewCore(zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
				TimeKey:     "ts",
				LevelKey:    "level",
				MessageKey:  "msg",
				EncodeLevel: zapcore.CapitalColorLevelEncoder,
				EncodeTime:  zapcore.TimeEncoderOfLayout("15:04:05"),
			}), zapcore.AddSync(os.Stdout), level))
		}
	}

	if len(cores) == 0 {
		cores = append(cores, zapcore.NewCore(
			zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
				TimeKey:     "ts",
				LevelKey:    "level",
				MessageKey:  "msg",
				EncodeLevel: zapcore.CapitalColorLevelEncoder,
				EncodeTime:  zapcore.TimeEncoderOfLayout("15:04:05"),
			}),
			zapcore.AddSync(os.Stdout), level))
	}

	return zapcore.NewTee(cores...)
}

// Init 使用指定级别初始化全局 logger（输出到控制台）。仅首调用生效。
func Init(level string) {
	initOnce.Do(func() {
		core := buildCore(Config{Level: level, Output: "console"})
		globalLogger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	})
}

// InitWithConfig 使用完整配置初始化全局 logger。仅首调用生效。
func InitWithConfig(cfg Config) {
	initOnce.Do(func() {
		core := buildCore(cfg)
		globalLogger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	})
}

// 便捷函数：不同级别日志
func Debug(args ...interface{})                       { Sugar().Debug(args...) }
func Debugf(template string, args ...interface{})     { Sugar().Debugf(template, args...) }
func Debugw(msg string, keysAndValues ...interface{}) { Sugar().Debugw(msg, keysAndValues...) }
func Info(args ...interface{})                        { Sugar().Info(args...) }
func Infof(template string, args ...interface{})      { Sugar().Infof(template, args...) }
func Infow(msg string, keysAndValues ...interface{})  { Sugar().Infow(msg, keysAndValues...) }
func Warn(args ...interface{})                        { Sugar().Warn(args...) }
func Warnf(template string, args ...interface{})      { Sugar().Warnf(template, args...) }
func Warnw(msg string, keysAndValues ...interface{})  { Sugar().Warnw(msg, keysAndValues...) }
func Error(args ...interface{})                       { Sugar().Error(args...) }
func Errorf(template string, args ...interface{})     { Sugar().Errorf(template, args...) }
func Errorw(msg string, keysAndValues ...interface{}) { Sugar().Errorw(msg, keysAndValues...) }

// Sync 刷新日志缓冲。
func Sync() {
	if globalLogger != nil {
		_ = globalLogger.Sync()
	}
}
