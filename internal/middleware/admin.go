package middleware

import (
	"gin/internal/config"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// AdminMiddleware 管理员权限验证中间件（优化：使用map代替循环查找）
func AdminMiddleware(cfg *config.Config) gin.HandlerFunc {
	// 预先构建管理员map（O(1)查找，优于O(n)循环）
	adminMap := make(map[string]bool, len(cfg.Admin.Usernames))
	for _, adminUsername := range cfg.Admin.Usernames {
		adminMap[adminUsername] = true
	}

	logger := utils.GetLogger()

	return func(c *gin.Context) {
		// 获取当前用户名
		username, exists := c.Get("username")
		if !exists {
			logger.Warn("管理员验证失败：无法获取用户名",
				"path", c.Request.URL.Path,
				"ip", c.ClientIP())
			utils.ForbiddenResponse(c, "需要管理员权限")
			c.Abort()
			return
		}

		usernameStr, ok := username.(string)
		if !ok {
			logger.Warn("管理员验证失败：用户名类型错误",
				"path", c.Request.URL.Path,
				"ip", c.ClientIP())
			utils.ForbiddenResponse(c, "需要管理员权限")
			c.Abort()
			return
		}

		// 使用map查找（O(1)，比循环O(n)快）
		if !adminMap[usernameStr] {
			logger.Warn("管理员验证失败：用户不是管理员",
				"username", usernameStr,
				"path", c.Request.URL.Path,
				"ip", c.ClientIP())
			utils.ForbiddenResponse(c, "需要管理员权限")
			c.Abort()
			return
		}

		logger.Debug("管理员验证成功",
			"username", usernameStr,
			"path", c.Request.URL.Path)

		// 在上下文中标记为管理员
		c.Set("isAdmin", true)
		c.Next()
	}
}
