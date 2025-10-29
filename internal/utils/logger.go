package utils

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gin/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

// levelRotateWriter 按级别和日期轮转的写入器
// 目录结构: log/2025-10-29/info.log
type levelRotateWriter struct {
	baseDir     string   // 基础目录 (log)
	level       string   // 日志级别 (debug/info/warn/error/fatal)
	currentDate string   // 当前日期 (2025-10-29)
	currentDir  string   // 当前目录 (log/2025-10-29)
	file        *os.File // 当前文件句柄
	mu          sync.Mutex
	compressOld bool // 是否压缩旧文件
}

// newLevelRotateWriter 创建按级别轮转的写入器
func newLevelRotateWriter(baseDir, level string, compressOld bool) (*levelRotateWriter, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}

	w := &levelRotateWriter{
		baseDir:     baseDir,
		level:       strings.ToLower(level),
		compressOld: compressOld,
	}

	// 初始化时创建当天的文件
	if err := w.rotateIfNeeded(); err != nil {
		return nil, err
	}

	// 如果启用压缩，异步扫描并压缩旧日志
	if compressOld {
		go w.compressOldLogs()
	}

	return w, nil
}

// Write 实现 io.Writer 接口
func (w *levelRotateWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 检查是否需要轮转
	if err := w.rotateIfNeeded(); err != nil {
		return 0, err
	}

	return w.file.Write(p)
}

// rotateIfNeeded 检查并执行日志轮转
func (w *levelRotateWriter) rotateIfNeeded() error {
	dateStr := time.Now().Format("2006-01-02")

	// 如果日期没变且文件已打开，无需轮转
	if w.file != nil && dateStr == w.currentDate {
		return nil
	}

	// 保存旧文件路径用于压缩
	var oldFilePath string
	if w.file != nil {
		oldFilePath = filepath.Join(w.currentDir, w.level+".log")
		w.file.Close()
		w.file = nil
	}

	// 创建新日期目录
	dateDir := filepath.Join(w.baseDir, dateStr)
	if err := os.MkdirAll(dateDir, 0755); err != nil {
		return err
	}

	// 打开新日期的日志文件
	filename := filepath.Join(dateDir, w.level+".log")
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	w.file = f
	w.currentDate = dateStr
	w.currentDir = dateDir

	// 异步压缩旧文件
	if oldFilePath != "" && w.compressOld {
		go w.compressFile(oldFilePath)
	}

	return nil
}

// compressOldLogs 扫描并压缩旧日志（初始化时调用）
func (w *levelRotateWriter) compressOldLogs() {
	// 读取 log 目录下的所有子目录
	entries, err := os.ReadDir(w.baseDir)
	if err != nil {
		return
	}

	today := time.Now().Format("2006-01-02")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		// 跳过今天的目录
		if dirName == today {
			continue
		}

		// 检查是否是日期格式的目录
		if _, err := time.Parse("2006-01-02", dirName); err != nil {
			continue
		}

		// 压缩该目录下的当前级别日志文件
		logPath := filepath.Join(w.baseDir, dirName, w.level+".log")
		if _, err := os.Stat(logPath); err == nil {
			w.compressFile(logPath)
		}
	}
}

// compressFile 压缩日志文件
func (w *levelRotateWriter) compressFile(srcPath string) {
	// 检查源文件是否存在
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return
	}

	// 检查是否已经压缩过
	gzPath := srcPath + ".gz"
	if _, err := os.Stat(gzPath); err == nil {
		// 已存在压缩文件，尝试删除原文件（带重试）
		w.removeFileWithRetry(srcPath, 3)
		return
	}

	// 打开源文件
	srcFile, err := os.Open(srcPath)
	if err != nil {
		fmt.Printf("压缩日志失败: 无法打开源文件 %s: %v\n", srcPath, err)
		return
	}

	// 创建压缩文件
	gzFile, err := os.Create(gzPath)
	if err != nil {
		srcFile.Close()
		fmt.Printf("压缩日志失败: 无法创建压缩文件 %s: %v\n", gzPath, err)
		return
	}

	// 创建 gzip writer
	gzWriter := gzip.NewWriter(gzFile)

	// 复制数据
	_, copyErr := io.Copy(gzWriter, srcFile)

	// 关闭所有句柄
	srcFile.Close()
	gzWriter.Close()
	gzFile.Close()

	if copyErr != nil {
		fmt.Printf("压缩日志失败: 复制数据失败 %s: %v\n", srcPath, copyErr)
		os.Remove(gzPath) // 删除不完整的压缩文件
		return
	}

	// 延迟后删除原文件（确保文件句柄完全释放）
	time.Sleep(100 * time.Millisecond)
	w.removeFileWithRetry(srcPath, 3)
}

