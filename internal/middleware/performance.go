package middleware

import (
	"runtime"
	"time"

	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// PerformanceMiddleware 性能监控中间件
// 记录请求的性能指标：内存使用、goroutine数量、数据库连接池状态等
func PerformanceMiddleware(db *services.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		logger := utils.GetLogger()

		// 记录请求开始时的系统状态
		var memStatsBefore runtime.MemStats
		runtime.ReadMemStats(&memStatsBefore)
		goroutinesBefore := runtime.NumGoroutine()

		// 数据库连接池状态（请求前）
		var dbStatsBefore map[string]interface{}
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
			"dbConnections", dbStatsBefore)

		// 处理请求
		c.Next()

		// 请求结束，计算性能指标
		duration := time.Since(start)

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

		// 构建详细的性能日志
		performanceFields := map[string]interface{}{
			"path":           c.Request.URL.Path,
			"method":         c.Request.Method,
			"statusCode":     c.Writer.Status(),
			"duration":       duration,
			"durationMs":     duration.Milliseconds(),
			"durationMicros": duration.Microseconds(),

			// 内存指标
			"memAllocBeforeMB": float64(memStatsBefore.Alloc) / 1024 / 1024,
			"memAllocAfterMB":  float64(memStatsAfter.Alloc) / 1024 / 1024,
			"memAllocDeltaMB":  float64(memAllocDelta) / 1024 / 1024,
			"memSysMB":         float64(memStatsAfter.Sys) / 1024 / 1024,
			"memTotalAllocMB":  float64(memStatsAfter.TotalAlloc) / 1024 / 1024,
			"memHeapAllocMB":   float64(memStatsAfter.HeapAlloc) / 1024 / 1024,
			"memHeapSysMB":     float64(memStatsAfter.HeapSys) / 1024 / 1024,
			"memHeapInUseMB":   float64(memStatsAfter.HeapInuse) / 1024 / 1024,

			// GC指标
			"numGC":      numGC,
			"gcPauseNs":  memStatsAfter.PauseNs[(memStatsAfter.NumGC+255)%256],
			"lastGCTime": time.Unix(0, int64(memStatsAfter.LastGC)).Format(time.RFC3339),

			// Goroutine指标
			"goroutinesBefore": goroutinesBefore,
			"goroutinesAfter":  goroutinesAfter,
			"goroutinesDelta":  goroutinesDelta,

			// 数据库连接池指标
			"dbStatsBefore": dbStatsBefore,
			"dbStatsAfter":  dbStatsAfter,
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
			// 非常慢的请求
			logger.Warn("性能警告: 请求耗时过长", performanceFields)
		} else if duration > 500*time.Millisecond {
			// 慢请求
			logger.Info("性能监控: 慢请求", performanceFields)
		} else if duration > 200*time.Millisecond {
			// 一般慢请求
			logger.Info("性能监控: 请求完成", performanceFields)
		} else {
			// 正常请求
			logger.Debug("性能监控: 请求完成", performanceFields)
		}

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
}
