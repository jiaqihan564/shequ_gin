package models

import "time"

// UserStatistics 用户统计数据
type UserStatistics struct {
	ID            uint      `json:"id" db:"id"`
	Date          time.Time `json:"date" db:"date"`
	LoginCount    int       `json:"login_count" db:"login_count"`
	RegisterCount int       `json:"register_count" db:"register_count"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// ApiStatistics API统计数据
type ApiStatistics struct {
	ID           uint      `json:"id" db:"id"`
	Date         time.Time `json:"date" db:"date"`
	Endpoint     string    `json:"endpoint" db:"endpoint"`
	Method       string    `json:"method" db:"method"`
	SuccessCount int       `json:"success_count" db:"success_count"`
	ErrorCount   int       `json:"error_count" db:"error_count"`
	TotalCount   int       `json:"total_count" db:"total_count"`
	AvgLatencyMs float64   `json:"avg_latency_ms" db:"avg_latency_ms"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// StatisticsOverview 统计总览
type StatisticsOverview struct {
	TodayLogin       int     `json:"today_login"`
	TodayRegister    int     `json:"today_register"`
	TodayApiCalls    int     `json:"today_api_calls"`
	TodaySuccessRate float64 `json:"today_success_rate"`
}

// UserStatsResponse 用户统计响应
type UserStatsResponse struct {
	Stats []UserStatistics `json:"stats"`
	Total struct {
		TotalLogin    int `json:"total_login"`
		TotalRegister int `json:"total_register"`
	} `json:"total"`
}

// ApiStatsResponse API统计响应
type ApiStatsResponse struct {
	Stats []ApiStatistics `json:"stats"`
	Total struct {
		TotalCalls  int     `json:"total_calls"`
		SuccessRate float64 `json:"success_rate"`
		AvgLatency  float64 `json:"avg_latency"`
	} `json:"total"`
}

// EndpointRanking 接口排行
type EndpointRanking struct {
	Endpoint     string  `json:"endpoint"`
	Method       string  `json:"method"`
	TotalCount   int     `json:"total_count"`
	SuccessCount int     `json:"success_count"`
	ErrorCount   int     `json:"error_count"`
	SuccessRate  float64 `json:"success_rate"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
}
