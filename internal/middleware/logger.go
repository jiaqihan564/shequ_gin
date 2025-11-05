package middleware

import (
	"bytes"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"gin/internal/config"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// responseWriter 包装gin的ResponseWriter以捕获响应体
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

var (
	// 日志采样计数器
	logSampleCounter uint64
	// 是否是生产环境
	isProduction = gin.Mode() == gin.ReleaseMode
)

// getLogSampleRate 从配置或环境变量获取采样率
func getLogSampleRate(cfg *config.Config) int {
	// 优先使用环境变量
	if rate := os.Getenv("LOG_SAMPLE_RATE"); rate != "" {
		// 简单解析，实际生产环境应使用 strconv.Atoi
		if rate == "100" {
			return 100 // 100%记录
		}
		if rate == "10" {
			return 10 // 10%记录
		}
		if rate == "1" {
			return 1 // 1%记录
		}
	}
	// 从配置文件读取
	if isProduction {
		return cfg.LogExtended.SampleRateProduction
	}
	return cfg.LogExtended.SampleRateDevelopment
}

// shouldSample 判断是否应该记录详细日志
func shouldSample(logSampleRate int) bool {
	// 开发模式下总是记录
	if !isProduction {
		return true
	}

	// 生产模式下进行采样
	counter := atomic.AddUint64(&logSampleCounter, 1)
	return (counter % uint64(100/logSampleRate)) == 0
}

// shouldLogPath 判断路径是否需要详细日志（从配置读取）
func shouldLogPath(path string, skipPaths []string) bool {
	// 检查路径是否在跳过列表中
	for _, skip := range skipPaths {
		if strings.HasPrefix(path, skip) {
			return false
		}
	}
	return true
}

// LoggerMiddleware 自定义日志中间件（带采样，从配置读取）
func LoggerMiddleware(cfg *config.Config) gin.HandlerFunc {
	// 获取配置
	logSampleRate := getLogSampleRate(cfg)
	skipPaths := cfg.LogExtended.SkipPaths
	truncateSize := cfg.LogExtended.RequestBodyTruncateSize

	return func(c *gin.Context) {
		// 开始时间
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		logger := utils.GetLogger()

		// 判断是否需要详细日志
		needDetailLog := shouldLogPath(path, skipPaths) && shouldSample(logSampleRate)

		// 只在采样时记录请求详情
		var requestBody string
		var requestBodySize int64
		if needDetailLog {
			// 记录请求头部（脱敏）
			requestHeaders := make(map[string][]string)
			for k, v := range c.Request.Header {
				requestHeaders[k] = v
			}
			sanitizedHeaders := utils.SanitizeHeaders(requestHeaders)

			// 读取并记录请求体（针对POST/PUT/PATCH）
			if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
				if c.Request.Body != nil {
					bodyBytes, err := io.ReadAll(c.Request.Body)
					if err == nil {
						requestBodySize = int64(len(bodyBytes))
						// 使用对象池重新设置body（优化内存分配）
						buf := utils.GetBuffer()
						buf.Write(bodyBytes)
						c.Request.Body = io.NopCloser(buf)
						// 注意：这里不能PutBuffer，因为body还要被后续使用

						// 截断大请求体，使用配置的截断大小（减少内存占用）
						if len(bodyBytes) > truncateSize {
							requestBody = utils.TruncateString(string(bodyBytes), truncateSize)
						} else {
							requestBody = string(bodyBytes)
						}
					}
				}
			}

			logger.Debug("HTTP请求详情",
				"method", c.Request.Method,
				"path", path,
				"query", raw,
				"headers", sanitizedHeaders,
				"requestBody", requestBody,
				"requestBodySize", requestBodySize,
				"contentType", c.Request.Header.Get("Content-Type"),
				"ip", c.ClientIP(),
				"userAgent", c.Request.UserAgent(),
				"protocol", c.Request.Proto,
				"host", c.Request.Host,
				"sampled", true)
		}

		// 只在需要详细日志时包装ResponseWriter（减少性能开销）
		var blw *responseWriter
		if needDetailLog {
			// 使用对象池获取Buffer
			buf := utils.GetBuffer()
			blw = &responseWriter{
				ResponseWriter: c.Writer,
				body:           buf,
			}
			c.Writer = blw
			// 在请求结束后归还Buffer
			defer utils.PutBuffer(buf)
		}

		// 处理请求
		c.Next()

		// 计算延迟
		latency := time.Since(start)

		// 获取响应体（仅在需要时）
		var responseBody string
		var responseBodySize int
		if needDetailLog && blw != nil {
			responseBody = blw.body.String()
			responseBodySize = len(responseBody)

			// 截断大响应体（减少到512字节）
			if responseBodySize > 512 {
				responseBody = utils.TruncateString(responseBody, 512)
			}
		}

		// 构建日志字段（简化版，只保留关键信息）
		fields := map[string]interface{}{
			"status":    c.Writer.Status(),
			"method":    c.Request.Method,
			"path":      path,
			"ip":        c.ClientIP(),
			"latency":   latency.String(),
			"latencyMs": latency.Milliseconds(),
		}

		// 只在详细日志模式下添加额外字段
		if needDetailLog {
			fields["query"] = raw
			fields["user_agent"] = c.Request.UserAgent()
			fields["latencyMicros"] = latency.Microseconds()
			fields["time"] = start.Format(time.RFC3339)
			fields["requestBodySize"] = requestBodySize
			fields["responseBodySize"] = responseBodySize
			fields["responseBody"] = responseBody
			fields["contentType"] = c.Writer.Header().Get("Content-Type")
			fields["sampled"] = true
		}

		// 添加用户信息（如果已认证）- 包含IP、用户名、邮箱、地址
		if userID, exists := c.Get("userID"); exists {
			fields["user_id"] = userID
		}
		if username, exists := c.Get("username"); exists {
			fields["username"] = username
		}
		if email, exists := c.Get("email"); exists {
			fields["email"] = email
		}
		if province, exists := c.Get("province"); exists {
			fields["province"] = province
		}
		if city, exists := c.Get("city"); exists {
			fields["city"] = city
		}

		// 添加请求ID（如果存在）
		if requestID, exists := c.Get("requestID"); exists {
			fields["request_id"] = requestID
		}

		// 添加错误信息（如果有）
		if len(c.Errors) > 0 {
			fields["errors"] = c.Errors.String()
		}

		// 根据状态码选择日志级别（简化逻辑）
		status := c.Writer.Status()
		switch {
		case status >= 500:
			// 服务器错误总是记录
			logger.Error("HTTP请求失败", fields)
		case status >= 400:
			// 客户端错误记录警告
			logger.Warn("HTTP请求错误", fields)
		case latency > 500*time.Millisecond:
			// 慢请求总是记录
			logger.Warn("慢请求检测", fields)
		case needDetailLog:
			// 采样的正常请求记录Info
			logger.Info("HTTP请求完成", fields)
		default:
			// 其他正常请求只记录Debug（生产环境通常不会输出）
			logger.Debug("HTTP请求完成", fields)
		}
	}
}
