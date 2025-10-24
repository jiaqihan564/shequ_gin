package middleware

import (
	"strings"
	"time"

	"gin/internal/config"
	"gin/internal/models"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// AuthMiddleware JWT认证中间件
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试从Authorization头获取token
		authHeader := c.GetHeader("Authorization")
		var tokenString string

		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = authHeader[7:]
		} else {
			// 如果没有Authorization头，尝试从URL参数获取token（用于下载等场景）
			tokenString = c.Query("token")
		}

		if tokenString == "" {
			utils.GetLogger().Warn("认证失败：缺少token", "ip", c.ClientIP(), "path", c.Request.URL.Path)
			utils.UnauthorizedResponse(c, "缺少Authorization头或token参数")
			c.Abort()
			return
		}
		claims := &models.Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// 验证签名方法
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.JWT.SecretKey), nil
		})

		if err != nil {
			utils.GetLogger().Warn("认证失败：token解析错误", "error", err.Error(), "ip", c.ClientIP(), "path", c.Request.URL.Path)
			utils.UnauthorizedResponse(c, "无效的token")
			c.Abort()
			return
		}

		if !token.Valid {
			utils.GetLogger().Warn("认证失败：token无效", "ip", c.ClientIP(), "path", c.Request.URL.Path)
			utils.UnauthorizedResponse(c, "无效的token")
			c.Abort()
			return
		}

		// 检查token是否过期
		if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
			utils.GetLogger().Warn("认证失败：token已过期", "ip", c.ClientIP(), "path", c.Request.URL.Path)
			utils.UnauthorizedResponse(c, "token已过期")
			c.Abort()
			return
		}

		// 从claims中获取用户信息
		userID := claims.Subject
		if userID == "" {
			utils.GetLogger().Warn("认证失败：token中缺少用户ID", "ip", c.ClientIP(), "path", c.Request.URL.Path)
			utils.UnauthorizedResponse(c, "无效的token")
			c.Abort()
			return
		}

		// 验证issuer
		if claims.Issuer != cfg.JWT.Issuer {
			utils.GetLogger().Warn("认证失败：token issuer不匹配", "expected", cfg.JWT.Issuer, "actual", claims.Issuer, "ip", c.ClientIP(), "path", c.Request.URL.Path)
			utils.UnauthorizedResponse(c, "无效的token")
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("userID", userID)
		// 从自定义claims中获取用户名
		if claims.Username != "" {
			c.Set("username", claims.Username)
		}

		// 设置请求ID用于追踪
		if claims.ID != "" {
			c.Set("requestID", claims.ID)
		}

		utils.GetLogger().Debug("用户认证成功", "userID", userID, "ip", c.ClientIP(), "path", c.Request.URL.Path)
		c.Next()
	}
}
