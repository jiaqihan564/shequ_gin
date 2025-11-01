// Package utils 提供缓存工具包括带TTL的LRU缓存
package utils

import (
	"container/list"
	"sync"
	"sync/atomic"
	"time"
)

// CacheItem 缓存项
type CacheItem struct {
	Key        string
	Value      interface{}
	ExpireTime time.Time
	Size       int64 // 估算的内存大小（字节）
}

// IsExpired 检查是否过期
func (item *CacheItem) IsExpired() bool {
	return time.Now().After(item.ExpireTime)
}

// LRUCache LRU缓存（带容量限制和TTL）
type LRUCache struct {
	capacity   int                      // 最大条目数
	maxMemory  int64                    // 最大内存使用（字节）
	items      map[string]*list.Element // 键到链表节点的映射
	lruList    *list.List               // LRU链表（最近使用的在前）
	mutex      sync.RWMutex             // 读写锁
	stopClean  chan struct{}            // 停止清理信号
	defaultTTL time.Duration            // 默认TTL

	// 统计信息
	hits       uint64 // 命中次数
	misses     uint64 // 未命中次数
	evictions  uint64 // 淘汰次数
	currentMem int64  // 当前内存使用
}

// LRUCacheConfig LRU缓存配置
type LRUCacheConfig struct {
	Capacity   int           // 最大条目数（0表示无限制）
	MaxMemory  int64         // 最大内存（字节，0表示无限制）
	DefaultTTL time.Duration // 默认TTL
}

// NewLRUCache 创建LRU缓存
// 注意：默认值将在运行时应用，如果未提供配置则使用硬编码默认值
func NewLRUCache(config LRUCacheConfig) *LRUCache {
	// 应用默认值（这些默认值可通过全局配置覆盖）
	if config.Capacity <= 0 {
		config.Capacity = 10000 // 默认最多1万条
	}
	if config.MaxMemory <= 0 {
		config.MaxMemory = 100 * 1024 * 1024 // 默认最多100MB
	}
	if config.DefaultTTL <= 0 {
		config.DefaultTTL = 5 * time.Minute // 默认5分钟
	}

	// 清理间隔默认1分钟
	cleanupInterval := 1 * time.Minute

	cache := &LRUCache{
		capacity:   config.Capacity,
		maxMemory:  config.MaxMemory,
		items:      make(map[string]*list.Element, config.Capacity),
		lruList:    list.New(),
		stopClean:  make(chan struct{}),
		defaultTTL: config.DefaultTTL,
	}

	// 启动定期清理过期条目（使用默认清理间隔）
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				cache.cleanupExpired()
			case <-cache.stopClean:
				return
			}
		}
	}()

	return cache
}

// Set 设置缓存项（使用默认TTL）
func (c *LRUCache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL 设置缓存项（指定TTL）
func (c *LRUCache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	item := &CacheItem{
		Key:        key,
		Value:      value,
		ExpireTime: now.Add(ttl),
		Size:       estimateSize(value),
	}

	// 如果键已存在，更新并移到前面
	if elem, exists := c.items[key]; exists {
		oldItem := elem.Value.(*CacheItem)
		atomic.AddInt64(&c.currentMem, item.Size-oldItem.Size)
		elem.Value = item
		c.lruList.MoveToFront(elem)
		return
	}

	// 检查容量限制，必要时淘汰
	c.evictIfNeeded(item.Size)

	// 添加新项
	elem := c.lruList.PushFront(item)
	c.items[key] = elem
	atomic.AddInt64(&c.currentMem, item.Size)
}

// Get 获取缓存项
func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	elem, exists := c.items[key]
	if !exists {
		atomic.AddUint64(&c.misses, 1)
		return nil, false
	}

	item := elem.Value.(*CacheItem)

	// 检查是否过期
	if item.IsExpired() {
		c.removeElement(elem)
		atomic.AddUint64(&c.misses, 1)
		return nil, false
	}

	// 移到前面（最近使用）
	c.lruList.MoveToFront(elem)
	atomic.AddUint64(&c.hits, 1)
	return item.Value, true
}

// GetWithoutUpdate 获取缓存项但不更新LRU顺序（用于只读操作）
func (c *LRUCache) GetWithoutUpdate(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	elem, exists := c.items[key]
	if !exists {
		return nil, false
	}

	item := elem.Value.(*CacheItem)
	if item.IsExpired() {
		return nil, false
	}

	return item.Value, true
}

// Delete 删除缓存项
func (c *LRUCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if elem, exists := c.items[key]; exists {
		c.removeElement(elem)
	}
}

// Clear 清空缓存
func (c *LRUCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items = make(map[string]*list.Element, c.capacity)
	c.lruList.Init()
	atomic.StoreInt64(&c.currentMem, 0)
}

