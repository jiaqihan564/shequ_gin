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
	articleHandler := handlers.NewArticleHandler(ctn.ArticleRepo)
	privateMsgHandler := handlers.NewPrivateMessageHandler(ctn.PrivateMsgRepo, ctn.UserRepo)
	resourceHandler := handlers.NewResourceHandler(ctn.ResourceRepo, ctn.ResourceCommentRepo)
	chunkUploadHandler := handlers.NewChunkUploadHandler(ctn.UploadMgr)

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

			// 文章相关接口
			auth.POST("/articles", articleHandler.CreateArticle)              // 创建文章
			auth.GET("/articles/:id", articleHandler.GetArticleDetail)        // 获取文章详情
			auth.PUT("/articles/:id", articleHandler.UpdateArticle)           // 更新文章
			auth.DELETE("/articles/:id", articleHandler.DeleteArticle)        // 删除文章
			auth.POST("/articles/:id/like", articleHandler.ToggleArticleLike) // 点赞/取消点赞
			auth.POST("/articles/:id/comments", articleHandler.CreateComment) // 发表评论
			auth.GET("/articles/:id/comments", articleHandler.GetComments)    // 获取评论
			auth.POST("/comments/:id/like", articleHandler.ToggleCommentLike) // 评论点赞
			auth.DELETE("/comments/:id", articleHandler.DeleteComment)        // 删除评论
			auth.POST("/articles/report", articleHandler.CreateReport)        // 举报文章/评论
			auth.GET("/articles", articleHandler.GetArticleList)              // 获取文章列表
			auth.GET("/articles/categories", articleHandler.GetCategories)    // 获取分类列表
			auth.GET("/articles/tags", articleHandler.GetTags)                // 获取标签列表

			// 私信相关接口
			auth.GET("/conversations", privateMsgHandler.GetConversations)                 // 获取会话列表
			auth.GET("/conversations/:id/messages", privateMsgHandler.GetMessages)         // 获取会话消息
			auth.POST("/messages/send", privateMsgHandler.SendMessage)                     // 发送消息
			auth.GET("/conversations/unread-count", privateMsgHandler.GetUnreadCount)      // 获取未读数
			auth.POST("/conversations/start/:userId", privateMsgHandler.StartConversation) // 开始会话

			// 资源相关接口
			auth.POST("/resources", resourceHandler.CreateResource)                             // 创建资源
			auth.GET("/resources", resourceHandler.GetResourceList)                             // 获取资源列表
			auth.GET("/resources/:id", resourceHandler.GetResourceDetail)                       // 获取资源详情
			auth.DELETE("/resources/:id", resourceHandler.DeleteResource)                       // 删除资源
			auth.POST("/resources/:id/like", resourceHandler.ToggleResourceLike)                // 点赞资源
			auth.GET("/resources/:id/download", resourceHandler.DownloadResource)               // 下载资源
			auth.GET("/resource-categories", resourceHandler.GetCategories)                     // 获取资源分类
			auth.POST("/resources/:id/comments", resourceHandler.CreateResourceComment)         // 发表资源评论
			auth.GET("/resources/:id/comments", resourceHandler.GetResourceComments)            // 获取资源评论
			auth.POST("/resource-comments/:id/like", resourceHandler.ToggleResourceCommentLike) // 资源评论点赞

			// 分片上传接口
			auth.POST("/upload/init", chunkUploadHandler.InitUpload)                  // 初始化上传
			auth.POST("/upload/chunk", chunkUploadHandler.UploadChunk)                // 上传分片
			auth.POST("/upload/merge", chunkUploadHandler.MergeChunks)                // 合并分片
			auth.GET("/upload/status/:upload_id", chunkUploadHandler.GetUploadStatus) // 查询进度
			auth.POST("/upload/cancel/:upload_id", chunkUploadHandler.CancelUpload)   // 取消上传
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
