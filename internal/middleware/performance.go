package middleware

import (
	"runtime"
	"sync/atomic"
	"time"

	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

var (
	// 性能监控采样计数器
	perfSampleCounter uint64
	// 采样率：10表示10%的请求进行详细性能监控
	perfSampleRate = 10
)

// shouldMonitorPerformance 判断是否应该进行详细性能监控（采样）
func shouldMonitorPerformance(c *gin.Context) bool {
	// 错误请求总是监控
	if c.Writer.Status() >= 400 {
		return true
	}

	// 开发模式下监控所有请求
	if gin.Mode() != gin.ReleaseMode {
		return true
	}

	// 生产模式下进行采样（10%）
	counter := atomic.AddUint64(&perfSampleCounter, 1)
	return (counter % uint64(100/perfSampleRate)) == 0
}

// PerformanceMiddleware 性能监控中间件（优化版）
// 记录请求的性能指标：内存使用、goroutine数量、数据库连接池状态等
// 使用采样机制减少 runtime.ReadMemStats 的性能开销
func PerformanceMiddleware(db *services.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		logger := utils.GetLogger()

		// 决定是否进行详细监控（采样）
		detailedMonitoring := shouldMonitorPerformance(c)

		// 记录请求开始时的系统状态（仅在采样时）
		var memStatsBefore runtime.MemStats
		var goroutinesBefore int
		var dbStatsBefore map[string]interface{}

		if detailedMonitoring {
			// ReadMemStats 开销较大（10-50微秒），只在采样时调用
			runtime.ReadMemStats(&memStatsBefore)
			goroutinesBefore = runtime.NumGoroutine()

			// 数据库连接池状态（请求前）
			if db != nil && db.DB != nil {
				stats := db.DB.Stats()
				dbStatsBefore = map[string]interface{}{
					"openConnections":   stats.OpenConnections,
					"inUse":             stats.InUse,
					"idle":              stats.Idle,
					"waitCount":         stats.WaitCount,
					"waitDuration":      stats.WaitDuration,
					"maxIdleClosed":     stats.MaxIdleClosed,
					"maxLifetimeClosed": stats.MaxLifetimeClosed,
				}
			}

			logger.Debug("请求开始-性能基线",
				"path", c.Request.URL.Path,
				"method", c.Request.Method,
				"goroutines", goroutinesBefore,
				"memAllocMB", float64(memStatsBefore.Alloc)/1024/1024,
				"memSysMB", float64(memStatsBefore.Sys)/1024/1024,
				"dbConnections", dbStatsBefore,
				"sampled", true)
		}

		// 处理请求
		c.Next()

		// 请求结束，计算性能指标
		duration := time.Since(start)

		// 构建基础性能日志（总是记录）
		performanceFields := map[string]interface{}{
			"path":       c.Request.URL.Path,
			"method":     c.Request.Method,
			"statusCode": c.Writer.Status(),
			"duration":   duration,
			"durationMs": duration.Milliseconds(),
		}

		// 详细监控（仅在采样时）
		if detailedMonitoring {
			// 记录请求结束时的系统状态
			var memStatsAfter runtime.MemStats
			runtime.ReadMemStats(&memStatsAfter)
			goroutinesAfter := runtime.NumGoroutine()

			// 数据库连接池状态（请求后）
			var dbStatsAfter map[string]interface{}
			if db != nil && db.DB != nil {
				stats := db.DB.Stats()
				dbStatsAfter = map[string]interface{}{
					"openConnections": stats.OpenConnections,
					"inUse":           stats.InUse,
					"idle":            stats.Idle,
					"waitCount":       stats.WaitCount,
					"waitDuration":    stats.WaitDuration,
				}
			}

			// 计算增量
			memAllocDelta := int64(memStatsAfter.Alloc) - int64(memStatsBefore.Alloc)
			goroutinesDelta := goroutinesAfter - goroutinesBefore
			numGC := memStatsAfter.NumGC - memStatsBefore.NumGC

			// 添加详细指标
			performanceFields["durationMicros"] = duration.Microseconds()
			performanceFields["sampled"] = true

			// 内存指标
			performanceFields["memAllocBeforeMB"] = float64(memStatsBefore.Alloc) / 1024 / 1024
			performanceFields["memAllocAfterMB"] = float64(memStatsAfter.Alloc) / 1024 / 1024
			performanceFields["memAllocDeltaMB"] = float64(memAllocDelta) / 1024 / 1024
			performanceFields["memSysMB"] = float64(memStatsAfter.Sys) / 1024 / 1024
			performanceFields["memTotalAllocMB"] = float64(memStatsAfter.TotalAlloc) / 1024 / 1024
			performanceFields["memHeapAllocMB"] = float64(memStatsAfter.HeapAlloc) / 1024 / 1024
			performanceFields["memHeapSysMB"] = float64(memStatsAfter.HeapSys) / 1024 / 1024
			performanceFields["memHeapInUseMB"] = float64(memStatsAfter.HeapInuse) / 1024 / 1024

			// GC指标
			performanceFields["numGC"] = numGC
			performanceFields["gcPauseNs"] = memStatsAfter.PauseNs[(memStatsAfter.NumGC+255)%256]
			performanceFields["lastGCTime"] = time.Unix(0, int64(memStatsAfter.LastGC)).Format(time.RFC3339)

			// Goroutine指标
			performanceFields["goroutinesBefore"] = goroutinesBefore
			performanceFields["goroutinesAfter"] = goroutinesAfter
			performanceFields["goroutinesDelta"] = goroutinesDelta

			// 数据库连接池指标
			performanceFields["dbStatsBefore"] = dbStatsBefore
			performanceFields["dbStatsAfter"] = dbStatsAfter

			// 内存增长警告
			if memAllocDelta > 10*1024*1024 { // 大于10MB
				logger.Warn("内存使用警告: 单次请求内存增长过大",
					"path", c.Request.URL.Path,
					"memAllocDeltaMB", float64(memAllocDelta)/1024/1024,
					"duration", duration)
			}

			// Goroutine泄露警告
			if goroutinesDelta > 10 {
				logger.Warn("Goroutine警告: 请求后goroutine数量增加较多",
					"path", c.Request.URL.Path,
					"goroutinesDelta", goroutinesDelta,
					"goroutinesAfter", goroutinesAfter)
			}

			// 数据库连接池警告
			if db != nil && db.DB != nil {
				stats := db.DB.Stats()
				maxConns := stats.MaxOpenConnections
				if maxConns > 0 && stats.OpenConnections > int(float64(maxConns)*0.8) {
					logger.Warn("数据库连接池警告: 连接使用率过高",
						"path", c.Request.URL.Path,
						"openConnections", stats.OpenConnections,
						"maxOpenConnections", maxConns,
						"utilizationPercent", float64(stats.OpenConnections)/float64(maxConns)*100)
				}
			}
		}

		// 添加用户信息（如果已认证）
		if userID, exists := c.Get("userID"); exists {
			performanceFields["userID"] = userID
		}
		if requestID, exists := c.Get("requestID"); exists {
			performanceFields["requestID"] = requestID
		}

		// 根据性能情况选择日志级别
		if duration > 1*time.Second {
			// 非常慢的请求（总是记录）
			logger.Warn("性能警告: 请求耗时过长", performanceFields)
		} else if duration > 500*time.Millisecond {
			// 慢请求（总是记录）
			logger.Info("性能监控: 慢请求", performanceFields)
		} else if detailedMonitoring {
			// 采样的正常请求
			if duration > 200*time.Millisecond {
				logger.Info("性能监控: 请求完成", performanceFields)
			} else {
				logger.Debug("性能监控: 请求完成", performanceFields)
			}
		} else {
			// 未采样的正常请求（只记录debug级别）
			logger.Debug("性能监控: 请求完成（未采样）", performanceFields)
		}
	}
}
