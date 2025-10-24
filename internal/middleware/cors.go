package middleware

import (
	"strings"

	"gin/internal/config"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware CORS中间件（优化：缓存header字符串）
func CORSMiddleware(cfg *config.Config) gin.HandlerFunc {
	// 预先拼接header字符串（避免每次请求都拼接）
	allowMethods := strings.Join(cfg.CORS.AllowMethods, ", ")
	allowHeaders := strings.Join(cfg.CORS.AllowHeaders, ", ")
	allowCredentials := "false"
	if cfg.CORS.AllowCredentials {
		allowCredentials = "true"
	}

	return func(c *gin.Context) {
		// 设置允许的源
		origin := c.Request.Header.Get("Origin")
		if isOriginAllowed(origin, cfg.CORS.AllowOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
		} else if len(cfg.CORS.AllowOrigins) > 0 && cfg.CORS.AllowOrigins[0] == "*" {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		// 设置允许的方法（使用预拼接的字符串）
		if allowMethods != "" {
			c.Header("Access-Control-Allow-Methods", allowMethods)
		}

		// 设置允许的头部（使用预拼接的字符串）
		if allowHeaders != "" {
			c.Header("Access-Control-Allow-Headers", allowHeaders)
		}

		// 设置是否允许凭证（使用预计算的字符串）
		c.Header("Access-Control-Allow-Credentials", allowCredentials)

		// 设置预检请求的缓存时间
		c.Header("Access-Control-Max-Age", "86400") // 24小时

		// 设置暴露的头部（性能优化：预定义常量）
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type, Content-Disposition, Content-Range, Accept-Ranges")

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			utils.GetLogger().Debug("处理CORS预检请求",
				"origin", origin,
				"method", c.Request.Method,
				"ip", c.ClientIP(),
				"allowOrigin", c.Writer.Header().Get("Access-Control-Allow-Origin"),
				"allowMethods", c.Writer.Header().Get("Access-Control-Allow-Methods"),
				"allowHeaders", c.Writer.Header().Get("Access-Control-Allow-Headers"))
			c.AbortWithStatus(204)
			return
		}

		// 对于非OPTIONS请求，也要确保CORS头信息存在
		utils.GetLogger().Debug("处理CORS请求",
			"origin", origin,
			"method", c.Request.Method,
			"ip", c.ClientIP(),
			"allowOrigin", c.Writer.Header().Get("Access-Control-Allow-Origin"))

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
