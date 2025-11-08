package bootstrap

import (
	"gin/internal/config"
	"gin/internal/services"
	"gin/internal/utils"
	"time"
)

// Container 应用容器（简单装配）
type Container struct {
	DB                  *services.Database
	Auth                services.AuthServiceInterface
	UserSvc             services.UserServiceInterface
	UserRepo            *services.UserRepository
	Storage             services.StorageClient
	MultiBucket         *services.MultiBucketStorage // 多桶存储服务（7桶架构）
	StatsRepo           *services.StatisticsRepository
	HistoryRepo         *services.HistoryRepository
	CumulativeRepo      *services.CumulativeStatsRepository
	ChatRepo            *services.ChatRepository
	ArticleRepo         *services.ArticleRepository
	PrivateMsgRepo      *services.PrivateMessageRepository
	ResourceRepo        *services.ResourceRepository
	ResourceCommentRepo *services.ResourceCommentRepository
	ResourceStorage     *services.ResourceStorageService // 废弃，使用MultiBucket替代
	ResourceImageSvc    *services.ResourceImageService   // 资源图片服务（7桶架构）
	UploadMgr           *services.UploadManager
	CacheSvc            *services.CacheService // 缓存服务
	CodeRepo            services.CodeRepository
	CodeExecutor        services.CodeExecutor
	Config              *config.Config         // 配置
}

// New 构建容器
func New(cfg *config.Config, db *services.Database) (*Container, error) {
	// 初始化管理员检查器（性能优化）
	utils.InitAdminChecker(cfg)

	userRepo := services.NewUserRepository(db)
	statsRepo := services.NewStatisticsRepository(db, cfg)
	historyRepo := services.NewHistoryRepository(db, cfg)
	cumulativeRepo := services.NewCumulativeStatsRepository(db, cfg)
	chatRepo := services.NewChatRepository(db, cfg)
	articleRepo := services.NewArticleRepository(db, cfg)
	privateMsgRepo := services.NewPrivateMessageRepository(db)
	resourceRepo := services.NewResourceRepository(db, cfg)
	resourceCommentRepo := services.NewResourceCommentRepository(db, cfg)
	authService := services.NewAuthService(cfg, userRepo, historyRepo)
	userService := services.NewUserService(userRepo)
	storageService, err := services.NewStorageService(cfg)
	if err != nil {
		// 允许存储失败返回 nil，由上层决定是否禁用上传
		storageService = nil
	}

	// 初始化资源存储服务（独立桶，废弃）
	resourceStorage, err := services.NewResourceStorageService(cfg)
	if err != nil {
		logger := utils.GetLogger()
		logger.Warn("资源存储服务初始化失败", "error", err.Error())
		resourceStorage = nil
	}

	// 初始化多桶存储服务（7桶架构）
	multiBucketStorage, err := services.NewMultiBucketStorage(cfg)
	if err != nil {
		logger := utils.GetLogger()
		logger.Error("多桶存储服务初始化失败", "error", err.Error())
		multiBucketStorage = nil
	}

	uploadMgr := services.NewUploadManager(db, storageService, cfg)
	
	// 关联多桶存储到UploadManager
	if multiBucketStorage != nil {
		uploadMgr.SetMultiBucketStorage(multiBucketStorage)
	}

	// 初始化资源图片服务（7桶架构）
	var resourceImageSvc *services.ResourceImageService
	if multiBucketStorage != nil {
		resourceImageSvc = services.NewResourceImageService(multiBucketStorage)
	}

	// 初始化缓存服务
	cacheService := services.NewCacheService(articleRepo, cfg)

	// 初始化代码仓库和执行器
	codeRepo := services.NewCodeRepository(db)
	codeExecutor := services.NewPistonCodeExecutor(
		cfg.CodeExecutor.PistonAPIURL,
		time.Duration(cfg.CodeExecutor.Timeout)*time.Second,
		cfg.HTTPClient.MaxIdleConns,
		cfg.HTTPClient.MaxIdleConnsPerHost,
		cfg.HTTPClient.IdleConnTimeout,
	)

	return &Container{
		DB:                  db,
		Auth:                authService,
		UserSvc:             userService,
		UserRepo:            userRepo,
		Storage:             storageService,
		MultiBucket:         multiBucketStorage,  // 多桶存储服务
		StatsRepo:           statsRepo,
		HistoryRepo:         historyRepo,
		CumulativeRepo:      cumulativeRepo,
		ChatRepo:            chatRepo,
		ArticleRepo:         articleRepo,
		PrivateMsgRepo:      privateMsgRepo,
		ResourceRepo:        resourceRepo,
		ResourceCommentRepo: resourceCommentRepo,
		ResourceStorage:     resourceStorage,     // 保留向后兼容
		ResourceImageSvc:    resourceImageSvc,    // 资源图片服务
		UploadMgr:           uploadMgr,
		CacheSvc:            cacheService,
		CodeRepo:            codeRepo,
		CodeExecutor:        codeExecutor,
		Config:              cfg,
	}, nil
}
