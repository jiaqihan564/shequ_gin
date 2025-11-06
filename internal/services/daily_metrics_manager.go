package services

import (
	"sync"
	"sync/atomic"
	"time"

	"gin/internal/config"
)

// DailyMetricsManager 每日指标管理器（内存中维护今日数据）
// 优化：使用atomic操作减少锁竞争，提升高并发性能
type DailyMetricsManager struct {
	mu sync.RWMutex

	// 今日数据
	currentDate   string
	activeUserIDs map[uint]bool // 活跃用户ID集合（去重）- 需要锁保护

	// 使用atomic操作的计数器（无锁，高性能）
	totalRequests     int64 // atomic
	successRequests   int64 // atomic
	errorRequests     int64 // atomic
	totalLatency      int64 // atomic - 累计响应时间（毫秒）
	newUsers          int64 // atomic
	peakConcurrent    int64 // atomic
	currentConcurrent int64 // atomic - 当前并发请求数

	// 接口调用计数（需要锁保护）
	endpointCallCount map[string]int64

	// 配置的初始容量（用于重置时）
	activeUsersCapacity   int
	endpointCallsCapacity int

	// 配置
	config *config.Config
}

// NewDailyMetricsManager 创建每日指标管理器
func NewDailyMetricsManager(cfg *config.Config) *DailyMetricsManager {
	activeUsersInitial := 500  // 默认值
	endpointCallsInitial := 50 // 默认值
	if cfg != nil {
		activeUsersInitial = cfg.MetricsCapacity.ActiveUsersInitial
		endpointCallsInitial = cfg.MetricsCapacity.EndpointCallsInitial
	}

	// 获取日期格式
	dateFormat := "2006-01-02" // 默认格式
	if cfg != nil {
		dateFormat = cfg.DateTimeFormats.DateOnly
	}

	return &DailyMetricsManager{
		currentDate:           time.Now().UTC().Format(dateFormat),
		activeUserIDs:         make(map[uint]bool, activeUsersInitial),      // 从配置读取容量
		endpointCallCount:     make(map[string]int64, endpointCallsInitial), // 从配置读取容量
		activeUsersCapacity:   activeUsersInitial,                           // 保存容量用于重置
		endpointCallsCapacity: endpointCallsInitial,                         // 保存容量用于重置
		config:                cfg,
	}
}

// RecordLogin 记录登录（添加活跃用户）
func (m *DailyMetricsManager) RecordLogin(userID uint) {
	// 先检查日期（在加锁前），避免在持有锁的情况下再次尝试加锁导致死锁
	m.checkDateAndReset()

	// 然后加锁添加用户
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeUserIDs[userID] = true
}

// RecordRegister 记录注册（优化：使用atomic）
func (m *DailyMetricsManager) RecordRegister() {
	m.checkDateAndReset() // 轻量级检查
	atomic.AddInt64(&m.newUsers, 1)
}

// RecordRequest 记录请求（优化：大部分使用atomic，只有endpoint计数需要锁）
func (m *DailyMetricsManager) RecordRequest(endpoint string, latencyMs int64, isSuccess, isError bool) {
	m.checkDateAndReset() // 轻量级检查

	// 使用atomic操作（无锁，高性能）
	atomic.AddInt64(&m.totalRequests, 1)
	atomic.AddInt64(&m.totalLatency, latencyMs)

	if isSuccess {
		atomic.AddInt64(&m.successRequests, 1)
	}
	if isError {
		atomic.AddInt64(&m.errorRequests, 1)
	}

	// 接口调用计数需要锁（map不支持并发写）
	m.mu.Lock()
	m.endpointCallCount[endpoint]++
	m.mu.Unlock()
}

// IncrementConcurrent 增加并发计数（优化：使用atomic）
func (m *DailyMetricsManager) IncrementConcurrent() {
	current := atomic.AddInt64(&m.currentConcurrent, 1)

	// 更新峰值（使用atomic CAS循环）
	for {
		peak := atomic.LoadInt64(&m.peakConcurrent)
		if current <= peak {
			break
		}
		if atomic.CompareAndSwapInt64(&m.peakConcurrent, peak, current) {
			break
		}
	}
}

