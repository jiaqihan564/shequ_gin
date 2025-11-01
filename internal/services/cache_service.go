package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gin/internal/config"
	"gin/internal/models"
	"gin/internal/utils"
)

// CacheService 缓存服务
// 为热点数据提供内存缓存，减少数据库查询
type CacheService struct {
	cache       *utils.MemoryCache
	articleRepo *ArticleRepository
	logger      utils.Logger
	config      *config.CacheConfig

	// 分组缓存（不同类型数据使用不同的LRU缓存）
	articleCache *utils.LRUCache // 文章缓存
	userCache    *utils.LRUCache // 用户缓存
	listCache    *utils.LRUCache // 列表缓存
}

// NewCacheService 创建缓存服务
func NewCacheService(articleRepo *ArticleRepository, cfg *config.Config) *CacheService {
	logger := utils.GetLogger()

	service := &CacheService{
		cache:       utils.GetCache(),
		articleRepo: articleRepo,
		logger:      logger,
		config:      &cfg.Cache,

		// 创建分组缓存（从配置读取）
		articleCache: utils.NewLRUCache(utils.LRUCacheConfig{
			Capacity:   cfg.Cache.Article.Capacity,
			MaxMemory:  int64(cfg.Cache.Article.MaxMemoryMB) * 1024 * 1024,
			DefaultTTL: time.Duration(cfg.Cache.Article.TTLMinutes) * time.Minute,
		}),
		userCache: utils.NewLRUCache(utils.LRUCacheConfig{
			Capacity:   cfg.Cache.User.Capacity,
			MaxMemory:  int64(cfg.Cache.User.MaxMemoryMB) * 1024 * 1024,
			DefaultTTL: time.Duration(cfg.Cache.User.TTLMinutes) * time.Minute,
		}),
		listCache: utils.NewLRUCache(utils.LRUCacheConfig{
			Capacity:   cfg.Cache.List.Capacity,
			MaxMemory:  int64(cfg.Cache.List.MaxMemoryMB) * 1024 * 1024,
			DefaultTTL: time.Duration(cfg.Cache.List.TTLMinutes) * time.Minute,
		}),
	}

	logger.Info("缓存服务已初始化",
		"articleCacheCapacity", cfg.Cache.Article.Capacity,
		"userCacheCapacity", cfg.Cache.User.Capacity,
		"listCacheCapacity", cfg.Cache.List.Capacity)

	// 启动缓存预热（异步）
	go service.warmupCache()

	return service
}

// Cache Keys
const (
	cacheKeyArticleCategories = "article:categories:all"
	cacheKeyArticleTags       = "article:tags:all"
	cacheKeyArticlePrefix     = "article:detail:"
	cacheKeyOnlineCount       = "chat:online:count"
)

// getCategoriesTTL 获取分类缓存TTL
func (s *CacheService) getCategoriesTTL() time.Duration {
	return time.Duration(s.config.CategoriesTTLMinutes) * time.Minute
}

// getTagsTTL 获取标签缓存TTL
func (s *CacheService) getTagsTTL() time.Duration {
	return time.Duration(s.config.TagsTTLMinutes) * time.Minute
}

// getArticleDetailTTL 获取文章详情缓存TTL
func (s *CacheService) getArticleDetailTTL() time.Duration {
	return time.Duration(s.config.ArticleDetailTTLMinutes) * time.Minute
}

// getOnlineCountTTL 获取在线人数缓存TTL
func (s *CacheService) getOnlineCountTTL() time.Duration {
	return time.Duration(s.config.OnlineCountTTLSeconds) * time.Second
}

// =============================================================================
// 文章分类缓存
// =============================================================================

// GetArticleCategories 获取文章分类（带缓存）
func (s *CacheService) GetArticleCategories(ctx context.Context) ([]models.ArticleCategory, error) {
	// 尝试从缓存获取
	if cached, ok := s.cache.Get(cacheKeyArticleCategories); ok {
		if categories, ok := cached.([]models.ArticleCategory); ok {
			return categories, nil
		}
	}

	// 缓存未命中，从数据库获取
	categories, err := s.articleRepo.GetAllCategories(ctx)
	if err != nil {
		return nil, err
	}

	// 写入缓存
	ttl := s.getCategoriesTTL()
	s.cache.SetWithTTL(cacheKeyArticleCategories, categories, ttl)
	s.logger.Info("分类数据已缓存", "count", len(categories), "ttl", ttl)

	return categories, nil
}

// InvalidateArticleCategories 使分类缓存失效
func (s *CacheService) InvalidateArticleCategories() {
	s.cache.Delete(cacheKeyArticleCategories)
	s.logger.Info("分类缓存已失效")
}

// =============================================================================
// 文章标签缓存
// =============================================================================

