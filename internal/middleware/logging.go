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
var globalMetrics = &Metrics{}

// GetGlobalMetrics 获取全局指标
func GetGlobalMetrics() *Metrics {
	return globalMetrics
}

// LoggingMiddleware 合并的日志和监控中间件（性能优化）
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 计算延迟
		latency := time.Since(start)
		status := c.Writer.Status()
		isError := status >= 400

		// 记录性能指标
		globalMetrics.RecordRequest(latency, isError)

		// 构建日志字段
		fields := map[string]interface{}{
			"status":     status,
			"method":     c.Request.Method,
			"path":       path,
			"query":      raw,
			"ip":         c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
			"latency":    latency.String(),
			"time":       start.Format(time.RFC3339),
		}

		// 添加用户信息（如果已认证）
		if userID, exists := c.Get("userID"); exists {
			fields["user_id"] = userID
		}
		if username, exists := c.Get("username"); exists {
			fields["username"] = username
		}
		if requestID, exists := c.Get("requestID"); exists {
			fields["request_id"] = requestID
		}

		// 根据状态码和延迟选择日志级别
		logger := utils.GetLogger()
		switch {
		case status >= 500:
			logger.Error("HTTP请求", fields)
		case status >= 400:
			logger.Warn("HTTP请求", fields)
		case latency > 1*time.Second:
			logger.Warn("慢请求检测", fields)
		case latency > 500*time.Millisecond:
			logger.Info("慢请求", fields)
		}
	}
}

// MetricsHandler 指标查询处理器
func MetricsHandler(c *gin.Context) {
	metrics := globalMetrics.GetMetrics()
	c.JSON(200, gin.H{
		"metrics":   metrics,
		"timestamp": time.Now(),
	})
}
