package middleware

import (
	"strings"

	"gin/internal/config"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware CORS中间件（优化版）
func CORSMiddleware(cfg *config.Config) gin.HandlerFunc {
	// 预计算CORS头（性能优化）
	allowOrigin := "*"
	allowMethods := strings.Join(cfg.CORS.AllowMethods, ", ")
	allowHeaders := strings.Join(cfg.CORS.AllowHeaders, ", ")
	allowCredentials := "false"
	if cfg.CORS.AllowCredentials {
		allowCredentials = "true"
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 设置CORS头
		if len(cfg.CORS.AllowOrigins) > 0 && cfg.CORS.AllowOrigins[0] != "*" {
			// 检查特定源
			for _, allowed := range cfg.CORS.AllowOrigins {
				if allowed == origin {
					c.Header("Access-Control-Allow-Origin", origin)
					break
				}
			}
		} else {
			c.Header("Access-Control-Allow-Origin", allowOrigin)
		}

		c.Header("Access-Control-Allow-Methods", allowMethods)
		c.Header("Access-Control-Allow-Headers", allowHeaders)
		c.Header("Access-Control-Allow-Credentials", allowCredentials)
		c.Header("Access-Control-Max-Age", "86400")

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
