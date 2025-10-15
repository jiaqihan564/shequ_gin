package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
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
	Close() error
}

// AppLogger 应用日志器
type AppLogger struct {
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
	fatalLogger *log.Logger
	config      *config.LogConfig

	// async pipeline
	asyncEnabled bool
	queue        chan logEvent
	dropPolicy   string // block | drop_new | drop_oldest
	closed       bool
	mu           sync.Mutex
	wg           sync.WaitGroup
}

type logEvent struct {
	level  string
	msg    string
	fields []interface{}
}

// dailyRotateWriter 按日期切割写入 log 目录
type dailyRotateWriter struct {
	directory string
	file      *os.File
	current   string
	mu        sync.Mutex
}

func newDailyRotateWriter(directory string) (*dailyRotateWriter, error) {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, err
	}
	w := &dailyRotateWriter{directory: directory}
	if err := w.rotateIfNeeded(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *dailyRotateWriter) rotateIfNeeded() error {
	dateStr := time.Now().Format("2006.1.2")
	if w.file != nil && dateStr == w.current {
		return nil
	}
	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}
	filename := filepath.Join(w.directory, dateStr+".log")
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	w.file = f
	w.current = dateStr
	return nil
}

func (w *dailyRotateWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.rotateIfNeeded(); err != nil {
		return 0, err
	}
	return w.file.Write(p)
}

// NewLogger 创建新的日志器
func NewLogger(cfg *config.LogConfig) (*AppLogger, error) {
	logger := &AppLogger{
		config: cfg,
	}

	// 设置日志输出和创建日志器
	flags := log.LstdFlags | log.Lshortfile
	if cfg.Format == "json" {
		flags = 0 // JSON格式不需要时间戳和文件信息
	}

	switch cfg.Output {
	case "file":
		// 输出到与 internal 同级的工作目录下 log 目录，按日期切割
		w, werr := newDailyRotateWriter("log")
		if werr != nil {
			return nil, werr
		}
		logger.infoLogger = log.New(w, "[INFO] ", flags)
		logger.warnLogger = log.New(w, "[WARN] ", flags)
		logger.errorLogger = log.New(w, "[ERROR] ", flags)
		logger.debugLogger = log.New(w, "[DEBUG] ", flags)
		logger.fatalLogger = log.New(w, "[FATAL] ", flags)
	default:
		// 默认输出到标准输出
		logger.infoLogger = log.New(os.Stdout, "[INFO] ", flags)
		logger.warnLogger = log.New(os.Stdout, "[WARN] ", flags)
		logger.errorLogger = log.New(os.Stdout, "[ERROR] ", flags)
		logger.debugLogger = log.New(os.Stdout, "[DEBUG] ", flags)
		logger.fatalLogger = log.New(os.Stdout, "[FATAL] ", flags)
	}

	// 配置异步
	// 当 cfg.Buffer 大于 0 或 cfg.Async 为 true 时启用异步
	bufferSize := 0
	if cfg.Async {
		if cfg.Buffer > 0 {
			bufferSize = cfg.Buffer
		} else {
			bufferSize = 1024
		}
	} else if cfg.Buffer > 0 {
		bufferSize = cfg.Buffer
	}
	if bufferSize > 0 {
		logger.asyncEnabled = true
		logger.dropPolicy = cfg.DropPolicy
		if logger.dropPolicy == "" {
			logger.dropPolicy = "block"
		}
		logger.queue = make(chan logEvent, bufferSize)
		logger.wg.Add(1)
		go func() {
			defer logger.wg.Done()
			for ev := range logger.queue {
				logger.writeSync(ev.level, ev.msg, ev.fields...)
			}
		}()
	}

	return logger, nil
}

// Info 记录信息日志
func (l *AppLogger) Info(msg string, fields ...interface{}) {
	if l.config.Level == "debug" || l.config.Level == "info" {
		l.write("INFO", msg, fields...)
	}
}

// Warn 记录警告日志
func (l *AppLogger) Warn(msg string, fields ...interface{}) {
	if l.config.Level == "debug" || l.config.Level == "info" || l.config.Level == "warn" {
		l.write("WARN", msg, fields...)
	}
}

// Error 记录错误日志
func (l *AppLogger) Error(msg string, fields ...interface{}) {
	l.write("ERROR", msg, fields...)
}

