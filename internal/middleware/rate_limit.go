package middleware

import (
	"container/list"
	"sync"
	"time"

	"gin/internal/config"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow(key string) bool
	Reset(key string)
}

// TokenBucket 令牌桶限流器
type TokenBucket struct {
	capacity   int           // 桶容量
	tokens     int           // 当前令牌数
	refillRate time.Duration // 令牌补充速率
	lastRefill time.Time     // 上次补充时间
	mutex      sync.RWMutex  // 读写锁
}

// NewTokenBucket 创建令牌桶
func NewTokenBucket(capacity int, refillRate time.Duration) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow 检查是否允许请求
func (tb *TokenBucket) Allow(key string) bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	now := time.Now()
	// 计算需要补充的令牌数
	elapsed := now.Sub(tb.lastRefill)
	tokensToAdd := int(elapsed / tb.refillRate)

	if tokensToAdd > 0 {
		tb.tokens += tokensToAdd
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
		tb.lastRefill = now
	}

	// 检查是否有可用令牌
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}
	return false
}

// Reset 重置令牌桶
func (tb *TokenBucket) Reset(key string) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	tb.tokens = tb.capacity
	tb.lastRefill = time.Now()
}

// LRURateLimiter 基于LRU的限流器（优化版）
// 解决原版本的内存泄漏问题，使用LRU缓存自动淘汰旧条目
type LRURateLimiter struct {
	capacity   int
	refillRate time.Duration
	maxSize    int // LRU缓存最大大小

	limiters  map[string]*lruEntry
	lruList   *list.List
	mutex     sync.RWMutex
	stopClean chan struct{}
}

// lruEntry LRU条目
type lruEntry struct {
	key        string
	limiter    *TokenBucket
	lastAccess time.Time
	element    *list.Element // 在LRU链表中的位置
}

// NewLRURateLimiter 创建LRU限流器
func NewLRURateLimiter(capacity int, refillRate time.Duration, maxSize int, cleanupMinutes int, expireMinutes int) *LRURateLimiter {
	rl := &LRURateLimiter{
		capacity:   capacity,
		refillRate: refillRate,
		maxSize:    maxSize,
		limiters:   make(map[string]*lruEntry),
		lruList:    list.New(),
		stopClean:  make(chan struct{}),
	}

	// 启动定期清理（使用配置的清理间隔）
	go func() {
		ticker := time.NewTicker(time.Duration(cleanupMinutes) * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				rl.cleanup(expireMinutes)
			case <-rl.stopClean:
				return
			}
		}
	}()

	return rl
}

// GetLimiter 获取或创建限流器（LRU优化）
func (rl *LRURateLimiter) GetLimiter(key string) *TokenBucket {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// 如果存在，移到LRU链表前端
	if entry, exists := rl.limiters[key]; exists {
		entry.lastAccess = time.Now()
		rl.lruList.MoveToFront(entry.element)
		return entry.limiter
	}

	// 检查是否达到最大容量，淘汰最久未使用的
	if len(rl.limiters) >= rl.maxSize {
		rl.evictOldest()
	}

	// 创建新限流器
	limiter := NewTokenBucket(rl.capacity, rl.refillRate)
	entry := &lruEntry{
		key:        key,
		limiter:    limiter,
		lastAccess: time.Now(),
	}
	entry.element = rl.lruList.PushFront(entry)
	rl.limiters[key] = entry

	return limiter
}

// evictOldest 淘汰最久未使用的条目
func (rl *LRURateLimiter) evictOldest() {
	elem := rl.lruList.Back()
	if elem != nil {
		entry := elem.Value.(*lruEntry)
		delete(rl.limiters, entry.key)
		rl.lruList.Remove(elem)

		utils.GetLogger().Debug("LRU淘汰限流器条目",
			"key", entry.key,
			"lastAccess", entry.lastAccess)
	}
}

// cleanup 清理过期条目
func (rl *LRURateLimiter) cleanup(expireMinutes int) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	expireTime := time.Duration(expireMinutes) * time.Minute
	removed := 0

	// 从链表尾部开始清理（最久未使用的）
	for elem := rl.lruList.Back(); elem != nil; {
		entry := elem.Value.(*lruEntry)
		if now.Sub(entry.lastAccess) > expireTime {
			prev := elem.Prev()
			delete(rl.limiters, entry.key)
			rl.lruList.Remove(elem)
			removed++
			elem = prev
		} else {
			// 链表是按访问时间排序的，遇到未过期的就停止
			break
		}
	}

	if removed > 0 {
		utils.GetLogger().Info("清理过期限流器条目（LRU）",
			"removed", removed,
			"remaining", len(rl.limiters))
	}
}

// Allow 检查是否允许请求
func (rl *LRURateLimiter) Allow(key string) bool {
	limiter := rl.GetLimiter(key)
	return limiter.Allow(key)
}

