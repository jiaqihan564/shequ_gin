package routes

import (
	"gin/internal/config"
	"gin/internal/handlers"
	"gin/internal/middleware"
	"gin/internal/services"

	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置路由
func SetupRoutes(cfg *config.Config, db *services.Database) *gin.Engine {
	// 设置Gin模式
	gin.SetMode(cfg.Server.Mode)

	r := gin.Default()

	// 添加中间件
	r.Use(middleware.CORSMiddleware(cfg))
	r.Use(middleware.LoggerMiddleware())

	// 初始化数据访问层
	userRepo := services.NewUserRepository(db)

	// 初始化服务
	authService := services.NewAuthService(cfg, userRepo)
	userService := services.NewUserService(userRepo)

	// 初始化处理器
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	healthHandler := handlers.NewHealthHandler()

	// 健康检查路由
	r.GET("/health", healthHandler.Check)

	// API路由组
	api := r.Group("/api/v1")
	{
		// 用户认证相关路由
		api.POST("/register", authHandler.Register)
		api.POST("/login", authHandler.Login)

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

	return r
}
