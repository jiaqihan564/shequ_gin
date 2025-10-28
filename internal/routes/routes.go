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

	r := gin.New() // 使用 gin.New() 而不是 gin.Default()，手动控制中间件

	// 添加中间件（顺序很重要）
	r.Use(middleware.PanicRecoveryMiddleware())                               // 1. Panic恢复（最先执行）
	r.Use(middleware.RequestIDMiddleware())                                   // 2. 请求ID中间件
	r.Use(middleware.SecurityHeadersMiddleware())                             // 3. 安全响应头
	r.Use(middleware.CORSMiddleware(cfg))                                     // 4. CORS跨域
	r.Use(middleware.RequestSizeLimitMiddleware(10 * 1024 * 1024))            // 5. 请求体大小限制（10MB）
	r.Use(middleware.FastCompressionMiddleware())                             // 6. 响应压缩（速度优先）
	r.Use(middleware.LoggerMiddleware())                                      // 7. 详细日志（包含请求/响应体）
	r.Use(middleware.PerformanceMiddleware(ctn.DB))                           // 8. 性能追踪（内存、CPU、数据库连接池）
	r.Use(middleware.MetricsMiddleware())                                     // 9. 性能监控中间件
	r.Use(middleware.RateLimitMiddleware())                                   // 10. 添加全局限流
	r.Use(middleware.StatisticsMiddleware(ctn.StatsRepo, ctn.CumulativeRepo)) // 11. 统计中间件（自动收集数据）

	// 初始化处理器
	uploadMaxBytes := int64(cfg.Assets.MaxAvatarSizeMB) * 1024 * 1024
	authHandler := handlers.NewAuthHandler(ctn.Auth)
	userHandler := handlers.NewUserHandler(ctn.UserSvc, ctn.HistoryRepo, cfg)
	healthHandler := handlers.NewHealthHandler(ctn.DB)
	uploadHandler := handlers.NewUploadHandler(ctn.Storage, ctn.ResourceStorage, ctn.UserSvc, uploadMaxBytes, cfg.Assets.MaxAvatarHistory, ctn.HistoryRepo)
	statsHandler := handlers.NewStatisticsHandler(ctn.StatsRepo)
	historyHandler := handlers.NewHistoryHandler(ctn.HistoryRepo)
	cumulativeHandler := handlers.NewCumulativeStatsHandler(ctn.CumulativeRepo)
	chatHandler := handlers.NewChatHandler(ctn.ChatRepo, ctn.UserRepo)
	articleHandler := handlers.NewArticleHandler(ctn.ArticleRepo, ctn.CacheSvc)
	privateMsgHandler := handlers.NewPrivateMessageHandler(ctn.PrivateMsgRepo, ctn.UserRepo)
	resourceHandler := handlers.NewResourceHandler(ctn.ResourceRepo, ctn.ResourceCommentRepo, ctn.ResourceStorage)
	chunkUploadHandler := handlers.NewChunkUploadHandler(ctn.UploadMgr)
	codeHandler := handlers.NewCodeHandler(ctn.CodeRepo, ctn.CodeExecutor)

	// 健康检查路由
	r.GET("/health", healthHandler.Check)
	r.GET("/ready", healthHandler.Ready)
	r.GET("/live", healthHandler.Live)

	// 性能监控路由
	r.GET("/metrics", middleware.MetricsHandler)
	r.GET("/metrics/compression", func(c *gin.Context) {
		stats := middleware.GetCompressionStats()
		c.JSON(200, gin.H{
			"code":    200,
			"message": "success",
			"data":    stats,
		})
	})
	r.GET("/metrics/cache", func(c *gin.Context) {
		stats := ctn.CacheSvc.GetAllCacheStats()
		c.JSON(200, gin.H{
			"code":    200,
			"message": "success",
			"data":    stats,
		})
	})
	r.GET("/metrics/performance", func(c *gin.Context) {
		profiler := utils.GetGlobalProfiler()
		report := profiler.GetFullReport()
		c.JSON(200, gin.H{
			"code":    200,
			"message": "success",
			"data": gin.H{
				"uptime": report.Uptime.String(),
				"latency": gin.H{
					"p50": report.Latency.P50.String(),
					"p95": report.Latency.P95.String(),
					"p99": report.Latency.P99.String(),
					"min": report.Latency.Min.String(),
					"max": report.Latency.Max.String(),
				},
				"goroutine": report.Goroutine,
				"memory": gin.H{
					"alloc":      report.Memory.Alloc,
					"totalAlloc": report.Memory.TotalAlloc,
					"sys":        report.Memory.Sys,
					"numGC":      report.Memory.NumGC,
					"heapInuse":  report.Memory.HeapInuse,
				},
			},
		})
	})
	r.GET("/metrics/slow-queries", func(c *gin.Context) {
		detector := utils.GetGlobalSlowQueryDetector()
		stats := detector.GetStats()
		queries := detector.GetSlowQueries()
		c.JSON(200, gin.H{
			"code":    200,
			"message": "success",
			"data": gin.H{
				"stats":   stats,
				"queries": queries,
			},
		})
	})
	r.GET("/metrics/worker-pool", func(c *gin.Context) {
		pool := utils.GetGlobalPool()
		metrics := pool.GetMetrics()
		c.JSON(200, gin.H{
			"code":    200,
			"message": "success",
			"data":    metrics,
		})
	})

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
			auth.POST("/resources/images/upload", uploadHandler.UploadResourceImage)    // 上传资源预览图
			auth.POST("/resources/documents/upload", uploadHandler.UploadDocumentImage) // 上传文档图片

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

			// 历史记录接口（用户查看自己的历史）
			auth.GET("/history/login", historyHandler.GetLoginHistory)
			auth.GET("/history/operations", historyHandler.GetOperationHistory)
			auth.GET("/history/profile-changes", historyHandler.GetProfileChangeHistory)

			// 聊天室接口（所有登录用户可访问）
			auth.POST("/chat/send", chatHandler.SendMessage)             // 发送消息
			auth.GET("/chat/messages", chatHandler.GetMessages)          // 获取消息列表
			auth.GET("/chat/messages/new", chatHandler.GetNewMessages)   // 获取新消息（轮询）
			auth.DELETE("/chat/messages/:id", chatHandler.DeleteMessage) // 删除消息
			auth.GET("/chat/online-count", chatHandler.GetOnlineCount)   // 获取在线用户数
			auth.POST("/chat/offline", chatHandler.UserOffline)          // 用户下线

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
			auth.GET("/resources/:id/download", resourceHandler.DownloadResource)               // 下载资源（返回直接链接）
			auth.GET("/resources/:id/proxy-download", resourceHandler.ProxyDownloadResource)    // 代理下载资源（支持Range和大文件）
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

			// 在线代码执行相关接口
			auth.POST("/code/execute", codeHandler.ExecuteCode)                  // 执行代码
			auth.POST("/code/snippets", codeHandler.CreateSnippet)               // 保存代码片段
			auth.GET("/code/snippets", codeHandler.GetSnippets)                  // 获取代码片段列表
			auth.GET("/code/public", codeHandler.GetPublicSnippets)              // 获取公开代码片段列表
			auth.GET("/code/snippets/:id", codeHandler.GetSnippetByID)           // 获取代码片段详情
			auth.PUT("/code/snippets/:id", codeHandler.UpdateSnippet)            // 更新代码片段
			auth.DELETE("/code/snippets/:id", codeHandler.DeleteSnippet)         // 删除代码片段
			auth.GET("/code/executions", codeHandler.GetExecutions)              // 获取执行记录
			auth.POST("/code/snippets/:id/share", codeHandler.GenerateShareLink) // 生成分享链接
			auth.GET("/code/languages", codeHandler.GetLanguages)                // 获取支持的语言列表
		}

		// 公开访问的代码分享（无需认证）
		api.GET("/code/share/:token", codeHandler.GetSharedSnippet) // 通过分享令牌访问代码

		// 管理员专用路由
		admin := api.Group("/")
		admin.Use(middleware.AuthMiddleware(cfg))
		admin.Use(middleware.AdminMiddleware(cfg))
		{
			// 统计相关接口（仅管理员可访问）
			admin.GET("/statistics/overview", statsHandler.GetOverview)
			admin.GET("/statistics/users", statsHandler.GetUserStatistics)
			admin.GET("/statistics/apis", statsHandler.GetApiStatistics)
			admin.GET("/statistics/ranking", statsHandler.GetEndpointRanking)

			// 地区分布统计
			admin.GET("/location/distribution", historyHandler.GetLocationDistribution)

			// 累计统计接口（全站数据）
			admin.GET("/cumulative-stats", cumulativeHandler.GetCumulativeStats)
			admin.GET("/daily-metrics", cumulativeHandler.GetDailyMetrics)
			admin.GET("/realtime-metrics", cumulativeHandler.GetRealtimeMetrics)
		}
	}

	logger := utils.GetLogger()
	logger.Info("路由设置完成",
		"mode", cfg.Server.Mode,
		"port", cfg.Server.Port,
		"middlewares", []string{
			"1.PanicRecovery",
			"2.RequestID",
			"3.SecurityHeaders",
			"4.CORS",
			"5.RequestSizeLimit",
			"6.Compression",
			"7.Logger",
			"8.Performance",
			"9.Metrics",
			"10.RateLimit",
			"11.Statistics",
		})
	logger.Debug("中间件详情",
		"panicRecovery", "全局panic恢复，防止服务崩溃",
		"securityHeaders", "添加安全响应头（XSS、点击劫持等防护）",
		"requestSizeLimit", "限制请求体大小，防止大文件攻击",
		"logger", "捕获请求/响应体，记录详细头部信息",
		"performance", "追踪内存使用、Goroutine数量、数据库连接池状态",
		"metrics", "性能指标统计",
		"rateLimit", "全局和特定路由限流（LRU优化）")
	return r
}
