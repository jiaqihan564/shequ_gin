package services

import (
	"context"
	"time"

	"gin/internal/config"
	"gin/internal/models"
	"gin/internal/utils"
)

// StatisticsRepository 统计数据访问层
type StatisticsRepository struct {
	db     *Database
	logger utils.Logger
	config *config.Config
}

// NewStatisticsRepository 创建统计数据访问层
func NewStatisticsRepository(db *Database, cfg *config.Config) *StatisticsRepository {
	return &StatisticsRepository{
		db:     db,
		logger: utils.GetLogger(),
		config: cfg,
	}
}

// IncrementLoginCount 增加登录次数
func (r *StatisticsRepository) IncrementLoginCount(date string) error {
	query := `INSERT INTO user_statistics (date, login_count, register_count) 
			  VALUES (?, 1, 0)
			  ON DUPLICATE KEY UPDATE login_count = login_count + 1`

	ctx, cancel := context.WithTimeout(context.Background(), r.db.GetQueryTimeout())
	defer cancel()

	_, err := r.db.DB.ExecContext(ctx, query, date)
	if err != nil {
		r.logger.Error("增加登录统计失败", "date", date, "error", err.Error())
		return err
	}

	return nil
}

// IncrementRegisterCount 增加注册次数
func (r *StatisticsRepository) IncrementRegisterCount(date string) error {
	query := `INSERT INTO user_statistics (date, login_count, register_count) 
			  VALUES (?, 0, 1)
			  ON DUPLICATE KEY UPDATE register_count = register_count + 1`

	ctx, cancel := context.WithTimeout(context.Background(), r.db.GetQueryTimeout())
	defer cancel()

	_, err := r.db.DB.ExecContext(ctx, query, date)
	if err != nil {
		r.logger.Error("增加注册统计失败", "date", date, "error", err.Error())
		return err
	}

	return nil
}

// RecordApiCall 记录API调用
func (r *StatisticsRepository) RecordApiCall(date, endpoint, method string, isSuccess, isError bool, latencyMs int64) error {
	successIncr := 0
	errorIncr := 0
	if isSuccess {
		successIncr = 1
	}
	if isError {
		errorIncr = 1
	}

	// 使用 ON DUPLICATE KEY UPDATE 实现 UPSERT
	// 平均延迟计算：(旧平均 * 旧总数 + 新延迟) / (旧总数 + 1)
	query := `INSERT INTO api_statistics (date, endpoint, method, total_count, success_count, error_count, avg_latency_ms) 
			  VALUES (?, ?, ?, 1, ?, ?, ?)
			  ON DUPLICATE KEY UPDATE 
			    total_count = total_count + 1,
			    success_count = success_count + VALUES(success_count),
			    error_count = error_count + VALUES(error_count),
			    avg_latency_ms = (avg_latency_ms * total_count + VALUES(avg_latency_ms)) / (total_count + 1)`

	ctx, cancel := context.WithTimeout(context.Background(), r.db.GetQueryTimeout())
	defer cancel()

	_, err := r.db.DB.ExecContext(ctx, query, date, endpoint, method, successIncr, errorIncr, latencyMs)
	if err != nil {
		r.logger.Error("记录API统计失败",
			"date", date,
			"endpoint", endpoint,
			"method", method,
			"error", err.Error())
		return err
	}

	return nil
}

