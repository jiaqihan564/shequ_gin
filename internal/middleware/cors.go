package middleware

import (
	"strings"

	"gin/internal/config"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware CORS中间件
func CORSMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置允许的源
		origin := c.Request.Header.Get("Origin")
		if isOriginAllowed(origin, cfg.CORS.AllowOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
		} else if len(cfg.CORS.AllowOrigins) > 0 && cfg.CORS.AllowOrigins[0] == "*" {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		// 设置允许的方法
		if len(cfg.CORS.AllowMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", strings.Join(cfg.CORS.AllowMethods, ", "))
		}

		// 设置允许的头部
		if len(cfg.CORS.AllowHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", strings.Join(cfg.CORS.AllowHeaders, ", "))
		}

		// 设置是否允许凭证
		if cfg.CORS.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// isOriginAllowed 检查源是否被允许
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false
	}

	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}

	return false
}
