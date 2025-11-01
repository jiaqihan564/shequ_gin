package middleware

import (
	"fmt"
	"net/http"

	"gin/internal/config"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware 添加安全响应头（从配置读取）
func SecurityHeadersMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防止点击劫持
		c.Header("X-Frame-Options", cfg.SecurityHeaders.XFrameOptions)
		// 防止MIME类型嗅探
		c.Header("X-Content-Type-Options", cfg.SecurityHeaders.XContentTypeOptions)
		// XSS保护
		c.Header("X-XSS-Protection", cfg.SecurityHeaders.XXSSProtection)
		// 内容安全策略（可根据需要配置）
		c.Header("Content-Security-Policy", cfg.SecurityHeaders.ContentSecurityPolicy)
		// 引用策略
		c.Header("Referrer-Policy", cfg.SecurityHeaders.ReferrerPolicy)
		// 权限策略
		c.Header("Permissions-Policy", cfg.SecurityHeaders.PermissionsPolicy)
		// 强制HTTPS（根据配置决定是否启用）
		if cfg.SecurityHeaders.EnableHSTS {
			c.Header("Strict-Transport-Security", fmt.Sprintf("max-age=%d; includeSubDomains", cfg.SecurityHeaders.HSTSMaxAge))
		}

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
