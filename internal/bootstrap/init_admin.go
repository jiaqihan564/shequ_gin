package bootstrap

import (
	"context"
	"time"

	"gin/internal/config"
	"gin/internal/models"
	"gin/internal/services"
	"gin/internal/utils"
)

// InitAdminAccounts 初始化管理员账号
// 在应用启动时自动创建配置文件中定义的管理员账号（如果不存在）
func InitAdminAccounts(cfg *config.Config, userRepo *services.UserRepository) error {
	logger := utils.GetLogger()
	ctx := context.Background()

	logger.Info("开始初始化管理员账号", "count", len(cfg.Admin.Usernames))

	// 默认密码
	defaultPassword := "admin123"
	if cfg.Admin.DefaultPassword != "" {
		defaultPassword = cfg.Admin.DefaultPassword
	}

	createdCount := 0
	existsCount := 0

	for _, username := range cfg.Admin.Usernames {
		// 检查管理员账号是否已存在
		_, err := userRepo.GetUserByUsername(ctx, username)
		if err == nil {
			// 已存在，跳过
			logger.Debug("管理员账号已存在，跳过创建", "username", username)
			existsCount++
			continue
		}

		// 不存在，创建管理员账号
		logger.Info("创建管理员账号", "username", username)

		// 加密密码
		hashedPassword, err := utils.HashPassword(defaultPassword)
		if err != nil {
			logger.Error("加密管理员密码失败", "username", username, "error", err.Error())
			continue
		}

		// 创建用户记录
		user := &models.User{
			Username:      username,
			PasswordHash:  hashedPassword,
			Email:         username + "@admin.local",
			AuthStatus:    1, // 已验证
			AccountStatus: 1, // 正常
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err = userRepo.CreateUser(ctx, user)
		if err != nil {
			logger.Error("创建管理员账号失败", "username", username, "error", err.Error())
			continue
		}

		logger.Info("管理员账号创建成功",
			"username", username,
			"email", user.Email,
			"defaultPassword", defaultPassword)
		createdCount++
	}

	logger.Info("管理员账号初始化完成",
		"total", len(cfg.Admin.Usernames),
		"created", createdCount,
		"exists", existsCount)

	return nil
}
