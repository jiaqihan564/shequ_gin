package middleware

import (
	"bytes"
	"io"
	"time"

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

// LoggerMiddleware 自定义日志中间件
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		logger := utils.GetLogger()

		// 记录请求头部（脱敏）
		requestHeaders := make(map[string][]string)
		for k, v := range c.Request.Header {
			requestHeaders[k] = v
		}
		sanitizedHeaders := utils.SanitizeHeaders(requestHeaders)

		// 读取并记录请求体（针对POST/PUT/PATCH）
		var requestBody string
		var requestBodySize int64
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			if c.Request.Body != nil {
				bodyBytes, err := io.ReadAll(c.Request.Body)
				if err == nil {
					requestBodySize = int64(len(bodyBytes))
					// 重新设置body以供后续处理使用
					c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

					// 截断大请求体，只记录前1024字节
					if len(bodyBytes) > 1024 {
						requestBody = utils.TruncateString(string(bodyBytes), 1024)
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
			"remoteAddr", c.Request.RemoteAddr)

		// 包装ResponseWriter以捕获响应体
		blw := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = blw

		// 处理请求
		c.Next()

		// 计算延迟
		latency := time.Since(start)

		// 获取响应体
		responseBody := blw.body.String()
		responseBodySize := len(responseBody)

		// 截断大响应体
		if responseBodySize > 1024 {
			responseBody = utils.TruncateString(responseBody, 1024)
		}

		// 构建日志字段
		fields := map[string]interface{}{
			"status":           c.Writer.Status(),
			"method":           c.Request.Method,
			"path":             path,
			"query":            raw,
			"ip":               c.ClientIP(),
			"user_agent":       c.Request.UserAgent(),
			"latency":          latency.String(),
			"latencyMs":        latency.Milliseconds(),
			"latencyMicros":    latency.Microseconds(),
			"time":             start.Format(time.RFC3339),
			"requestBodySize":  requestBodySize,
			"responseBodySize": responseBodySize,
			"responseBody":     responseBody,
			"contentType":      c.Writer.Header().Get("Content-Type"),
			"responseHeaders":  c.Writer.Header(),
		}

		// 添加用户信息（如果已认证）
		if userID, exists := c.Get("userID"); exists {
			fields["user_id"] = userID
		}
		if username, exists := c.Get("username"); exists {
			fields["username"] = username
		}

		// 添加请求ID（如果存在）
		if requestID, exists := c.Get("requestID"); exists {
			fields["request_id"] = requestID
		}

		// 添加错误信息（如果有）
		if len(c.Errors) > 0 {
			fields["errors"] = c.Errors.String()
		}

		// 根据状态码选择日志级别
		status := c.Writer.Status()
		switch {
		case status >= 500:
			logger.Error("HTTP请求完成", fields)
		case status >= 400:
			logger.Warn("HTTP请求完成", fields)
		case status >= 200 && status < 300:
			// 记录所有2xx请求（调试模式）
			logger.Info("HTTP请求完成", fields)
			// 只在慢请求时记录详细信息
			if latency > 500*time.Millisecond {
				logger.Warn("慢请求检测", fields)
			}
		case status >= 300 && status < 400:
			logger.Debug("HTTP重定向", fields)
		}
	}
}
