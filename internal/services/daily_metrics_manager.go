package services

import (
	"sync"
	"time"
)

// DailyMetricsManager 每日指标管理器（内存中维护今日数据）
type DailyMetricsManager struct {
	mu sync.RWMutex

	// 今日数据
	currentDate       string
	activeUserIDs     map[uint]bool // 活跃用户ID集合（去重）
	totalRequests     int64
	successRequests   int64
	errorRequests     int64
	totalLatency      int64 // 累计响应时间（毫秒）
	newUsers          int64
	peakConcurrent    int64
	currentConcurrent int64            // 当前并发请求数
	endpointCallCount map[string]int64 // 接口调用次数统计
}

// NewDailyMetricsManager 创建每日指标管理器
func NewDailyMetricsManager() *DailyMetricsManager {
	return &DailyMetricsManager{
		currentDate:       time.Now().Format("2006-01-02"),
		activeUserIDs:     make(map[uint]bool),
		endpointCallCount: make(map[string]int64),
	}
}

// RecordLogin 记录登录（添加活跃用户）
func (m *DailyMetricsManager) RecordLogin(userID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.checkAndResetIfNewDay()
	m.activeUserIDs[userID] = true
}

// RecordRegister 记录注册
func (m *DailyMetricsManager) RecordRegister() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.checkAndResetIfNewDay()
	m.newUsers++
}

// RecordRequest 记录请求
func (m *DailyMetricsManager) RecordRequest(endpoint string, latencyMs int64, isSuccess, isError bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.checkAndResetIfNewDay()

	m.totalRequests++
	m.totalLatency += latencyMs

	if isSuccess {
		m.successRequests++
	}
	if isError {
		m.errorRequests++
	}

	// 统计接口调用次数
	m.endpointCallCount[endpoint]++
}

// IncrementConcurrent 增加并发计数
func (m *DailyMetricsManager) IncrementConcurrent() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.checkAndResetIfNewDay()
	m.currentConcurrent++

	// 更新峰值
	if m.currentConcurrent > m.peakConcurrent {
		m.peakConcurrent = m.currentConcurrent
	}
}

// DecrementConcurrent 减少并发计数
func (m *DailyMetricsManager) DecrementConcurrent() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.currentConcurrent--
	if m.currentConcurrent < 0 {
		m.currentConcurrent = 0
	}
}

// GetTodayMetrics 获取今日指标
func (m *DailyMetricsManager) GetTodayMetrics() (activeUsers, newUsers, totalReqs, peakConcurrent int, avgLatency, successRate float64, mostPopularEndpoint string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	activeUsers = len(m.activeUserIDs)
	newUsers = int(m.newUsers)
	totalReqs = int(m.totalRequests)
	peakConcurrent = int(m.peakConcurrent)

	// 计算平均响应时间
	if m.totalRequests > 0 {
		avgLatency = float64(m.totalLatency) / float64(m.totalRequests)
	}

	// 计算成功率
	if m.totalRequests > 0 {
		successRate = float64(m.successRequests) * 100.0 / float64(m.totalRequests)
	}

	// 找出最受欢迎的接口
	maxCalls := int64(0)
	for endpoint, calls := range m.endpointCallCount {
		if calls > maxCalls {
			maxCalls = calls
			mostPopularEndpoint = endpoint
		}
	}

	return
}

// checkAndResetIfNewDay 检查是否是新的一天，如果是则重置数据
func (m *DailyMetricsManager) checkAndResetIfNewDay() {
	today := time.Now().Format("2006-01-02")
	if today != m.currentDate {
		// 新的一天，重置所有计数
		m.currentDate = today
		m.activeUserIDs = make(map[uint]bool)
		m.totalRequests = 0
		m.successRequests = 0
		m.errorRequests = 0
		m.totalLatency = 0
		m.newUsers = 0
		m.peakConcurrent = 0
		m.endpointCallCount = make(map[string]int64)
	}
}

// 全局单例
var globalDailyMetricsManager *DailyMetricsManager
var dailyMetricsOnce sync.Once

// GetDailyMetricsManager 获取全局每日指标管理器
func GetDailyMetricsManager() *DailyMetricsManager {
	dailyMetricsOnce.Do(func() {
		globalDailyMetricsManager = NewDailyMetricsManager()
	})
	return globalDailyMetricsManager
}
