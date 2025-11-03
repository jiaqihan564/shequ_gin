package services

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"gin/internal/config"
)

// RealtimeMetricsManager 实时指标管理器
type RealtimeMetricsManager struct {
	mu sync.RWMutex

	// 在线用户（最近N分钟活跃的用户）
	onlineUsers map[uint]time.Time // userID -> 最后活跃时间

	// QPS统计
	requestCount int64 // 当前秒的请求数
	lastSecond   int64 // 上一秒的时间戳
	currentQPS   int32 // 当前QPS

	// 最后错误时间
	lastErrorTime string

	// 启动时间
	startTime time.Time

	// 配置
	config *config.MetricsConfig
}

var (
	globalRealtimeMetricsManager *RealtimeMetricsManager
	realtimeMetricsOnce          sync.Once
)

// GetRealtimeMetricsManager 获取全局实时指标管理器
func GetRealtimeMetricsManager() *RealtimeMetricsManager {
	// Note: 这个函数可能在配置加载前被调用，使用默认值
	return GetRealtimeMetricsManagerWithConfig(nil)
}

// GetRealtimeMetricsManagerWithConfig 使用配置获取全局实时指标管理器
func GetRealtimeMetricsManagerWithConfig(cfg *config.MetricsConfig) *RealtimeMetricsManager {
	realtimeMetricsOnce.Do(func() {
		// 使用默认配置
		defaultCfg := &config.MetricsConfig{
			OnlineUsersInitialCapacity: 1000,
			OnlineUserCleanupInterval:  1,
			OnlineUserExpireTime:       5,
			CPUGoroutineBaseline:       200,
		}

		// 如果提供了配置，使用提供的配置
		if cfg != nil {
			defaultCfg = cfg
		}

		globalRealtimeMetricsManager = &RealtimeMetricsManager{
			onlineUsers: make(map[uint]time.Time, defaultCfg.OnlineUsersInitialCapacity),
			startTime:   time.Now().UTC(),
			config:      defaultCfg,
		}

		// 启动清理协程（使用配置的间隔清理超时的在线用户）
		go globalRealtimeMetricsManager.cleanupOnlineUsers()
	})
	return globalRealtimeMetricsManager
}

// RecordUserActivity 记录用户活跃
func (m *RealtimeMetricsManager) RecordUserActivity(userID uint) {
	if userID == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.onlineUsers[userID] = time.Now().UTC()
}

// RecordRequest 记录请求（用于QPS统计）
func (m *RealtimeMetricsManager) RecordRequest() {
	currentSecond := time.Now().UTC().Unix()
	lastSec := atomic.LoadInt64(&m.lastSecond)

	if currentSecond != lastSec {
		// 新的一秒，重置计数
		atomic.StoreInt64(&m.lastSecond, currentSecond)
		atomic.StoreInt32(&m.currentQPS, int32(atomic.SwapInt64(&m.requestCount, 1)))
	} else {
		// 同一秒，增加计数
		atomic.AddInt64(&m.requestCount, 1)
	}
}

// RecordError 记录错误
func (m *RealtimeMetricsManager) RecordError() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 使用标准的日期时间格式（与config.yaml中的datetime_full格式一致）
	m.lastErrorTime = time.Now().UTC().Format("2006-01-02 15:04:05")
}

// GetOnlineUsers 获取在线用户数
func (m *RealtimeMetricsManager) GetOnlineUsers() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.onlineUsers)
}

// GetCurrentQPS 获取当前QPS
func (m *RealtimeMetricsManager) GetCurrentQPS() int {
	return int(atomic.LoadInt32(&m.currentQPS))
}

// GetLastErrorTime 获取最后错误时间
func (m *RealtimeMetricsManager) GetLastErrorTime() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.lastErrorTime
}

// GetSystemMetrics 获取系统指标（CPU和内存）
func (m *RealtimeMetricsManager) GetSystemMetrics() (cpuPercent, memoryPercent float64) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	// 内存使用率 = 已分配内存 / 系统内存
	// 使用 HeapAlloc / HeapSys 作为内存使用率的近似值
	if mem.HeapSys > 0 {
		memoryPercent = float64(mem.HeapAlloc) * 100.0 / float64(mem.HeapSys)
	}

	// CPU使用率的简化估算：基于Goroutine数量（使用配置的基准值）
	numGoroutine := runtime.NumGoroutine()
	baseline := float64(m.config.CPUGoroutineBaseline)
	if baseline == 0 {
		baseline = 200.0 // 防止除零
	}
	cpuPercent = float64(numGoroutine) / baseline * 100.0
	if cpuPercent > 100.0 {
		cpuPercent = 100.0
	}

	return
}

// cleanupOnlineUsers 定期清理超时的在线用户（使用配置的过期时间）
func (m *RealtimeMetricsManager) cleanupOnlineUsers() {
	ticker := time.NewTicker(time.Duration(m.config.OnlineUserCleanupInterval) * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now().UTC()
		expireTime := time.Duration(m.config.OnlineUserExpireTime) * time.Minute
		for userID, lastActive := range m.onlineUsers {
			// 超过配置的时间无活动，视为离线
			if now.Sub(lastActive) > expireTime {
				delete(m.onlineUsers, userID)
			}
		}
		m.mu.Unlock()
	}
}
