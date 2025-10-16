package bootstrap

import (
	"gin/internal/config"
	"gin/internal/services"
)

// Container 应用容器（简单装配）
type Container struct {
	DB             *services.Database
	Auth           services.AuthServiceInterface
	UserSvc        services.UserServiceInterface
	UserRepo       *services.UserRepository
	Storage        services.StorageClient
	StatsRepo      *services.StatisticsRepository
	HistoryRepo    *services.HistoryRepository
	CumulativeRepo *services.CumulativeStatsRepository
	ChatRepo       *services.ChatRepository
	ArticleRepo    *services.ArticleRepository
	PrivateMsgRepo *services.PrivateMessageRepository
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
	authService := services.NewAuthService(cfg, userRepo, historyRepo)
	userService := services.NewUserService(userRepo)
	storageService, err := services.NewStorageService(cfg)
	if err != nil {
		// 允许存储失败返回 nil，由上层决定是否禁用上传
		storageService = nil
	}

	return &Container{
		DB:             db,
		Auth:           authService,
		UserSvc:        userService,
		UserRepo:       userRepo,
		Storage:        storageService,
		StatsRepo:      statsRepo,
		HistoryRepo:    historyRepo,
		CumulativeRepo: cumulativeRepo,
		ChatRepo:       chatRepo,
		ArticleRepo:    articleRepo,
		PrivateMsgRepo: privateMsgRepo,
	}, nil
}
