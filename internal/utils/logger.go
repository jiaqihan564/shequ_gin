package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gin/internal/config"
)

// Logger 日志接口
type Logger interface {
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
	Fatal(msg string, fields ...interface{})
}

// AppLogger 应用日志器
type AppLogger struct {
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
	fatalLogger *log.Logger
	config      *config.LogConfig
}

// NewLogger 创建新的日志器
func NewLogger(cfg *config.LogConfig) (*AppLogger, error) {
	logger := &AppLogger{
		config: cfg,
	}

	// 设置日志输出
	var output *os.File
	var err error

	switch cfg.Output {
	case "file":
		// 确保日志目录存在
		if err := os.MkdirAll(filepath.Dir(cfg.FilePath), 0755); err != nil {
			return nil, err
		}
		output, err = os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
	default:
		output = os.Stdout
	}

	// 创建不同级别的日志器
	flags := log.LstdFlags | log.Lshortfile
	if cfg.Format == "json" {
		flags = 0 // JSON格式不需要时间戳和文件信息
	}

	logger.infoLogger = log.New(output, "[INFO] ", flags)
	logger.warnLogger = log.New(output, "[WARN] ", flags)
	logger.errorLogger = log.New(output, "[ERROR] ", flags)
	logger.debugLogger = log.New(output, "[DEBUG] ", flags)
	logger.fatalLogger = log.New(output, "[FATAL] ", flags)

	return logger, nil
}

// Info 记录信息日志
func (l *AppLogger) Info(msg string, fields ...interface{}) {
	if l.config.Level == "debug" || l.config.Level == "info" {
		if l.config.Format == "json" {
			l.logJSON("INFO", msg, fields...)
		} else {
			l.infoLogger.Printf(msg, fields...)
		}
	}
}

// Warn 记录警告日志
func (l *AppLogger) Warn(msg string, fields ...interface{}) {
	if l.config.Level == "debug" || l.config.Level == "info" || l.config.Level == "warn" {
		if l.config.Format == "json" {
			l.logJSON("WARN", msg, fields...)
		} else {
			l.warnLogger.Printf(msg, fields...)
		}
	}
}

// Error 记录错误日志
func (l *AppLogger) Error(msg string, fields ...interface{}) {
	if l.config.Format == "json" {
		l.logJSON("ERROR", msg, fields...)
	} else {
		l.errorLogger.Printf(msg, fields...)
	}
}

// Debug 记录调试日志
func (l *AppLogger) Debug(msg string, fields ...interface{}) {
	if l.config.Level == "debug" {
		if l.config.Format == "json" {
			l.logJSON("DEBUG", msg, fields...)
		} else {
			l.debugLogger.Printf(msg, fields...)
		}
	}
}

// Fatal 记录致命错误日志
func (l *AppLogger) Fatal(msg string, fields ...interface{}) {
	if l.config.Format == "json" {
		l.logJSON("FATAL", msg, fields...)
	} else {
		l.fatalLogger.Printf(msg, fields...)
	}
	os.Exit(1)
}

// logJSON 记录JSON格式日志
func (l *AppLogger) logJSON(level, msg string, fields ...interface{}) {
	entry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     level,
		"message":   msg,
	}

	// 添加额外字段
	if len(fields) > 0 {
		for i := 0; i < len(fields); i += 2 {
			if i+1 < len(fields) {
				key, ok := fields[i].(string)
				if ok {
					entry[key] = fields[i+1]
				}
			}
		}
	}

	// 简单的JSON序列化（生产环境建议使用专门的JSON库）
	jsonStr := `{"timestamp":"` + entry["timestamp"].(string) + `","level":"` + level + `","message":"` + msg + `"`
	for key, value := range entry {
		if key != "timestamp" && key != "level" && key != "message" {
			jsonStr += `,"` + key + `":"` + toString(value) + `"`
		}
	}
	jsonStr += "}"

	l.infoLogger.Println(jsonStr)
}

// toString 将值转换为字符串
func toString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// 全局日志器实例
var globalLogger Logger

// InitLogger 初始化全局日志器
func InitLogger(cfg *config.LogConfig) error {
	logger, err := NewLogger(cfg)
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}

// GetLogger 获取全局日志器
func GetLogger() Logger {
	if globalLogger == nil {
		// 如果没有初始化，创建一个默认的日志器
		cfg := &config.LogConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		}
		logger, _ := NewLogger(cfg)
		globalLogger = logger
	}
	return globalLogger
}

// 便捷函数
func Info(msg string, fields ...interface{}) {}

func Warn(msg string, fields ...interface{}) {}

func Error(msg string, fields ...interface{}) {}

func Debug(msg string, fields ...interface{}) {}

func Fatal(msg string, fields ...interface{}) {}
