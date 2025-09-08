package middleware

import (
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

// IPRateLimiter IP限流器
type IPRateLimiter struct {
	limiters   map[string]*TokenBucket
	mutex      sync.RWMutex
	capacity   int
	refillRate time.Duration
}

// NewIPRateLimiter 创建IP限流器
func NewIPRateLimiter(capacity int, refillRate time.Duration) *IPRateLimiter {
	return &IPRateLimiter{
		limiters:   make(map[string]*TokenBucket),
		capacity:   capacity,
		refillRate: refillRate,
	}
}

// GetLimiter 获取指定IP的限流器
func (rl *IPRateLimiter) GetLimiter(ip string) *TokenBucket {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	limiter, exists := rl.limiters[ip]
	if !exists {
		limiter = NewTokenBucket(rl.capacity, rl.refillRate)
		rl.limiters[ip] = limiter
	}
	return limiter
}

// Allow 检查IP是否允许请求
func (rl *IPRateLimiter) Allow(ip string) bool {
	limiter := rl.GetLimiter(ip)
	return limiter.Allow(ip)
}

// Reset 重置IP的限流器
func (rl *IPRateLimiter) Reset(ip string) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	if limiter, exists := rl.limiters[ip]; exists {
		limiter.Reset(ip)
	}
}

// 全局IP限流器
var globalIPRateLimiter *IPRateLimiter

// InitRateLimiter 初始化限流器
func InitRateLimiter(cfg *config.Config) {
	// 默认配置：每分钟100个请求，每秒补充1.67个令牌
	capacity := 100
	refillRate := time.Second * 100 / 167 // 避免浮点数截断

	// 如果配置中有限流设置，使用配置值
	if cfg != nil {
		// 可以从配置中读取限流参数
		// 这里暂时使用默认值，后续可以扩展配置
	}

	globalIPRateLimiter = NewIPRateLimiter(capacity, refillRate)
	utils.GetLogger().Info("限流器初始化完成", "capacity", capacity, "refillRate", refillRate)
}

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if globalIPRateLimiter == nil {
			// 如果没有初始化限流器，使用默认配置
			InitRateLimiter(nil)
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

// LoginRateLimitMiddleware 登录限流中间件
func LoginRateLimitMiddleware() gin.HandlerFunc {
	// 登录限流更严格：每分钟5次尝试
	loginLimiter := NewIPRateLimiter(5, time.Minute/5)

	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		if !loginLimiter.Allow(clientIP) {
			utils.TooManyRequestsResponse(c, "登录尝试次数过多，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RegisterRateLimitMiddleware 注册限流中间件
func RegisterRateLimitMiddleware() gin.HandlerFunc {
	// 注册限流：每分钟10次尝试
	registerLimiter := NewIPRateLimiter(10, time.Minute/10)

	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		if !registerLimiter.Allow(clientIP) {
			utils.TooManyRequestsResponse(c, "注册尝试次数过多，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}