// DecrementConcurrent 减少并发计数（优化：使用atomic）
func (m *DailyMetricsManager) DecrementConcurrent() {
	atomic.AddInt64(&m.currentConcurrent, -1)

	// 确保不会为负数（使用CAS）
	for {
		current := atomic.LoadInt64(&m.currentConcurrent)
		if current >= 0 {
			break
		}
		if atomic.CompareAndSwapInt64(&m.currentConcurrent, current, 0) {
			break
		}
	}
}

// GetTodayMetrics 获取今日指标（优化：使用atomic读取）
func (m *DailyMetricsManager) GetTodayMetrics() (activeUsers, newUsers, totalReqs, peakConcurrent int, avgLatency, successRate float64, mostPopularEndpoint string) {
	// 使用atomic读取无锁计数器（高性能）
	newUsers = int(atomic.LoadInt64(&m.newUsers))
	totalReqs = int(atomic.LoadInt64(&m.totalRequests))
	peakConcurrent = int(atomic.LoadInt64(&m.peakConcurrent))
	totalLat := atomic.LoadInt64(&m.totalLatency)
	successReqs := atomic.LoadInt64(&m.successRequests)

	// 计算平均响应时间
	if totalReqs > 0 {
		avgLatency = float64(totalLat) / float64(totalReqs)
	}

	// 计算成功率
	if totalReqs > 0 {
		successRate = float64(successReqs) * 100.0 / float64(totalReqs)
	}

	// 读取map需要锁
	m.mu.RLock()
	activeUsers = len(m.activeUserIDs)

	// 找出最受欢迎的接口
	maxCalls := int64(0)
	for endpoint, calls := range m.endpointCallCount {
		if calls > maxCalls {
			maxCalls = calls
			mostPopularEndpoint = endpoint
		}
	}
	m.mu.RUnlock()

	return
}

// checkDateAndReset 轻量级检查日期（优化：使用读锁先检查，避免频繁加写锁）
func (m *DailyMetricsManager) checkDateAndReset() {
	dateFormat := "2006-01-02"
	if m.config != nil {
		dateFormat = m.config.DateTimeFormats.DateOnly
	}
	today := time.Now().UTC().Format(dateFormat)

	// 先用读锁检查
	m.mu.RLock()
	needReset := (today != m.currentDate)
	m.mu.RUnlock()

	if !needReset {
		return
	}

	// 需要重置，获取写锁
	m.mu.Lock()
	defer m.mu.Unlock()

	// 双重检查（防止并发重置）
	if today != m.currentDate {
		// 新的一天，重置所有计数
		m.currentDate = today
		m.activeUserIDs = make(map[uint]bool, m.activeUsersCapacity) // 使用保存的容量值
		atomic.StoreInt64(&m.totalRequests, 0)
		atomic.StoreInt64(&m.successRequests, 0)
		atomic.StoreInt64(&m.errorRequests, 0)
		atomic.StoreInt64(&m.totalLatency, 0)
		atomic.StoreInt64(&m.newUsers, 0)
		atomic.StoreInt64(&m.peakConcurrent, 0)
		// 注意：不重置 currentConcurrent
		// 原因：在日期切换时可能还有跨日请求在处理中（从昨天开始但还未完成）
		// 如果重置为 0，这些请求完成时调用 DecrementConcurrent() 会导致计数变为负数
		// currentConcurrent 会随着请求自然完成而归零
		m.endpointCallCount = make(map[string]int64, m.endpointCallsCapacity) // 使用保存的容量值
	}
}

// 全局单例
var globalDailyMetricsManager *DailyMetricsManager
var dailyMetricsOnce sync.Once
var dailyMetricsConfig *config.Config

// InitDailyMetricsManager 初始化每日指标管理器（必须在使用前调用）
func InitDailyMetricsManager(cfg *config.Config) {
	dailyMetricsConfig = cfg
}

// GetDailyMetricsManager 获取全局每日指标管理器
func GetDailyMetricsManager() *DailyMetricsManager {
	dailyMetricsOnce.Do(func() {
		globalDailyMetricsManager = NewDailyMetricsManager(dailyMetricsConfig)
	})
	return globalDailyMetricsManager
}
