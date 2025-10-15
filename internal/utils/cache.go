package utils

import (
	"sync"
	"time"
)

// CacheItem 缓存项
type CacheItem struct {
	Value      interface{}
	ExpireTime time.Time
}

// IsExpired 检查是否过期
func (item *CacheItem) IsExpired() bool {
	return time.Now().After(item.ExpireTime)
}

// MemoryCache 内存缓存
type MemoryCache struct {
	items      map[string]*CacheItem
	mutex      sync.RWMutex
	stopClean  chan struct{}
	defaultTTL time.Duration
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache(defaultTTL time.Duration) *MemoryCache {
	cache := &MemoryCache{
		items:      make(map[string]*CacheItem),
		stopClean:  make(chan struct{}),
		defaultTTL: defaultTTL,
	}

	// 启动定期清理过期条目（每5分钟清理一次）
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				cache.cleanup()
			case <-cache.stopClean:
				return
			}
		}
	}()

	return cache
}

// Set 设置缓存项（使用默认TTL）
func (c *MemoryCache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL 设置缓存项（指定TTL）
func (c *MemoryCache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items[key] = &CacheItem{
		Value:      value,
		ExpireTime: time.Now().Add(ttl),
	}
}

// Get 获取缓存项
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	if item.IsExpired() {
		// 过期了，异步删除
		go c.Delete(key)
		return nil, false
	}

	return item.Value, true
}

// Delete 删除缓存项
func (c *MemoryCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.items, key)
}

// Clear 清空缓存
func (c *MemoryCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.items = make(map[string]*CacheItem)
}

// cleanup 清理过期条目
func (c *MemoryCache) cleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	removed := 0

	for key, item := range c.items {
		if now.After(item.ExpireTime) {
			delete(c.items, key)
			removed++
		}
	}

	if removed > 0 {
		GetLogger().Info("清理过期缓存条目", "removed", removed, "remaining", len(c.items))
	}
}

// Stop 停止清理goroutine
func (c *MemoryCache) Stop() {
	close(c.stopClean)
}

// Size 获取缓存大小
func (c *MemoryCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.items)
}

// 全局缓存实例
var globalCache *MemoryCache
var cacheOnce sync.Once

// GetCache 获取全局缓存实例
func GetCache() *MemoryCache {
	cacheOnce.Do(func() {
		globalCache = NewMemoryCache(5 * time.Minute)
	})
	return globalCache
}
