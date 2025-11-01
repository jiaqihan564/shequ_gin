package utils

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"gin/internal/config"
)

// Profiler 性能分析工具
type Profiler struct {
	startTime time.Time

	// 延迟统计（P50, P95, P99）
	latencies    []time.Duration
	latencyMutex sync.Mutex
	maxLatencies int // 保留最近N个延迟记录
	cleanupRatio int // 清理百分比

	// Goroutine泄漏检测
	initialGoroutines   int
	goroutineLeakThreshold int

	// 内存统计
	memStats runtime.MemStats
}

// NewProfiler 创建性能分析器
func NewProfiler(cfg *config.ProfilerConfig) *Profiler {
	maxLatencies := 1000
	cleanupRatio := 10
	goroutineLeakThreshold := 100

	if cfg != nil {
		if cfg.LatencyMaxRecords > 0 {
			maxLatencies = cfg.LatencyMaxRecords
		}
		if cfg.LatencyCleanupRatio > 0 {
			cleanupRatio = cfg.LatencyCleanupRatio
		}
		if cfg.GoroutineLeakThreshold > 0 {
			goroutineLeakThreshold = cfg.GoroutineLeakThreshold
		}
	}

	return &Profiler{
		startTime:              time.Now(),
		latencies:              make([]time.Duration, 0, maxLatencies),
		maxLatencies:           maxLatencies,
		cleanupRatio:           cleanupRatio,
		initialGoroutines:      runtime.NumGoroutine(),
		goroutineLeakThreshold: goroutineLeakThreshold,
	}
}

// RecordLatency 记录延迟
func (p *Profiler) RecordLatency(latency time.Duration) {
	p.latencyMutex.Lock()
	defer p.latencyMutex.Unlock()

	// 保持固定大小的滑动窗口
	if len(p.latencies) >= p.maxLatencies {
		// 移除最旧的N%（使用配置的清理比例）
		removeCount := p.maxLatencies * p.cleanupRatio / 100
		if removeCount < 1 {
			removeCount = 1
		}
		p.latencies = p.latencies[removeCount:]
	}

	p.latencies = append(p.latencies, latency)
}

// GetLatencyStats 获取延迟统计（P50, P95, P99）（优化：使用sort.Slice）
func (p *Profiler) GetLatencyStats() LatencyStats {
	p.latencyMutex.Lock()
	defer p.latencyMutex.Unlock()

	if len(p.latencies) == 0 {
		return LatencyStats{}
	}

	// 复制并排序（使用标准库sort，比冒泡排序快得多）
	sorted := make([]time.Duration, len(p.latencies))
	copy(sorted, p.latencies)

	// 使用sort.Slice（O(n log n)，比冒泡排序O(n²)快得多）
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	return LatencyStats{
		P50:   percentile(sorted, 50),
		P95:   percentile(sorted, 95),
		P99:   percentile(sorted, 99),
		Min:   sorted[0],
		Max:   sorted[len(sorted)-1],
		Count: len(sorted),
	}
}

// LatencyStats 延迟统计
type LatencyStats struct {
	P50   time.Duration // 中位数
	P95   time.Duration // 95分位
	P99   time.Duration // 99分位
	Min   time.Duration // 最小值
	Max   time.Duration // 最大值
	Count int           // 样本数
}

// percentile 计算百分位数
func percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	index := (len(sorted) * p) / 100
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

// CheckGoroutineLeak 检查goroutine泄漏
func (p *Profiler) CheckGoroutineLeak() GoroutineInfo {
	current := runtime.NumGoroutine()
	leaked := current - p.initialGoroutines

	return GoroutineInfo{
		Initial: p.initialGoroutines,
		Current: current,
		Leaked:  leaked,
		IsLeak:  leaked > p.goroutineLeakThreshold, // 使用配置的泄露阈值
	}
}

// GoroutineInfo goroutine信息
type GoroutineInfo struct {
	Initial int  // 初始数量
	Current int  // 当前数量
	Leaked  int  // 泄漏数量
	IsLeak  bool // 是否泄漏
}

// GetMemoryStats 获取内存统计
func (p *Profiler) GetMemoryStats() MemoryStats {
	runtime.ReadMemStats(&p.memStats)

	return MemoryStats{
		Alloc:        p.memStats.Alloc,
		TotalAlloc:   p.memStats.TotalAlloc,
		Sys:          p.memStats.Sys,
		NumGC:        p.memStats.NumGC,
		PauseTotalNs: p.memStats.PauseTotalNs,
		HeapAlloc:    p.memStats.HeapAlloc,
		HeapSys:      p.memStats.HeapSys,
		HeapIdle:     p.memStats.HeapIdle,
		HeapInuse:    p.memStats.HeapInuse,
		HeapReleased: p.memStats.HeapReleased,
	}
}

