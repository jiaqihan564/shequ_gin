package services

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"gin/internal/models"
	"gin/internal/utils"
)

// CumulativeStatsRepository 累计统计数据访问层
type CumulativeStatsRepository struct {
	db     *Database
	logger utils.Logger
}

// NewCumulativeStatsRepository 创建累计统计数据访问层
func NewCumulativeStatsRepository(db *Database) *CumulativeStatsRepository {
	return &CumulativeStatsRepository{
		db:     db,
		logger: utils.GetLogger(),
	}
}

// IncrementCumulativeStat 递增累计统计
func (r *CumulativeStatsRepository) IncrementCumulativeStat(statKey string, increment int64) error {
	query := `UPDATE cumulative_statistics SET stat_value = stat_value + ? WHERE stat_key = ?`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.db.DB.ExecContext(ctx, query, increment, statKey)
	if err != nil {
		r.logger.Error("递增累计统计失败",
			"statKey", statKey,
			"increment", increment,
			"error", err.Error())
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		r.logger.Warn("累计统计项不存在或未更新",
			"statKey", statKey,
			"increment", increment)
	}

	return nil
}

// GetAllCumulativeStats 获取所有累计统计（按分类分组）
func (r *CumulativeStatsRepository) GetAllCumulativeStats() (*models.CumulativeStatsResponse, error) {
	query := `SELECT stat_key, stat_value, stat_desc, category, updated_at 
			  FROM cumulative_statistics 
			  ORDER BY category, stat_key`

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := r.db.DB.QueryContext(ctx, query)
	if err != nil {
		r.logger.Error("查询累计统计失败", "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}
	defer rows.Close()

	// 预分配slice容量（性能优化）
	response := &models.CumulativeStatsResponse{
		User:     make([]models.CumulativeStatistics, 0, 10),
		API:      make([]models.CumulativeStatistics, 0, 10),
		Security: make([]models.CumulativeStatistics, 0, 10),
		Content:  make([]models.CumulativeStatistics, 0, 10),
	}

	for rows.Next() {
		var stat models.CumulativeStatistics
		err := rows.Scan(
			&stat.StatKey,
			&stat.StatValue,
			&stat.StatDesc,
			&stat.Category,
			&stat.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("扫描累计统计数据失败", "error", err.Error())
			continue
		}

		// 按分类分组（使用tagged switch）
		switch category := stat.Category; category {
		case "user":
			response.User = append(response.User, stat)
		case "api":
			response.API = append(response.API, stat)
		case "security":
			response.Security = append(response.Security, stat)
		case "content":
			response.Content = append(response.Content, stat)
		default:
			r.logger.Warn("未知的统计分类", "category", category)
		}
	}

	return response, nil
}

// UpsertDailyMetric 创建或更新每日指标
func (r *CumulativeStatsRepository) UpsertDailyMetric(date string, activeUsers, newUsers, totalRequests int, avgResponseTime, successRate float64, peakConcurrent int, mostPopularEndpoint string) error {
	query := `INSERT INTO daily_metrics 
			    (date, active_users, avg_response_time, success_rate, peak_concurrent, most_popular_endpoint, new_users, total_requests) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			  ON DUPLICATE KEY UPDATE
			    active_users = VALUES(active_users),
			    avg_response_time = VALUES(avg_response_time),
			    success_rate = VALUES(success_rate),
			    peak_concurrent = GREATEST(peak_concurrent, VALUES(peak_concurrent)),
			    most_popular_endpoint = VALUES(most_popular_endpoint),
			    new_users = VALUES(new_users),
			    total_requests = VALUES(total_requests)`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.db.DB.ExecContext(ctx, query, date, activeUsers, avgResponseTime, successRate, peakConcurrent, mostPopularEndpoint, newUsers, totalRequests)
	if err != nil {
		r.logger.Error("更新每日指标失败",
			"date", date,
			"error", err.Error())
		return err
	}

	return nil
}

// GetDailyMetrics 获取每日指标
func (r *CumulativeStatsRepository) GetDailyMetrics(startDate, endDate string) ([]models.DailyMetrics, error) {
	query := `SELECT id, date, active_users, avg_response_time, success_rate, peak_concurrent, 
				most_popular_endpoint, new_users, total_requests, created_at, updated_at
			  FROM daily_metrics 
			  WHERE date >= ? AND date <= ?
			  ORDER BY date DESC`

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := r.db.DB.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		r.logger.Error("查询每日指标失败",
			"startDate", startDate,
			"endDate", endDate,
			"error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}
	defer rows.Close()

	// 预分配slice容量（性能优化，一般查询30-90天数据）
	metrics := make([]models.DailyMetrics, 0, 90)
	for rows.Next() {
		var m models.DailyMetrics
		err := rows.Scan(
			&m.ID,
			&m.Date,
			&m.ActiveUsers,
			&m.AvgResponseTime,
			&m.SuccessRate,
			&m.PeakConcurrent,
			&m.MostPopularEndpoint,
			&m.NewUsers,
			&m.TotalRequests,
			&m.CreatedAt,
			&m.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("扫描每日指标数据失败", "error", err.Error())
			continue
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

// UpdateRealtimeMetric 更新实时指标
func (r *CumulativeStatsRepository) UpdateRealtimeMetric(metricKey, metricValue string) error {
	query := `INSERT INTO realtime_metrics (metric_key, metric_value) 
			  VALUES (?, ?)
			  ON DUPLICATE KEY UPDATE metric_value = VALUES(metric_value)`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.db.DB.ExecContext(ctx, query, metricKey, metricValue)
	if err != nil {
		r.logger.Error("更新实时指标失败",
			"metricKey", metricKey,
			"metricValue", metricValue,
			"error", err.Error())
		return err
	}

	return nil
}

// GetAllRealtimeMetrics 获取所有实时指标
func (r *CumulativeStatsRepository) GetAllRealtimeMetrics() (*models.RealtimeMetricsResponse, error) {
	query := `SELECT metric_key, metric_value FROM realtime_metrics`

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := r.db.DB.QueryContext(ctx, query)
	if err != nil {
		r.logger.Error("查询实时指标失败", "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}
	defer rows.Close()

	response := &models.RealtimeMetricsResponse{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}

		switch key {
		case "online_users":
			if v, err := strconv.Atoi(value); err == nil {
				response.OnlineUsers = v
			}
		case "current_qps":
			if v, err := strconv.Atoi(value); err == nil {
				response.CurrentQPS = v
			}
		case "system_cpu":
			if v, err := strconv.ParseFloat(value, 64); err == nil {
				response.SystemCPU = v
			}
		case "system_memory":
			if v, err := strconv.ParseFloat(value, 64); err == nil {
				response.SystemMemory = v
			}
		case "service_status":
			response.ServiceStatus = value
		case "last_error_time":
			response.LastErrorTime = value
		}
	}

	return response, nil
}

// GetTodayDailyMetric 获取今日每日指标
func (r *CumulativeStatsRepository) GetTodayDailyMetric() (*models.DailyMetrics, error) {
	today := time.Now().Format("2006-01-02")

	query := `SELECT id, date, active_users, avg_response_time, success_rate, peak_concurrent, 
				most_popular_endpoint, new_users, total_requests, created_at, updated_at
			  FROM daily_metrics 
			  WHERE date = ?`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	metric := &models.DailyMetrics{}
	err := r.db.DB.QueryRowContext(ctx, query, today).Scan(
		&metric.ID,
		&metric.Date,
		&metric.ActiveUsers,
		&metric.AvgResponseTime,
		&metric.SuccessRate,
		&metric.PeakConcurrent,
		&metric.MostPopularEndpoint,
		&metric.NewUsers,
		&metric.TotalRequests,
		&metric.CreatedAt,
		&metric.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// 今日没有记录，返回默认值
			return &models.DailyMetrics{
				Date:            time.Now(),
				ActiveUsers:     0,
				AvgResponseTime: 0,
				SuccessRate:     0,
				PeakConcurrent:  0,
				NewUsers:        0,
				TotalRequests:   0,
			}, nil
		}
		r.logger.Error("查询今日指标失败", "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}

	return metric, nil
}
