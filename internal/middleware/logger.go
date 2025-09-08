package middleware

import (
	"time"

	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// LoggerMiddleware 自定义日志中间件
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 计算延迟
		latency := time.Since(start)

		// 获取日志记录器
		logger := utils.GetLogger()

		// 构建日志字段
		fields := map[string]interface{}{
			"status":     c.Writer.Status(),
			"method":     c.Request.Method,
			"path":       path,
			"query":      raw,
			"ip":         c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
			"latency":    latency.String(),
			"time":       start.Format(time.RFC3339),
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

		// 根据状态码选择日志级别
		status := c.Writer.Status()
		switch {
		case status >= 500:
			logger.Error("HTTP请求", fields)
		case status >= 400:
			logger.Warn("HTTP请求", fields)
		case status >= 200 && status < 300:
			// 只记录2xx状态码的慢请求
			if latency > 500*time.Millisecond {
				logger.Info("慢请求", fields)
			}
		}
	}
}
