package middleware

import (
	"container/list"
	"sync"
	"time"

	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

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
func NewLRURateLimiter(capacity int, refillRate time.Duration, maxSize int) *LRURateLimiter {
	rl := &LRURateLimiter{
		capacity:   capacity,
		refillRate: refillRate,
		maxSize:    maxSize,
		limiters:   make(map[string]*lruEntry),
		lruList:    list.New(),
		stopClean:  make(chan struct{}),
	}

	// 启动定期清理（优化：缩短到10分钟）
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				rl.cleanup()
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

// cleanup 清理过期条目（30分钟未访问）
func (rl *LRURateLimiter) cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	expireTime := 30 * time.Minute
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

// 优化的全局限流器（使用LRU）
var optimizedIPRateLimiter *LRURateLimiter

// InitOptimizedRateLimiter 初始化优化的限流器
func InitOptimizedRateLimiter() {
	// 配置：每分钟100个请求，最多缓存10000个IP
	capacity := 100
	refillRate := time.Second * 60 / 100
	maxSize := 10000

	optimizedIPRateLimiter = NewLRURateLimiter(capacity, refillRate, maxSize)
	utils.GetLogger().Info("优化的限流器初始化完成（LRU）",
		"capacity", capacity,
		"refillRate", refillRate,
		"maxSize", maxSize)
}

// OptimizedRateLimitMiddleware 优化的限流中间件（使用LRU）
func OptimizedRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if optimizedIPRateLimiter == nil {
			InitOptimizedRateLimiter()
		}

		clientIP := c.ClientIP()

		if !optimizedIPRateLimiter.Allow(clientIP) {
			utils.TooManyRequestsResponse(c, "请求频率过高，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}
