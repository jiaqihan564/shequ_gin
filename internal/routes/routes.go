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
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.CORSMiddleware(cfg))
	r.Use(middleware.LoggingMiddleware()) // 合并的日志和监控中间件
	r.Use(middleware.RateLimitMiddleware())

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
		// 用户认证相关路由（使用认证限流）
		authLimit := middleware.AuthRateLimitMiddleware()
		api.POST("/auth/register", authLimit, authHandler.Register)
		api.POST("/auth/login", authLimit, authHandler.Login)

		// 需要认证的路由
		auth := api.Group("/")
		auth.Use(middleware.AuthMiddleware(cfg))
		{
			// 用户信息管理
			auth.GET("/auth/me", userHandler.GetMe)
			auth.PUT("/auth/me", userHandler.UpdateMe)
			auth.POST("/auth/logout", authHandler.Logout)

			// 文件上传
			auth.POST("/upload", uploadHandler.UploadAvatar)
			auth.GET("/avatar/history", uploadHandler.ListAvatarHistory)
		}
	}

	utils.GetLogger().Info("路由设置完成", "mode", cfg.Server.Mode, "port", cfg.Server.Port)
	return r
}
