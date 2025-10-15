package middleware

import (
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// ValidateJSON 验证JSON请求体的中间件
func ValidateJSON(validatorFunc func(interface{}) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload interface{}
		if err := c.ShouldBindJSON(&payload); err != nil {
			utils.ValidationErrorResponse(c, "请求参数错误: "+err.Error())
			c.Abort()
			return
		}

		if validatorFunc != nil {
			if err := validatorFunc(payload); err != nil {
				utils.ValidationErrorResponse(c, err.Error())
				c.Abort()
				return
			}
		}

		c.Set("payload", payload)
		c.Next()
	}
}

// RequireAuth 要求认证的中间件（简化版）
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := utils.GetUserIDFromContext(c)
		if err != nil {
			utils.GetLogger().Warn("未认证访问", "path", c.Request.URL.Path, "ip", c.ClientIP())
			utils.UnauthorizedResponse(c, "需要认证")
			c.Abort()
			return
		}
		c.Next()
	}
}

// SanitizeInput 输入清理中间件
func SanitizeInput() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 可以在这里添加全局的输入清理逻辑
		// 例如：清理XSS、SQL注入等
		c.Next()
	}
}
