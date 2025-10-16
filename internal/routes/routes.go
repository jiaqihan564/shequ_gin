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
	r.Use(middleware.RequestIDMiddleware())                                   // 请求ID中间件
	r.Use(middleware.CORSMiddleware(cfg))                                     // CORS跨域
	r.Use(middleware.LoggerMiddleware())                                      // 详细日志（包含请求/响应体）
	r.Use(middleware.PerformanceMiddleware(ctn.DB))                           // 性能追踪（内存、CPU、数据库连接池）
	r.Use(middleware.MetricsMiddleware())                                     // 性能监控中间件
	r.Use(middleware.RateLimitMiddleware())                                   // 添加全局限流
	r.Use(middleware.StatisticsMiddleware(ctn.StatsRepo, ctn.CumulativeRepo)) // 统计中间件（自动收集数据）

	// 初始化处理器
	uploadMaxBytes := int64(cfg.Assets.MaxAvatarSizeMB) * 1024 * 1024
	authHandler := handlers.NewAuthHandler(ctn.Auth)
	userHandler := handlers.NewUserHandler(ctn.UserSvc, ctn.HistoryRepo, cfg)
	healthHandler := handlers.NewHealthHandler(ctn.DB)
	uploadHandler := handlers.NewUploadHandler(ctn.Storage, ctn.UserSvc, uploadMaxBytes, cfg.Assets.MaxAvatarHistory)
	statsHandler := handlers.NewStatisticsHandler(ctn.StatsRepo)
	historyHandler := handlers.NewHistoryHandler(ctn.HistoryRepo)
	cumulativeHandler := handlers.NewCumulativeStatsHandler(ctn.CumulativeRepo)
	chatHandler := handlers.NewChatHandler(ctn.ChatRepo, ctn.UserRepo)

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
		api.POST("/auth/forgot-password", authHandler.ForgotPassword) // 忘记密码
		api.POST("/auth/reset-password", authHandler.ResetPassword)   // 重置密码

		// 需要认证的路由
		auth := api.Group("/")
		auth.Use(middleware.AuthMiddleware(cfg))
		{
			// 前端期望的统一接口
			auth.GET("/auth/me", userHandler.GetMe)                        // 获取当前用户信息
			auth.PUT("/auth/me", userHandler.UpdateMe)                     // 更新当前用户信息
			auth.POST("/auth/change-password", authHandler.ChangePassword) // 修改密码

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

			// 统计相关接口（所有登录用户可访问）
			auth.GET("/statistics/overview", statsHandler.GetOverview)
			auth.GET("/statistics/users", statsHandler.GetUserStatistics)
			auth.GET("/statistics/apis", statsHandler.GetApiStatistics)
			auth.GET("/statistics/ranking", statsHandler.GetEndpointRanking)

			// 历史记录接口（用户查看自己的历史）
			auth.GET("/history/login", historyHandler.GetLoginHistory)
			auth.GET("/history/operations", historyHandler.GetOperationHistory)
			auth.GET("/history/profile-changes", historyHandler.GetProfileChangeHistory)

			// 地区分布统计
			auth.GET("/location/distribution", historyHandler.GetLocationDistribution)

			// 累计统计接口（全站数据）
			auth.GET("/cumulative-stats", cumulativeHandler.GetCumulativeStats)
			auth.GET("/daily-metrics", cumulativeHandler.GetDailyMetrics)
			auth.GET("/realtime-metrics", cumulativeHandler.GetRealtimeMetrics)

			// 聊天室接口（所有登录用户可访问）
			auth.POST("/chat/send", chatHandler.SendMessage)             // 发送消息
			auth.GET("/chat/messages", chatHandler.GetMessages)          // 获取消息列表
			auth.GET("/chat/messages/new", chatHandler.GetNewMessages)   // 获取新消息（轮询）
			auth.DELETE("/chat/messages/:id", chatHandler.DeleteMessage) // 删除消息
			auth.GET("/chat/online-count", chatHandler.GetOnlineCount)   // 获取在线用户数
		}
	}

	logger := utils.GetLogger()
	logger.Info("路由设置完成",
		"mode", cfg.Server.Mode,
		"port", cfg.Server.Port,
		"middlewares", []string{
			"RequestID",
			"CORS",
			"Logger (detailed)",
			"Performance (memory/CPU/DB)",
			"Metrics",
			"RateLimit",
		})
	logger.Debug("中间件详情",
		"loggerMiddleware", "捕获请求/响应体，记录详细头部信息",
		"performanceMiddleware", "追踪内存使用、Goroutine数量、数据库连接池状态",
		"metricsMiddleware", "性能指标统计",
		"rateLimitMiddleware", "全局和特定路由限流")
	return r
}