// MemoryStats 内存统计
type MemoryStats struct {
	Alloc        uint64 // 当前分配的字节数
	TotalAlloc   uint64 // 累计分配的字节数
	Sys          uint64 // 从系统获取的字节数
	NumGC        uint32 // GC次数
	PauseTotalNs uint64 // GC总暂停时间（纳秒）
	HeapAlloc    uint64 // 堆分配
	HeapSys      uint64 // 堆系统内存
	HeapIdle     uint64 // 堆空闲
	HeapInuse    uint64 // 堆使用中
	HeapReleased uint64 // 堆已释放
}

// GetFullReport 获取完整性能报告
func (p *Profiler) GetFullReport() ProfileReport {
	return ProfileReport{
		Uptime:    time.Since(p.startTime),
		Latency:   p.GetLatencyStats(),
		Goroutine: p.CheckGoroutineLeak(),
		Memory:    p.GetMemoryStats(),
		Timestamp: time.Now(),
	}
}

// ProfileReport 性能报告
type ProfileReport struct {
	Uptime    time.Duration
	Latency   LatencyStats
	Goroutine GoroutineInfo
	Memory    MemoryStats
	Timestamp time.Time
}

// 全局profiler
var globalProfiler *Profiler
var profilerOnce sync.Once
var profilerConfig *config.ProfilerConfig

// InitGlobalProfiler 初始化全局profiler（在应用启动时调用）
func InitGlobalProfiler(cfg *config.ProfilerConfig) {
	profilerConfig = cfg
	profilerOnce.Do(func() {
		globalProfiler = NewProfiler(cfg)
	})
}

// GetGlobalProfiler 获取全局profiler
func GetGlobalProfiler() *Profiler {
	profilerOnce.Do(func() {
		// 如果没有初始化，使用默认配置
		globalProfiler = NewProfiler(profilerConfig)
	})
	return globalProfiler
}

// SlowQueryDetector 慢查询检测器
type SlowQueryDetector struct {
	threshold      time.Duration
	queries        []SlowQueryRecord
	mutex          sync.Mutex
	maxQueries     int
	cleanupRatio   int // 清理百分比
	totalQueries   uint64
	slowQueries    uint64
}

// SlowQueryRecord 慢查询记录
type SlowQueryRecord struct {
	Query     string
	Duration  time.Duration
	Timestamp time.Time
	Params    []interface{}
}

// NewSlowQueryDetector 创建慢查询检测器
func NewSlowQueryDetector(threshold time.Duration, maxRecords int, cleanupRatio int) *SlowQueryDetector {
	if maxRecords <= 0 {
		maxRecords = 100
	}
	if cleanupRatio <= 0 {
		cleanupRatio = 20 // 默认清理20%
	}
	return &SlowQueryDetector{
		threshold:    threshold,
		queries:      make([]SlowQueryRecord, 0, maxRecords),
		maxQueries:   maxRecords,
		cleanupRatio: cleanupRatio,
	}
}

// Record 记录查询
func (d *SlowQueryDetector) Record(query string, duration time.Duration, params []interface{}) {
	atomic.AddUint64(&d.totalQueries, 1)

	if duration <= d.threshold {
		return
	}

	atomic.AddUint64(&d.slowQueries, 1)

	d.mutex.Lock()
	defer d.mutex.Unlock()

	// 保持固定大小
	if len(d.queries) >= d.maxQueries {
		// 移除最旧的N%（使用配置的清理比例）
		removeCount := d.maxQueries * d.cleanupRatio / 100
		if removeCount < 1 {
			removeCount = 1
		}
		d.queries = d.queries[removeCount:]
	}

	d.queries = append(d.queries, SlowQueryRecord{
		Query:     query,
		Duration:  duration,
		Timestamp: time.Now(),
		Params:    params,
	})

	// 记录到日志
	GetLogger().Warn("检测到慢查询",
		"query", TruncateString(query, 200),
		"duration", duration,
		"threshold", d.threshold,
		"params", FormatSQLParams(params))
}

// GetSlowQueries 获取慢查询列表
func (d *SlowQueryDetector) GetSlowQueries() []SlowQueryRecord {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	result := make([]SlowQueryRecord, len(d.queries))
	copy(result, d.queries)
	return result
}

