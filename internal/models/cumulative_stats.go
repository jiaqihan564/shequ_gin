package models

import "time"

// CumulativeStatistics 累计统计数据
type CumulativeStatistics struct {
	ID        uint      `json:"id" db:"id"`
	StatKey   string    `json:"stat_key" db:"stat_key"`
	StatValue int64     `json:"stat_value" db:"stat_value"`
	StatDesc  string    `json:"stat_desc" db:"stat_desc"`
	Category  string    `json:"category" db:"category"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// DailyMetrics 每日指标数据
type DailyMetrics struct {
	ID                  uint      `json:"id" db:"id"`
	Date                time.Time `json:"date" db:"date"`
	ActiveUsers         int       `json:"active_users" db:"active_users"`
	AvgResponseTime     float64   `json:"avg_response_time" db:"avg_response_time"`
	SuccessRate         float64   `json:"success_rate" db:"success_rate"`
	PeakConcurrent      int       `json:"peak_concurrent" db:"peak_concurrent"`
	MostPopularEndpoint string    `json:"most_popular_endpoint" db:"most_popular_endpoint"`
	NewUsers            int       `json:"new_users" db:"new_users"`
	TotalRequests       int       `json:"total_requests" db:"total_requests"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// RealtimeMetrics 实时指标数据
type RealtimeMetrics struct {
	MetricKey   string    `json:"metric_key" db:"metric_key"`
	MetricValue string    `json:"metric_value" db:"metric_value"`
	MetricDesc  string    `json:"metric_desc" db:"metric_desc"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// CumulativeStatsResponse 累计统计响应
type CumulativeStatsResponse struct {
	User     []CumulativeStatistics `json:"user"`
	API      []CumulativeStatistics `json:"api"`
	Security []CumulativeStatistics `json:"security"`
	Content  []CumulativeStatistics `json:"content"`
}

// DailyMetricsResponse 每日指标响应
type DailyMetricsResponse struct {
	Today   DailyMetrics      `json:"today"`
	Trend   []DailyMetrics    `json:"trend"`
	Summary DailyMetricsStats `json:"summary"`
}

// DailyMetricsStats 每日指标统计汇总
type DailyMetricsStats struct {
	AvgActiveUsers    float64 `json:"avg_active_users"`
	AvgResponseTime   float64 `json:"avg_response_time"`
	AvgSuccessRate    float64 `json:"avg_success_rate"`
	MaxPeakConcurrent int     `json:"max_peak_concurrent"`
	TotalNewUsers     int     `json:"total_new_users"`
}

// RealtimeMetricsResponse 实时指标响应
type RealtimeMetricsResponse struct {
	OnlineUsers   int     `json:"online_users"`
	CurrentQPS    int     `json:"current_qps"`
	SystemCPU     float64 `json:"system_cpu"`
	SystemMemory  float64 `json:"system_memory"`
	ServiceStatus string  `json:"service_status"`
	LastErrorTime string  `json:"last_error_time"`
}
