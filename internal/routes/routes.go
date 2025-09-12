package routes

import (
	"gin/internal/bootstrap"
	"gin/internal/config"
	"gin/internal/handlers"
	"gin/internal/middleware"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置路由
func SetupRoutes(cfg *config.Config, ctn *bootstrap.Container) *gin.Engine {
	// 设置Gin模式
	gin.SetMode(cfg.Server.Mode)

	r := gin.Default()

	// 添加中间件
	r.Use(middleware.RequestIDMiddleware()) // 请求ID中间件
	r.Use(middleware.CORSMiddleware(cfg))
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.MetricsMiddleware())   // 性能监控中间件
	r.Use(middleware.RateLimitMiddleware()) // 添加全局限流

	// 初始化处理器
	uploadMaxBytes := int64(cfg.Assets.MaxAvatarSizeMB) * 1024 * 1024
	authHandler := handlers.NewAuthHandler(ctn.Auth)
	userHandler := handlers.NewUserHandler(ctn.UserSvc)
	healthHandler := handlers.NewHealthHandler(ctn.DB)
	uploadHandler := handlers.NewUploadHandler(ctn.Storage, uploadMaxBytes, cfg.Assets.MaxAvatarHistory)

	// 健康检查路由
	r.GET("/health", healthHandler.Check)
	r.GET("/ready", healthHandler.Ready)
	r.GET("/live", healthHandler.Live)

	// 性能监控路由
	r.GET("/metrics", middleware.MetricsHandler)

	// API路由组
	api := r.Group("/api")
	{
		// 用户认证相关路由（使用专门的限流）
		api.POST("/auth/register", middleware.RegisterRateLimitMiddleware(), authHandler.Register)
		api.POST("/auth/login", middleware.LoginRateLimitMiddleware(), authHandler.Login)

		// 需要认证的路由
		auth := api.Group("/")
		auth.Use(middleware.AuthMiddleware(cfg))
		{
			// 用户相关路由
			auth.GET("/user/profile", userHandler.GetProfile)
			auth.PUT("/user/profile", userHandler.UpdateProfile)
			// 兼容别名：PUT /api/auth/me -> UpdateProfile
			auth.PUT("/auth/me", userHandler.UpdateProfile)
			auth.GET("/user/:id", userHandler.GetUserByID)

			// 退出登录（JWT无状态，主要用于客户端清除token）
			auth.POST("/auth/logout", authHandler.Logout)

			// 文件上传（头像）仅保留兼容别名
			auth.POST("/files/upload", uploadHandler.UploadAvatar)
			// 兼容别名：POST /api/upload -> UploadAvatar
			auth.POST("/upload", uploadHandler.UploadAvatar)

			// 历史头像查询
			auth.GET("/user/avatar/history", uploadHandler.ListAvatarHistory)
			// 兼容别名：GET /api/avatar/history
			auth.GET("/avatar/history", uploadHandler.ListAvatarHistory)
			// 兼容别名：PUT /api/auth/me/avatar -> JSON 更新头像URL
			auth.PUT("/auth/me/avatar", userHandler.UpdateAvatar)
		}
	}

	utils.GetLogger().Info("路由设置完成", "mode", cfg.Server.Mode, "port", cfg.Server.Port)
	return r
}
