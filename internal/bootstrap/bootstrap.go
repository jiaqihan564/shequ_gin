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
	StatsRepo           *services.StatisticsRepository
	HistoryRepo         *services.HistoryRepository
	CumulativeRepo      *services.CumulativeStatsRepository
	ChatRepo            *services.ChatRepository
	ArticleRepo         *services.ArticleRepository
	PrivateMsgRepo      *services.PrivateMessageRepository
	ResourceRepo        *services.ResourceRepository
	ResourceCommentRepo *services.ResourceCommentRepository
	ResourceStorage     *services.ResourceStorageService
	UploadMgr           *services.UploadManager
	CacheSvc            *services.CacheService // 缓存服务
	CodeRepo            services.CodeRepository
	CodeExecutor        services.CodeExecutor
}

// New 构建容器
func New(cfg *config.Config, db *services.Database) (*Container, error) {
	userRepo := services.NewUserRepository(db)
	statsRepo := services.NewStatisticsRepository(db)
	historyRepo := services.NewHistoryRepository(db)
	cumulativeRepo := services.NewCumulativeStatsRepository(db)
	chatRepo := services.NewChatRepository(db)
	articleRepo := services.NewArticleRepository(db)
	privateMsgRepo := services.NewPrivateMessageRepository(db)
	resourceRepo := services.NewResourceRepository(db)
	resourceCommentRepo := services.NewResourceCommentRepository(db)
	authService := services.NewAuthService(cfg, userRepo, historyRepo)
	userService := services.NewUserService(userRepo)
	storageService, err := services.NewStorageService(cfg)
	if err != nil {
		// 允许存储失败返回 nil，由上层决定是否禁用上传
		storageService = nil
	}

	// 初始化资源存储服务（独立桶）
	resourceStorage, err := services.NewResourceStorageService(cfg)
	if err != nil {
		logger := utils.GetLogger()
		logger.Warn("资源存储服务初始化失败", "error", err.Error())
		resourceStorage = nil
	}

	uploadMgr := services.NewUploadManager(db, storageService)

	// 初始化缓存服务
	cacheService := services.NewCacheService(articleRepo)

	// 初始化代码仓库和执行器
	codeRepo := services.NewCodeRepository(db)
	codeExecutor := services.NewPistonCodeExecutor(
		cfg.CodeExecutor.PistonAPIURL,
		time.Duration(cfg.CodeExecutor.Timeout)*time.Second,
	)

	return &Container{
		DB:                  db,
		Auth:                authService,
		UserSvc:             userService,
		UserRepo:            userRepo,
		Storage:             storageService,
		StatsRepo:           statsRepo,
		HistoryRepo:         historyRepo,
		CumulativeRepo:      cumulativeRepo,
		ChatRepo:            chatRepo,
		ArticleRepo:         articleRepo,
		PrivateMsgRepo:      privateMsgRepo,
		ResourceRepo:        resourceRepo,
		ResourceCommentRepo: resourceCommentRepo,
		ResourceStorage:     resourceStorage,
		UploadMgr:           uploadMgr,
		CacheSvc:            cacheService,
		CodeRepo:            codeRepo,
		CodeExecutor:        codeExecutor,
	}, nil
}
