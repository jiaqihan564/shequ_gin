package middleware

import (
	"gin/internal/config"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// AdminMiddleware 管理员权限验证中间件
func AdminMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取当前用户名
		username, exists := c.Get("username")
		if !exists {
			utils.GetLogger().Warn("管理员验证失败：无法获取用户名",
				"path", c.Request.URL.Path,
				"ip", c.ClientIP())
			utils.ForbiddenResponse(c, "需要管理员权限")
			c.Abort()
			return
		}

		usernameStr, ok := username.(string)
		if !ok {
			utils.GetLogger().Warn("管理员验证失败：用户名类型错误",
				"path", c.Request.URL.Path,
				"ip", c.ClientIP())
			utils.ForbiddenResponse(c, "需要管理员权限")
			c.Abort()
			return
		}

		// 检查是否在管理员列表中
		isAdmin := false
		for _, adminUsername := range cfg.Admin.Usernames {
			if adminUsername == usernameStr {
				isAdmin = true
				break
			}
		}

		if !isAdmin {
			utils.GetLogger().Warn("管理员验证失败：用户不是管理员",
				"username", usernameStr,
				"path", c.Request.URL.Path,
				"ip", c.ClientIP())
			utils.ForbiddenResponse(c, "需要管理员权限")
			c.Abort()
			return
		}

		utils.GetLogger().Debug("管理员验证成功",
			"username", usernameStr,
			"path", c.Request.URL.Path)

		// 在上下文中标记为管理员
		c.Set("isAdmin", true)
		c.Next()
	}
}