// GetUserStatistics 获取用户统计数据
func (r *StatisticsRepository) GetUserStatistics(startDate, endDate string) ([]models.UserStatistics, error) {
	query := `SELECT id, date, login_count, register_count, created_at, updated_at 
			  FROM user_statistics 
			  WHERE date >= ? AND date <= ?
			  ORDER BY date DESC`

	ctx, cancel := context.WithTimeout(context.Background(), r.db.GetUpdateTimeout())
	defer cancel()

	rows, err := r.db.DB.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		r.logger.Error("查询用户统计失败",
			"startDate", startDate,
			"endDate", endDate,
			"error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}
	defer rows.Close()

	// 预分配slice容量（性能优化）
	stats := make([]models.UserStatistics, 0, 32)
	for rows.Next() {
		var stat models.UserStatistics
		err := rows.Scan(
			&stat.ID,
			&stat.Date,
			&stat.LoginCount,
			&stat.RegisterCount,
			&stat.CreatedAt,
			&stat.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("扫描用户统计数据失败", "error", err.Error())
			continue
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

// GetApiStatistics 获取API统计数据
func (r *StatisticsRepository) GetApiStatistics(startDate, endDate string) ([]models.ApiStatistics, error) {
	query := `SELECT id, date, endpoint, method, success_count, error_count, total_count, avg_latency_ms, created_at, updated_at 
			  FROM api_statistics 
			  WHERE date >= ? AND date <= ?
			  ORDER BY date DESC, total_count DESC`

	ctx, cancel := context.WithTimeout(context.Background(), r.db.GetUpdateTimeout())
	defer cancel()

	rows, err := r.db.DB.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		r.logger.Error("查询API统计失败",
			"startDate", startDate,
			"endDate", endDate,
			"error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}
	defer rows.Close()

	// 预分配slice容量（性能优化，预估100条API统计）
	stats := make([]models.ApiStatistics, 0, 100)
	for rows.Next() {
		var stat models.ApiStatistics
		err := rows.Scan(
			&stat.ID,
			&stat.Date,
			&stat.Endpoint,
			&stat.Method,
			&stat.SuccessCount,
			&stat.ErrorCount,
			&stat.TotalCount,
			&stat.AvgLatencyMs,
			&stat.CreatedAt,
			&stat.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("扫描API统计数据失败", "error", err.Error())
			continue
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

// GetEndpointRanking 获取接口调用排行
func (r *StatisticsRepository) GetEndpointRanking(startDate, endDate string, limit int) ([]models.EndpointRanking, error) {
	query := `SELECT 
				endpoint, 
				method, 
				SUM(total_count) as total_count,
				SUM(success_count) as success_count,
				SUM(error_count) as error_count,
				ROUND(SUM(success_count) * 100.0 / NULLIF(SUM(total_count), 0), 2) as success_rate,
				ROUND(AVG(avg_latency_ms), 2) as avg_latency_ms
			  FROM api_statistics 
			  WHERE date >= ? AND date <= ?
			  GROUP BY endpoint, method
			  ORDER BY total_count DESC
			  LIMIT ?`

	ctx, cancel := context.WithTimeout(context.Background(), r.db.GetUpdateTimeout())
	defer cancel()

	rows, err := r.db.DB.QueryContext(ctx, query, startDate, endDate, limit)
	if err != nil {
		r.logger.Error("查询接口排行失败",
			"startDate", startDate,
			"endDate", endDate,
			"error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}
	defer rows.Close()

	// 初始化为空数组，避免返回null
	rankings := make([]models.EndpointRanking, 0)
	for rows.Next() {
		var ranking models.EndpointRanking
		err := rows.Scan(
			&ranking.Endpoint,
			&ranking.Method,
			&ranking.TotalCount,
			&ranking.SuccessCount,
			&ranking.ErrorCount,
			&ranking.SuccessRate,
			&ranking.AvgLatencyMs,
		)
		if err != nil {
			r.logger.Error("扫描接口排行数据失败", "error", err.Error())
			continue
		}
		rankings = append(rankings, ranking)
	}

	return rankings, nil
}

// GetTodayOverview 获取今日总览
func (r *StatisticsRepository) GetTodayOverview() (*models.StatisticsOverview, error) {
	dateFormat := "2006-01-02"
	if r.config != nil {
		dateFormat = r.config.DateTimeFormats.DateOnly
	}
	today := time.Now().Format(dateFormat)

	overview := &models.StatisticsOverview{}

	// 获取今日用户统计
	userQuery := `SELECT COALESCE(SUM(login_count), 0), COALESCE(SUM(register_count), 0) 
				  FROM user_statistics 
				  WHERE date = ?`

	ctx, cancel := context.WithTimeout(context.Background(), r.db.GetQueryTimeout())
	defer cancel()

	err := r.db.DB.QueryRowContext(ctx, userQuery, today).Scan(&overview.TodayLogin, &overview.TodayRegister)
	if err != nil {
		r.logger.Error("查询今日用户统计失败", "error", err.Error())
		// 不返回错误，继续查询其他数据
	}

	// 获取今日API统计
	apiQuery := `SELECT 
					COALESCE(SUM(total_count), 0), 
					COALESCE(SUM(success_count), 0),
					COALESCE(SUM(error_count), 0)
				 FROM api_statistics 
				 WHERE date = ?`

	var totalCalls, successCalls, errorCalls int
	err = r.db.DB.QueryRowContext(ctx, apiQuery, today).Scan(&totalCalls, &successCalls, &errorCalls)
	if err != nil {
		r.logger.Error("查询今日API统计失败", "error", err.Error())
	}

	overview.TodayApiCalls = totalCalls
	if totalCalls > 0 {
		overview.TodaySuccessRate = float64(successCalls) * 100.0 / float64(totalCalls)
	}

	return overview, nil
}