// Stop 停止清理goroutine
func (rl *LRURateLimiter) Stop() {
	close(rl.stopClean)
}

// Size 获取当前缓存大小
func (rl *LRURateLimiter) Size() int {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()
	return len(rl.limiters)
}

// 全局限流器（使用LRU优化）
var (
	globalIPRateLimiter       *LRURateLimiter
	globalLoginRateLimiter    *LRURateLimiter
	globalRegisterRateLimiter *LRURateLimiter
	rateLimiterOnce           sync.Once
)

// InitRateLimiter 初始化所有限流器（应在应用启动时调用一次）
func InitRateLimiter(cfg *config.Config) {
	rateLimiterOnce.Do(func() {
		logger := utils.GetLogger()

		// 1. 全局IP限流器
		capacity := cfg.RateLimiter.Global.Capacity
		requestsPerMinute := cfg.RateLimiter.Global.RequestsPerMinute
		maxSize := cfg.RateLimiter.Global.MaxCacheSize
		refillRate := time.Minute / time.Duration(requestsPerMinute)
		cleanupInterval := cfg.RateLimiter.CleanupInterval
		expireTime := cfg.RateLimiter.EntryExpireTime

		globalIPRateLimiter = NewLRURateLimiter(capacity, refillRate, maxSize, cleanupInterval, expireTime)
		logger.Info("全局限流器初始化完成",
			"capacity", capacity,
			"requestsPerMinute", requestsPerMinute,
			"maxSize", maxSize)

		// 2. 登录限流器
		loginCapacity := cfg.RateLimiter.Login.Capacity
		loginRPM := cfg.RateLimiter.Login.RequestsPerMinute
		loginMaxSize := cfg.RateLimiter.Login.MaxCacheSize
		loginRefillRate := time.Minute / time.Duration(loginRPM)

		globalLoginRateLimiter = NewLRURateLimiter(loginCapacity, loginRefillRate, loginMaxSize, cleanupInterval, expireTime)
		logger.Info("登录限流器初始化完成",
			"capacity", loginCapacity,
			"requestsPerMinute", loginRPM,
			"maxSize", loginMaxSize)

		// 3. 注册限流器
		regCapacity := cfg.RateLimiter.Register.Capacity
		regRPM := cfg.RateLimiter.Register.RequestsPerMinute
		regMaxSize := cfg.RateLimiter.Register.MaxCacheSize
		regRefillRate := time.Minute / time.Duration(regRPM)

		globalRegisterRateLimiter = NewLRURateLimiter(regCapacity, regRefillRate, regMaxSize, cleanupInterval, expireTime)
		logger.Info("注册限流器初始化完成",
			"capacity", regCapacity,
			"requestsPerMinute", regRPM,
			"maxSize", regMaxSize)

		logger.Info("所有限流器初始化完成（LRU）")
	})
}

// ShutdownRateLimiters 优雅关闭所有限流器，释放资源
func ShutdownRateLimiters() {
	logger := utils.GetLogger()
	logger.Info("开始关闭所有限流器")

	if globalIPRateLimiter != nil {
		globalIPRateLimiter.Stop()
		logger.Info("全局限流器已关闭")
	}
	if globalLoginRateLimiter != nil {
		globalLoginRateLimiter.Stop()
		logger.Info("登录限流器已关闭")
	}
	if globalRegisterRateLimiter != nil {
		globalRegisterRateLimiter.Stop()
		logger.Info("注册限流器已关闭")
	}

	logger.Info("所有限流器已关闭")
}

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// globalIPRateLimiter should be initialized before routes setup
		if globalIPRateLimiter == nil {
			utils.GetLogger().Error("Global rate limiter not initialized")
			c.Next()
			return
		}

		clientIP := c.ClientIP()

		if !globalIPRateLimiter.Allow(clientIP) {
			utils.TooManyRequestsResponse(c, "请求频率过高，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}

// LoginRateLimitMiddleware 登录限流中间件（使用全局限流器，防止内存泄漏）
func LoginRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if globalLoginRateLimiter == nil {
			utils.GetLogger().Error("登录限流器未初始化")
			// 限流器未初始化时不阻止请求，但记录错误
			c.Next()
			return
		}

		clientIP := c.ClientIP()

		if !globalLoginRateLimiter.Allow(clientIP) {
			utils.TooManyRequestsResponse(c, "登录尝试次数过多，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RegisterRateLimitMiddleware 注册限流中间件（使用全局限流器，防止内存泄漏）
func RegisterRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if globalRegisterRateLimiter == nil {
			utils.GetLogger().Error("注册限流器未初始化")
			// 限流器未初始化时不阻止请求，但记录错误
			c.Next()
			return
		}

		clientIP := c.ClientIP()

		if !globalRegisterRateLimiter.Allow(clientIP) {
			utils.TooManyRequestsResponse(c, "注册尝试次数过多，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}