// Size 获取缓存大小
func (c *LRUCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.items)
}

// MemoryUsage 获取当前内存使用
func (c *LRUCache) MemoryUsage() int64 {
	return atomic.LoadInt64(&c.currentMem)
}

// Stats 获取缓存统计信息
func (c *LRUCache) Stats() CacheStats {
	hits := atomic.LoadUint64(&c.hits)
	misses := atomic.LoadUint64(&c.misses)
	total := hits + misses

	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	return CacheStats{
		Hits:        hits,
		Misses:      misses,
		HitRate:     hitRate,
		Evictions:   atomic.LoadUint64(&c.evictions),
		Size:        c.Size(),
		Capacity:    c.capacity,
		MemoryUsage: c.MemoryUsage(),
		MaxMemory:   c.maxMemory,
	}
}

// CacheStats 缓存统计信息
type CacheStats struct {
	Hits        uint64  // 命中次数
	Misses      uint64  // 未命中次数
	HitRate     float64 // 命中率（百分比）
	Evictions   uint64  // 淘汰次数
	Size        int     // 当前条目数
	Capacity    int     // 最大容量
	MemoryUsage int64   // 当前内存使用（字节）
	MaxMemory   int64   // 最大内存限制（字节）
}

// evictIfNeeded 必要时淘汰旧项
func (c *LRUCache) evictIfNeeded(newItemSize int64) {
	// 检查容量限制
	for len(c.items) >= c.capacity {
		c.evictOldest()
	}

	// 检查内存限制
	for c.currentMem+newItemSize > c.maxMemory && c.lruList.Len() > 0 {
		c.evictOldest()
	}
}

// evictOldest 淘汰最旧的项
func (c *LRUCache) evictOldest() {
	elem := c.lruList.Back()
	if elem != nil {
		c.removeElement(elem)
		atomic.AddUint64(&c.evictions, 1)
	}
}

// removeElement 移除元素
func (c *LRUCache) removeElement(elem *list.Element) {
	item := elem.Value.(*CacheItem)
	delete(c.items, item.Key)
	c.lruList.Remove(elem)
	atomic.AddInt64(&c.currentMem, -item.Size)
}

// cleanupExpired 清理过期条目
func (c *LRUCache) cleanupExpired() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	removed := 0

	// 遍历并移除过期项
	for elem := c.lruList.Back(); elem != nil; {
		item := elem.Value.(*CacheItem)
		next := elem.Prev() // 保存前一个节点

		if now.After(item.ExpireTime) {
			c.removeElement(elem)
			removed++
		}

		elem = next
	}

	if removed > 0 {
		GetLogger().Info("清理过期缓存条目",
			"removed", removed,
			"remaining", len(c.items),
			"memoryUsage", c.currentMem)
	}
}

// Stop 停止清理goroutine
func (c *LRUCache) Stop() {
	close(c.stopClean)
}

// estimateSize 估算对象大小（简单实现）
func estimateSize(value interface{}) int64 {
	// 简化实现：根据类型估算大小
	// 实际生产环境可以使用更精确的方法
	switch v := value.(type) {
	case string:
		return int64(len(v))
	case []byte:
		return int64(len(v))
	case int, int32, uint, uint32, float32:
		return 4
	case int64, uint64, float64:
		return 8
	default:
		// 默认估算为1KB
		return 1024
	}
}

// MemoryCache 原有接口的兼容包装
type MemoryCache struct {
	cache *LRUCache
}

// NewMemoryCache 创建内存缓存（兼容旧接口）
func NewMemoryCache(defaultTTL time.Duration) *MemoryCache {
	return &MemoryCache{
		cache: NewLRUCache(LRUCacheConfig{
			Capacity:   10000,
			MaxMemory:  100 * 1024 * 1024,
			DefaultTTL: defaultTTL,
		}),
	}
}

// Set 设置缓存项
func (c *MemoryCache) Set(key string, value interface{}) {
	c.cache.Set(key, value)
}

// SetWithTTL 设置缓存项（指定TTL）
func (c *MemoryCache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.cache.SetWithTTL(key, value, ttl)
}

// Get 获取缓存项
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	return c.cache.Get(key)
}

// Delete 删除缓存项
func (c *MemoryCache) Delete(key string) {
	c.cache.Delete(key)
}

// Clear 清空缓存
func (c *MemoryCache) Clear() {
	c.cache.Clear()
}

// Size 获取缓存大小
func (c *MemoryCache) Size() int {
	return c.cache.Size()
}

// Stop 停止清理
func (c *MemoryCache) Stop() {
	c.cache.Stop()
}

// Stats 获取统计信息
func (c *MemoryCache) Stats() CacheStats {
	return c.cache.Stats()
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
