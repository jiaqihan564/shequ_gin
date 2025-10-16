package middleware

import (
	"sync"
	"time"

	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// Metrics 性能指标
type Metrics struct {
	RequestCount    int64         `json:"request_count"`
	TotalLatency    time.Duration `json:"total_latency"`
	AverageLatency  time.Duration `json:"average_latency"`
	ErrorCount      int64         `json:"error_count"`
	LastRequestTime time.Time     `json:"last_request_time"`
	mutex           sync.RWMutex
}

// GetMetrics 获取性能指标
func (m *Metrics) GetMetrics() *Metrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	avgLatency := time.Duration(0)
	if m.RequestCount > 0 {
		avgLatency = m.TotalLatency / time.Duration(m.RequestCount)
	}

	return &Metrics{
		RequestCount:    m.RequestCount,
		TotalLatency:    m.TotalLatency,
		AverageLatency:  avgLatency,
		ErrorCount:      m.ErrorCount,
		LastRequestTime: m.LastRequestTime,
	}
}

// RecordRequest 记录请求
func (m *Metrics) RecordRequest(latency time.Duration, isError bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.RequestCount++
	m.TotalLatency += latency
	m.LastRequestTime = time.Now()

	if isError {
		m.ErrorCount++
	}
}

// 全局指标实例
var globalMetrics *Metrics

// InitMetrics 初始化指标
func InitMetrics() {
	globalMetrics = &Metrics{}
}

// GetMetrics 获取全局指标
func GetMetrics() *Metrics {
	if globalMetrics == nil {
		InitMetrics()
	}
	return globalMetrics
}

// MetricsMiddleware 性能监控中间件
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 处理请求
		c.Next()

		// 计算延迟
		latency := time.Since(start)

		// 记录指标
		isError := c.Writer.Status() >= 400
		GetMetrics().RecordRequest(latency, isError)

		// 记录慢请求
		if latency > 1*time.Second {
			utils.GetLogger().Warn("慢请求检测",
				"path", c.Request.URL.Path,
				"method", c.Request.Method,
				"latency", latency.String(),
				"status", c.Writer.Status(),
				"ip", c.ClientIP())
		}
	}
}

// MetricsHandler 指标处理器
func MetricsHandler(c *gin.Context) {
	metrics := GetMetrics().GetMetrics()

	c.JSON(200, gin.H{
		"metrics":   metrics,
		"timestamp": time.Now(),
	})
}
