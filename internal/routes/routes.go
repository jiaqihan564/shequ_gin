package routes

import (
	"gin/internal/config"
	"gin/internal/handlers"
	"gin/internal/middleware"
	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置路由
func SetupRoutes(cfg *config.Config, db *services.Database) *gin.Engine {
	// 初始化日志系统
	if err := utils.InitLogger(&cfg.Log); err != nil {
		utils.GetLogger().Error("初始化日志系统失败", "error", err.Error())
	}

	// 初始化响应处理器
	utils.InitResponseHandler()

	// 初始化限流器
	middleware.InitRateLimiter(cfg)

	// 初始化性能监控
	middleware.InitMetrics()

	// 设置Gin模式
	gin.SetMode(cfg.Server.Mode)

	r := gin.Default()

	// 添加中间件
	r.Use(middleware.RequestIDMiddleware()) // 请求ID中间件
	r.Use(middleware.CORSMiddleware(cfg))
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.MetricsMiddleware())   // 性能监控中间件
	r.Use(middleware.RateLimitMiddleware()) // 添加全局限流

	// 初始化数据访问层
	userRepo := services.NewUserRepository(db)

	// 初始化服务
	authService := services.NewAuthService(cfg, userRepo)
	userService := services.NewUserService(userRepo)

	// 初始化处理器
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	healthHandler := handlers.NewHealthHandler(db)

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
			auth.GET("/user/:id", userHandler.GetUserByID)
		}
	}

	utils.GetLogger().Info("路由设置完成", "mode", cfg.Server.Mode, "port", cfg.Server.Port)
	return r
}