// removeFileWithRetry 带重试的文件删除
func (w *levelRotateWriter) removeFileWithRetry(path string, maxRetries int) {
	for i := 0; i < maxRetries; i++ {
		if err := os.Remove(path); err == nil {
			return
		} else if os.IsNotExist(err) {
			return
		}
		// Windows 文件锁问题，等待后重试
		time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
	}
	// 最后一次尝试，不再打印警告（避免日志噪音）
	os.Remove(path)
}

// Close 关闭写入器
func (w *levelRotateWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		err := w.file.Close()
		w.file = nil
		return err
	}
	return nil
}

// ZapLogger 基于 Zap 的日志器实现
type ZapLogger struct {
	logger  *zap.Logger
	sugar   *zap.SugaredLogger
	config  *config.LogConfig
	writers []*levelRotateWriter // 持有所有写入器以便关闭
	mu      sync.Mutex
}

// NewLogger 创建新的日志器
func NewLogger(cfg *config.LogConfig) (Logger, error) {
	// 确定最小日志级别
	var minLevel zapcore.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		minLevel = zapcore.DebugLevel
	case "info":
		minLevel = zapcore.InfoLevel
	case "warn":
		minLevel = zapcore.WarnLevel
	case "error":
		minLevel = zapcore.ErrorLevel
	default:
		minLevel = zapcore.InfoLevel
	}

	// 创建编码器配置
	var encoderConfig zapcore.EncoderConfig
	if cfg.Format == "json" {
		encoderConfig = zap.NewProductionEncoderConfig()
	} else {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}

	// 自定义时间格式
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.LevelKey = "level"
	encoderConfig.MessageKey = "message"
	encoderConfig.CallerKey = "caller"
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// 创建编码器
	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	zapLogger := &ZapLogger{
		config:  cfg,
		writers: make([]*levelRotateWriter, 0),
	}

	var cores []zapcore.Core

	if cfg.Output == "file" {
		// 文件输出：为每个级别创建独立的 Core
		levels := []struct {
			name  string
			level zapcore.Level
		}{
			{"debug", zapcore.DebugLevel},
			{"info", zapcore.InfoLevel},
			{"warn", zapcore.WarnLevel},
			{"error", zapcore.ErrorLevel},
			{"fatal", zapcore.FatalLevel},
		}

		for _, l := range levels {
			// 创建级别专用的轮转写入器
			w, err := newLevelRotateWriter("log", l.name, true)
			if err != nil {
				return nil, fmt.Errorf("创建 %s 级别日志写入器失败: %w", l.name, err)
			}
			zapLogger.writers = append(zapLogger.writers, w)

			// 创建级别过滤器：只记录当前级别的日志
			levelFilter := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
				return lvl == l.level && lvl >= minLevel
			})

			// 创建 Core
			core := zapcore.NewCore(encoder, zapcore.AddSync(w), levelFilter)
			cores = append(cores, core)
		}
	} else {
		// stdout 输出：单个 Core
		core := zapcore.NewCore(
			encoder,
			zapcore.AddSync(os.Stdout),
			minLevel,
		)
		cores = append(cores, core)
	}

	// 组合所有 Core
	core := zapcore.NewTee(cores...)

	// 创建 logger，添加调用者信息和堆栈跟踪
	zapLogger.logger = zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(2),                  // 跳过包装层
		zap.AddStacktrace(zapcore.FatalLevel), // 只有 Fatal 级别记录堆栈，减少日志冗余
	)
	zapLogger.sugar = zapLogger.logger.Sugar()

	return zapLogger, nil
}

// convertMapToFields 将 map 转换为键值对切片
func convertMapToFields(m map[string]interface{}) []interface{} {
	fields := make([]interface{}, 0, len(m)*2)
	for k, v := range m {
		fields = append(fields, k, v)
	}
	return fields
}

// Info 记录信息日志
func (l *ZapLogger) Info(msg string, fields ...interface{}) {
	if len(fields) == 0 {
		l.sugar.Info(msg)
		return
	}

	// 支持 map 形式（中间件使用）
	if len(fields) == 1 {
		if m, ok := fields[0].(map[string]interface{}); ok {
			l.sugar.Infow(msg, convertMapToFields(m)...)
			return
		}
	}

	// 支持键值对形式
	l.sugar.Infow(msg, fields...)
}

