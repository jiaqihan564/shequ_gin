package services

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// RealtimeMetricsManager 实时指标管理器
type RealtimeMetricsManager struct {
	mu sync.RWMutex

	// 在线用户（最近5分钟活跃的用户）
	onlineUsers map[uint]time.Time // userID -> 最后活跃时间

	// QPS统计
	requestCount int64 // 当前秒的请求数
	lastSecond   int64 // 上一秒的时间戳
	currentQPS   int32 // 当前QPS

	// 最后错误时间
	lastErrorTime string

	// 启动时间
	startTime time.Time
}

var (
	globalRealtimeMetricsManager *RealtimeMetricsManager
	realtimeMetricsOnce          sync.Once
)

// GetRealtimeMetricsManager 获取全局实时指标管理器
func GetRealtimeMetricsManager() *RealtimeMetricsManager {
	realtimeMetricsOnce.Do(func() {
		globalRealtimeMetricsManager = &RealtimeMetricsManager{
			onlineUsers: make(map[uint]time.Time, 1000), // 预分配容量（性能优化）
			startTime:   time.Now(),
		}

		// 启动清理协程（每分钟清理超时的在线用户）
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

	m.onlineUsers[userID] = time.Now()
}

// RecordRequest 记录请求（用于QPS统计）
func (m *RealtimeMetricsManager) RecordRequest() {
	currentSecond := time.Now().Unix()
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

	m.lastErrorTime = time.Now().Format("2006-01-02 15:04:05")
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

	// CPU使用率的简化估算：基于Goroutine数量
	// 正常情况下Goroutine数量在10-50之间，超过100视为高负载
	numGoroutine := runtime.NumGoroutine()
	cpuPercent = float64(numGoroutine) / 200.0 * 100.0 // 200个goroutine视为100%
	if cpuPercent > 100.0 {
		cpuPercent = 100.0
	}

	return
}

// cleanupOnlineUsers 定期清理超时的在线用户（5分钟无活动视为离线）
func (m *RealtimeMetricsManager) cleanupOnlineUsers() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for userID, lastActive := range m.onlineUsers {
			// 超过5分钟无活动，视为离线
			if now.Sub(lastActive) > 5*time.Minute {
				delete(m.onlineUsers, userID)
			}
		}
		m.mu.Unlock()
	}
}