// Debug 记录调试日志
func (l *AppLogger) Debug(msg string, fields ...interface{}) {
	if l.config.Level == "debug" {
		l.write("DEBUG", msg, fields...)
	}
}

// Fatal 记录致命错误日志
func (l *AppLogger) Fatal(msg string, fields ...interface{}) {
	// fatal 始终同步写入，避免在进程退出时丢失
	l.writeSync("FATAL", msg, fields...)
	os.Exit(1)
}

// write 根据配置走异步或同步
func (l *AppLogger) write(level, msg string, fields ...interface{}) {
	if l.asyncEnabled {
		l.mu.Lock()
		closed := l.closed
		dropPolicy := l.dropPolicy
		q := l.queue
		l.mu.Unlock()
		if !closed {
			ev := logEvent{level: level, msg: msg, fields: fields}
			switch dropPolicy {
			case "drop_new":
				select {
				case q <- ev:
				default:
					// drop new
				}
				return
			case "drop_oldest":
				select {
				case q <- ev:
					return
				default:
					// 丢弃一个最旧的，然后再写入
					select {
					case <-q:
					default:
					}
					select {
					case q <- ev:
					default:
						// 如果仍然失败，放弃
					}
					return
				}
			default: // block
				q <- ev
				return
			}
		}
	}
	l.writeSync(level, msg, fields...)
}

// writeSync 直接同步输出
func (l *AppLogger) writeSync(level, msg string, fields ...interface{}) {
	if l.config.Format == "json" {
		l.logJSON(level, msg, fields...)
		return
	}
	switch level {
	case "DEBUG":
		l.debugLogger.Printf(msg, fields...)
	case "INFO":
		l.infoLogger.Printf(msg, fields...)
	case "WARN":
		l.warnLogger.Printf(msg, fields...)
	case "ERROR":
		l.errorLogger.Printf(msg, fields...)
	case "FATAL":
		l.fatalLogger.Printf(msg, fields...)
	default:
		l.infoLogger.Printf(msg, fields...)
	}
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
		if len(fields) == 1 {
			if m, ok := fields[0].(map[string]interface{}); ok {
				for k, v := range m {
					entry[k] = v
				}
			}
		}
		for i := 0; i < len(fields)-1; i += 2 {
			if key, ok := fields[i].(string); ok {
				entry[key] = fields[i+1]
			}
		}
	}

	// 使用标准库的 JSON 序列化，更安全高效
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		// 如果序列化失败，回退到简单格式
		jsonStr := fmt.Sprintf(`{"timestamp":"%s","level":"%s","message":"%s","error":"json marshal failed"}`,
			time.Now().Format(time.RFC3339), level, msg)
		jsonBytes = []byte(jsonStr)
	}

	switch level {
	case "DEBUG":
		l.debugLogger.Println(string(jsonBytes))
	case "INFO":
		l.infoLogger.Println(string(jsonBytes))
	case "WARN":
		l.warnLogger.Println(string(jsonBytes))
	case "ERROR":
		l.errorLogger.Println(string(jsonBytes))
	case "FATAL":
		l.fatalLogger.Println(string(jsonBytes))
	default:
		l.infoLogger.Println(string(jsonBytes))
	}
}

// Close 关闭异步日志，确保队列消费完成
func (l *AppLogger) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	if l.asyncEnabled && l.queue != nil {
		close(l.queue)
	}
	l.mu.Unlock()
	if l.asyncEnabled {
		l.wg.Wait()
	}
	return nil
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
			Level:      "info",
			Format:     "text",
			Output:     "stdout",
			Async:      true,
			Buffer:     1024,
			DropPolicy: "block",
		}
		logger, _ := NewLogger(cfg)
		globalLogger = logger
	}
	return globalLogger
}

// 便捷函数
func Info(msg string, fields ...interface{}) { GetLogger().Info(msg, fields...) }

func Warn(msg string, fields ...interface{}) { GetLogger().Warn(msg, fields...) }

func Error(msg string, fields ...interface{}) { GetLogger().Error(msg, fields...) }

func Debug(msg string, fields ...interface{}) { GetLogger().Debug(msg, fields...) }

func Fatal(msg string, fields ...interface{}) { GetLogger().Fatal(msg, fields...) }

// CloseLogger 优雅关闭全局日志器
func CloseLogger() error {
	if globalLogger != nil {
		if c, ok := globalLogger.(interface{ Close() error }); ok {
			return c.Close()
		}
	}
	return nil
}
