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
	userHandler := handlers.NewUserHandler(ctn.UserSvc, cfg)
	healthHandler := handlers.NewHealthHandler(ctn.DB)
	uploadHandler := handlers.NewUploadHandler(ctn.Storage, ctn.UserSvc, uploadMaxBytes, cfg.Assets.MaxAvatarHistory)

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
			// 前端期望的统一接口
			auth.GET("/auth/me", userHandler.GetMe)    // 获取当前用户信息
			auth.PUT("/auth/me", userHandler.UpdateMe) // 更新当前用户信息

			// 文件上传接口
			auth.POST("/upload", uploadHandler.UploadAvatar)

			// 退出登录（JWT无状态，主要用于客户端清除token）
			auth.POST("/auth/logout", authHandler.Logout)

			// 保留的原有接口（向后兼容）
			auth.GET("/user/profile", userHandler.GetProfile)
			auth.PUT("/user/profile", userHandler.UpdateProfile)
			auth.GET("/user/:id", userHandler.GetUserByID)
			auth.POST("/files/upload", uploadHandler.UploadAvatar)
			auth.GET("/user/avatar/history", uploadHandler.ListAvatarHistory)
			auth.GET("/avatar/history", uploadHandler.ListAvatarHistory)

			// 临时兼容旧的头像更新接口
			auth.PUT("/auth/me/avatar", userHandler.UpdateMe)
		}
	}

	utils.GetLogger().Info("路由设置完成", "mode", cfg.Server.Mode, "port", cfg.Server.Port)
	return r
}
