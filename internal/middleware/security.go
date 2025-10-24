package middleware

import (
	"net/http"

	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware 添加安全响应头
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防止点击劫持
		c.Header("X-Frame-Options", "DENY")
		// 防止MIME类型嗅探
		c.Header("X-Content-Type-Options", "nosniff")
		// XSS保护
		c.Header("X-XSS-Protection", "1; mode=block")
		// 强制HTTPS（生产环境推荐）
		// c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		// 内容安全策略（可根据需要配置）
		c.Header("Content-Security-Policy", "default-src 'self'")
		// 引用策略
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		// 权限策略
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}

// RequestSizeLimitMiddleware 限制请求体大小
func RequestSizeLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 限制请求体大小，防止大文件攻击
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

		c.Next()
	}
}

// PanicRecoveryMiddleware 自定义panic恢复中间件
func PanicRecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger := utils.GetLogger()
				logger.Error("请求处理发生panic",
					"error", err,
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
					"ip", c.ClientIP())

				// 返回500错误
				utils.InternalServerErrorResponse(c, "服务器内部错误")
				c.Abort()
			}
		}()
		c.Next()
	}
}