// Warn 记录警告日志
func (l *ZapLogger) Warn(msg string, fields ...interface{}) {
	if len(fields) == 0 {
		l.sugar.Warn(msg)
		return
	}

	if len(fields) == 1 {
		if m, ok := fields[0].(map[string]interface{}); ok {
			l.sugar.Warnw(msg, convertMapToFields(m)...)
			return
		}
	}

	l.sugar.Warnw(msg, fields...)
}

// Error 记录错误日志
func (l *ZapLogger) Error(msg string, fields ...interface{}) {
	if len(fields) == 0 {
		l.sugar.Error(msg)
		return
	}

	if len(fields) == 1 {
		if m, ok := fields[0].(map[string]interface{}); ok {
			l.sugar.Errorw(msg, convertMapToFields(m)...)
			return
		}
	}

	l.sugar.Errorw(msg, fields...)
}

// Debug 记录调试日志
func (l *ZapLogger) Debug(msg string, fields ...interface{}) {
	if len(fields) == 0 {
		l.sugar.Debug(msg)
		return
	}

	if len(fields) == 1 {
		if m, ok := fields[0].(map[string]interface{}); ok {
			l.sugar.Debugw(msg, convertMapToFields(m)...)
			return
		}
	}

	l.sugar.Debugw(msg, fields...)
}

// Fatal 记录致命错误日志并退出程序
func (l *ZapLogger) Fatal(msg string, fields ...interface{}) {
	// Fatal 必须同步写入，确保日志不丢失
	if len(fields) == 0 {
		l.sugar.Fatal(msg)
		return
	}

	if len(fields) == 1 {
		if m, ok := fields[0].(map[string]interface{}); ok {
			l.sugar.Fatalw(msg, convertMapToFields(m)...)
			return
		}
	}

	l.sugar.Fatalw(msg, fields...)
}

// Close 关闭日志器
func (l *ZapLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 同步所有缓冲的日志
	if err := l.logger.Sync(); err != nil {
		// 在 Windows 上 Sync() 可能返回错误，但可以忽略
		// 参考: https://github.com/uber-go/zap/issues/880
	}

	// 关闭所有写入器
	for _, w := range l.writers {
		if err := w.Close(); err != nil {
			return err
		}
	}

	return nil
}

// 全局日志器实例
var (
	globalLogger Logger
	globalMu     sync.RWMutex
)

// InitLogger 初始化全局日志器
func InitLogger(cfg *config.LogConfig) error {
	logger, err := NewLogger(cfg)
	if err != nil {
		return err
	}

	globalMu.Lock()
	globalLogger = logger
	globalMu.Unlock()

	return nil
}

// GetLogger 获取全局日志器
func GetLogger() Logger {
	globalMu.RLock()
	logger := globalLogger
	globalMu.RUnlock()

	if logger == nil {
		// 如果没有初始化，创建一个默认的日志器
		cfg := &config.LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			Async:      true,
			Buffer:     1024,
			DropPolicy: "block",
		}
		logger, _ = NewLogger(cfg)

		globalMu.Lock()
		if globalLogger == nil {
			globalLogger = logger
		}
		globalMu.Unlock()
	}

	return globalLogger
}

// CloseLogger 优雅关闭全局日志器
func CloseLogger() error {
	globalMu.Lock()
	logger := globalLogger
	globalLogger = nil
	globalMu.Unlock()

	if logger != nil {
		return logger.Close()
	}
	return nil
}

// 便捷函数
func Info(msg string, fields ...interface{})  { GetLogger().Info(msg, fields...) }
func Warn(msg string, fields ...interface{})  { GetLogger().Warn(msg, fields...) }
func Error(msg string, fields ...interface{}) { GetLogger().Error(msg, fields...) }
func Debug(msg string, fields ...interface{}) { GetLogger().Debug(msg, fields...) }
func Fatal(msg string, fields ...interface{}) { GetLogger().Fatal(msg, fields...) }

// ==================== 日志辅助函数 ====================

// SanitizeHeaders 脱敏HTTP头部
func SanitizeHeaders(headers map[string][]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range headers {
		if k == "Authorization" && len(v) > 0 {
			result[k] = SanitizeAuthHeader(v[0])
		} else if k == "Cookie" && len(v) > 0 {
			result[k] = "***"
		} else {
			result[k] = v
		}
	}
	return result
}

// SanitizeParams 脱敏查询参数（隐藏密码等敏感字段）
func SanitizeParams(params map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	sensitiveKeys := map[string]bool{
		"password":     true,
		"old_password": true,
		"new_password": true,
		"token":        true,
		"secret":       true,
		"api_key":      true,
	}

	for k, v := range params {
		if sensitiveKeys[k] {
			result[k] = "***"
		} else {
			result[k] = v
		}
	}
	return result
}