// GetArticleTags 获取文章标签（带缓存）
func (s *CacheService) GetArticleTags(ctx context.Context) ([]models.ArticleTag, error) {
	// 尝试从缓存获取
	if cached, ok := s.cache.Get(cacheKeyArticleTags); ok {
		if tags, ok := cached.([]models.ArticleTag); ok {
			return tags, nil
		}
	}

	// 缓存未命中，从数据库获取
	tags, err := s.articleRepo.GetAllTags(ctx)
	if err != nil {
		return nil, err
	}

	// 写入缓存
	ttl := s.getTagsTTL()
	s.cache.SetWithTTL(cacheKeyArticleTags, tags, ttl)
	s.logger.Info("标签数据已缓存", "count", len(tags), "ttl", ttl)

	return tags, nil
}

// InvalidateArticleTags 使标签缓存失效
func (s *CacheService) InvalidateArticleTags() {
	s.cache.Delete(cacheKeyArticleTags)
	s.logger.Info("标签缓存已失效")
}

// =============================================================================
// 文章详情缓存
// =============================================================================

// GetArticleDetail 获取文章详情（带缓存）
func (s *CacheService) GetArticleDetail(ctx context.Context, articleID uint, userID uint) (*models.ArticleDetailResponse, error) {
	cacheKey := fmt.Sprintf("%s%d:user%d", cacheKeyArticlePrefix, articleID, userID)

	// 尝试从缓存获取
	if cached, ok := s.cache.Get(cacheKey); ok {
		// 尝试类型断言
		if article, ok := cached.(*models.ArticleDetailResponse); ok {
			return article, nil
		}

		// 如果类型断言失败，尝试JSON反序列化
		if jsonData, ok := cached.(string); ok {
			var article models.ArticleDetailResponse
			if err := json.Unmarshal([]byte(jsonData), &article); err == nil {
				return &article, nil
			}
		}
	}

	// 缓存未命中，从数据库获取

	// 使用优化版本的查询（JOIN减少查询次数）
	article, err := s.articleRepo.GetArticleByID(ctx, articleID, userID)
	if err != nil {
		return nil, err
	}

	// 写入缓存（使用较短的TTL）
	s.cache.SetWithTTL(cacheKey, article, s.getArticleDetailTTL())
	return article, nil
}

// InvalidateArticleDetail 使文章详情缓存失效
func (s *CacheService) InvalidateArticleDetail(articleID uint) {
	// 文章详情缓存包含用户ID，需要清除所有相关缓存
	// 简单方案：使用前缀匹配删除（内存缓存不支持模式匹配，这里手动处理）
	// 更好的方案是使用Redis的SCAN + DEL

	// 由于TTL较短，简单记录日志即可
	// 缓存会自动过期
	s.logger.Info("文章详情缓存将在TTL后自动失效", "articleID", articleID, "ttl", s.getArticleDetailTTL())
}

// =============================================================================
// 在线用户数缓存
// =============================================================================

// SetOnlineCount 设置在线用户数缓存
func (s *CacheService) SetOnlineCount(count int) {
	s.cache.SetWithTTL(cacheKeyOnlineCount, count, s.getOnlineCountTTL())
}

// GetOnlineCount 获取在线用户数（从缓存）
func (s *CacheService) GetOnlineCount() (int, bool) {
	if cached, ok := s.cache.Get(cacheKeyOnlineCount); ok {
		if count, ok := cached.(int); ok {
			return count, true
		}
	}
	return 0, false
}

// =============================================================================
// 缓存统计
// =============================================================================

// GetCacheStats 获取缓存统计信息
func (s *CacheService) GetCacheStats() map[string]interface{} {
	return map[string]interface{}{
		"size": s.cache.Size(),
		"keys": []string{
			cacheKeyArticleCategories,
			cacheKeyArticleTags,
			cacheKeyOnlineCount,
			fmt.Sprintf("%s*", cacheKeyArticlePrefix),
		},
	}
}

// ClearAllCache 清空所有缓存（谨慎使用）
func (s *CacheService) ClearAllCache() {
	s.cache.Clear()
	s.articleCache.Clear()
	s.userCache.Clear()
	s.listCache.Clear()
	s.logger.Warn("所有缓存已清空")
}

// warmupCache 缓存预热
func (s *CacheService) warmupCache() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.config.WarmupTimeout)*time.Second)
	defer cancel()

	s.logger.Info("开始缓存预热...")

	// 预热分类和标签（最常访问的数据）
	if categories, err := s.articleRepo.GetAllCategories(ctx); err == nil {
		s.cache.SetWithTTL(cacheKeyArticleCategories, categories, s.getCategoriesTTL())
		s.logger.Info("分类数据已预热", "count", len(categories))
	}

	if tags, err := s.articleRepo.GetAllTags(ctx); err == nil {
		s.cache.SetWithTTL(cacheKeyArticleTags, tags, s.getTagsTTL())
		s.logger.Info("标签数据已预热", "count", len(tags))
	}

	s.logger.Info("缓存预热完成")
}

// GetAllCacheStats 获取所有缓存统计
func (s *CacheService) GetAllCacheStats() map[string]interface{} {
	return map[string]interface{}{
		"global":  s.cache.Stats(),
		"article": s.articleCache.Stats(),
		"user":    s.userCache.Stats(),
		"list":    s.listCache.Stats(),
	}
}