// GetStats 获取统计信息
func (d *SlowQueryDetector) GetStats() SlowQueryStats {
	total := atomic.LoadUint64(&d.totalQueries)
	slow := atomic.LoadUint64(&d.slowQueries)

	var slowRate float64
	if total > 0 {
		slowRate = float64(slow) / float64(total) * 100
	}

	return SlowQueryStats{
		TotalQueries: total,
		SlowQueries:  slow,
		SlowRate:     slowRate,
		Threshold:    d.threshold,
	}
}

// SlowQueryStats 慢查询统计
type SlowQueryStats struct {
	TotalQueries uint64
	SlowQueries  uint64
	SlowRate     float64 // 慢查询率（百分比）
	Threshold    time.Duration
}

// 全局慢查询检测器
var globalSlowQueryDetector *SlowQueryDetector
var slowQueryOnce sync.Once

// InitGlobalSlowQueryDetector 初始化全局慢查询检测器（在应用启动时调用）
func InitGlobalSlowQueryDetector(cfg *config.ProfilerConfig) {
	slowQueryOnce.Do(func() {
		threshold := 50 * time.Millisecond
		maxRecords := 100
		cleanupRatio := 20

		if cfg != nil {
			if cfg.SlowQueryThresholdMS > 0 {
				threshold = time.Duration(cfg.SlowQueryThresholdMS) * time.Millisecond
			}
			if cfg.SlowQueryMaxRecords > 0 {
				maxRecords = cfg.SlowQueryMaxRecords
			}
			if cfg.SlowQueryCleanupRatio > 0 {
				cleanupRatio = cfg.SlowQueryCleanupRatio
			}
		}

		globalSlowQueryDetector = NewSlowQueryDetector(threshold, maxRecords, cleanupRatio)
	})
}

// GetGlobalSlowQueryDetector 获取全局慢查询检测器
func GetGlobalSlowQueryDetector() *SlowQueryDetector {
	slowQueryOnce.Do(func() {
		// 如果没有初始化，使用默认配置
		globalSlowQueryDetector = NewSlowQueryDetector(50*time.Millisecond, 100, 20)
	})
	return globalSlowQueryDetector
}

// PerformanceMonitor 性能监控器（定期收集指标）
type PerformanceMonitor struct {
	profiler          *Profiler
	slowQueryDetector *SlowQueryDetector
	interval          time.Duration
	stopChan          chan struct{}
	logger            Logger
}

// NewPerformanceMonitor 创建性能监控器
func NewPerformanceMonitor(interval time.Duration) *PerformanceMonitor {
	return &PerformanceMonitor{
		profiler:          GetGlobalProfiler(),
		slowQueryDetector: GetGlobalSlowQueryDetector(),
		interval:          interval,
		stopChan:          make(chan struct{}),
		logger:            GetLogger(),
	}
}

// Start 启动监控
func (m *PerformanceMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	m.logger.Info("性能监控已启动", "interval", m.interval)

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("性能监控已停止")
			return
		case <-m.stopChan:
			m.logger.Info("性能监控已停止")
			return
		case <-ticker.C:
			m.collectMetrics()
		}
	}
}

// Stop 停止监控
func (m *PerformanceMonitor) Stop() {
	close(m.stopChan)
}

// collectMetrics 收集指标
func (m *PerformanceMonitor) collectMetrics() {
	// 检查goroutine泄漏
	goroutineInfo := m.profiler.CheckGoroutineLeak()
	if goroutineInfo.IsLeak {
		m.logger.Warn("检测到可能的goroutine泄漏",
			"initial", goroutineInfo.Initial,
			"current", goroutineInfo.Current,
			"leaked", goroutineInfo.Leaked)
	}

	// 记录内存统计
	memStats := m.profiler.GetMemoryStats()
	m.logger.Debug("内存统计",
		"alloc", formatBytes(memStats.Alloc),
		"sys", formatBytes(memStats.Sys),
		"numGC", memStats.NumGC,
		"heapInuse", formatBytes(memStats.HeapInuse))

	// 记录延迟统计
	latencyStats := m.profiler.GetLatencyStats()
	if latencyStats.Count > 0 {
		m.logger.Debug("延迟统计",
			"p50", latencyStats.P50,
			"p95", latencyStats.P95,
			"p99", latencyStats.P99,
			"count", latencyStats.Count)
	}

	// 慢查询统计
	slowQueryStats := m.slowQueryDetector.GetStats()
	if slowQueryStats.SlowQueries > 0 {
		m.logger.Info("慢查询统计",
			"total", slowQueryStats.TotalQueries,
			"slow", slowQueryStats.SlowQueries,
			"rate", slowQueryStats.SlowRate)
	}
}

// formatBytes 格式化字节数
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	return fmt.Sprintf("%d %s", bytes/div, units[exp])
}
