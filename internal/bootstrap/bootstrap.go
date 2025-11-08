package bootstrap

import (
	"fmt"
	"time"

	"gin/internal/config"
	"gin/internal/services"
	"gin/internal/utils"
)

// Container 应用容器（7桶架构）
type Container struct {
	DB                  *services.Database
	Auth                services.AuthServiceInterface
	UserSvc             services.UserServiceInterface
	UserRepo            *services.UserRepository
	MultiBucket         *services.MultiBucketStorage   // 多桶存储服务（7桶架构）
	StatsRepo           *services.StatisticsRepository
	HistoryRepo         *services.HistoryRepository
	CumulativeRepo      *services.CumulativeStatsRepository
	ChatRepo            *services.ChatRepository
	ArticleRepo         *services.ArticleRepository
	PrivateMsgRepo      *services.PrivateMessageRepository
	ResourceRepo        *services.ResourceRepository
	ResourceCommentRepo *services.ResourceCommentRepository
	ResourceImageSvc    *services.ResourceImageService // 资源图片服务
	UploadMgr           *services.UploadManager
	CacheSvc            *services.CacheService // 缓存服务
	CodeRepo            services.CodeRepository
	CodeExecutor        services.CodeExecutor
	Config              *config.Config // 配置
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

	// 初始化多桶存储服务（7桶架构）
	multiBucketStorage, err := services.NewMultiBucketStorage(cfg)
	if err != nil {
		logger := utils.GetLogger()
		logger.Error("多桶存储服务初始化失败", "error", err.Error())
		return nil, fmt.Errorf("多桶存储服务初始化失败: %w", err)
	}

	uploadMgr := services.NewUploadManager(db, multiBucketStorage, cfg)
	resourceImageSvc := services.NewResourceImageService(multiBucketStorage)

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
		MultiBucket:         multiBucketStorage,
		StatsRepo:           statsRepo,
		HistoryRepo:         historyRepo,
		CumulativeRepo:      cumulativeRepo,
		ChatRepo:            chatRepo,
		ArticleRepo:         articleRepo,
		PrivateMsgRepo:      privateMsgRepo,
		ResourceRepo:        resourceRepo,
		ResourceCommentRepo: resourceCommentRepo,
		ResourceImageSvc:    resourceImageSvc,
		UploadMgr:           uploadMgr,
		CacheSvc:            cacheService,
		CodeRepo:            codeRepo,
		CodeExecutor:        codeExecutor,
		Config:              cfg,
	}, nil
}
