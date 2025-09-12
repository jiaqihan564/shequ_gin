package bootstrap

import (
	"gin/internal/config"
	"gin/internal/services"
)

// Container 应用容器（简单装配）
type Container struct {
	DB      *services.Database
	Auth    services.AuthServiceInterface
	UserSvc services.UserServiceInterface
	Storage services.StorageClient
}

// New 构建容器
func New(cfg *config.Config, db *services.Database) (*Container, error) {
	userRepo := services.NewUserRepository(db)
	authService := services.NewAuthService(cfg, userRepo)
	userService := services.NewUserService(userRepo)
	storageService, err := services.NewStorageService(cfg)
	if err != nil {
		// 允许存储失败返回 nil，由上层决定是否禁用上传
		storageService = nil
	}

	return &Container{
		DB:      db,
		Auth:    authService,
		UserSvc: userService,
		Storage: storageService,
	}, nil
}
