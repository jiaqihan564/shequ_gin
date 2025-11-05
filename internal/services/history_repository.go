package services

import (
	"context"
	"database/sql"
	"time"

	"gin/internal/config"
	"gin/internal/models"
	"gin/internal/utils"
)

// HistoryRepository 历史记录数据访问层
type HistoryRepository struct {
	db     *Database
	logger utils.Logger
	config *config.Config
}

// NewHistoryRepository 创建历史记录数据访问层
func NewHistoryRepository(db *Database, cfg *config.Config) *HistoryRepository {
	return &HistoryRepository{
		db:     db,
		logger: utils.GetLogger(),
		config: cfg,
	}
}

// RecordLoginHistory 记录登录历史
func (r *HistoryRepository) RecordLoginHistory(userID uint, username, loginIP, userAgent, province, city string, loginStatus int) error {
	query := `INSERT INTO user_login_history (user_id, username, login_time, login_ip, user_agent, login_status, province, city) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	ctx, cancel := context.WithTimeout(context.Background(), r.db.GetUpdateTimeout())
	defer cancel()

	_, err := r.db.DB.ExecContext(ctx, query, userID, username, time.Now().UTC(), loginIP, userAgent, loginStatus, province, city)
	if err != nil {
		r.logger.Error("记录登录历史失败",
			"userID", userID,
			"username", username,
			"province", province,
			"city", city,
			"error", err.Error())
		return err
	}

	return nil
}

// RecordOperationHistory 记录操作历史
func (r *HistoryRepository) RecordOperationHistory(userID uint, username, operationType, operationDesc, ipAddress string) error {
	query := `INSERT INTO user_operation_history (user_id, username, operation_type, operation_desc, operation_time, ip_address) 
			  VALUES (?, ?, ?, ?, ?, ?)`

	ctx, cancel := context.WithTimeout(context.Background(), r.db.GetUpdateTimeout())
	defer cancel()

	_, err := r.db.DB.ExecContext(ctx, query, userID, username, operationType, operationDesc, time.Now().UTC(), ipAddress)
	if err != nil {
		r.logger.Error("记录操作历史失败",
			"userID", userID,
			"operationType", operationType,
			"error", err.Error())
		return err
	}

	return nil
}

// RecordProfileChange 记录资料修改历史
func (r *HistoryRepository) RecordProfileChange(userID uint, fieldName, oldValue, newValue, ipAddress string) error {
	query := `INSERT INTO profile_change_history (user_id, field_name, old_value, new_value, change_time, ip_address) 
			  VALUES (?, ?, ?, ?, ?, ?)`

	ctx, cancel := context.WithTimeout(context.Background(), r.db.GetUpdateTimeout())
	defer cancel()

	_, err := r.db.DB.ExecContext(ctx, query, userID, fieldName, oldValue, newValue, time.Now().UTC(), ipAddress)
	if err != nil {
		r.logger.Error("记录资料修改历史失败",
			"userID", userID,
			"fieldName", fieldName,
			"error", err.Error())
		return err
	}

	return nil
}

// GetLoginHistory 获取登录历史
func (r *HistoryRepository) GetLoginHistory(userID uint, limit int) ([]models.UserLoginHistory, error) {
	query := `SELECT id, user_id, username, login_time, login_ip, user_agent, login_status, province, city, created_at 
			  FROM user_login_history 
			  WHERE user_id = ?
			  ORDER BY login_time DESC
			  LIMIT ?`

	ctx, cancel := context.WithTimeout(context.Background(), r.db.GetQueryTimeout())
	defer cancel()

	rows, err := r.db.DB.QueryContext(ctx, query, userID, limit)
	if err != nil {
		r.logger.Error("查询登录历史失败",
			"userID", userID,
			"error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}
	defer rows.Close()

	// 预分配slice容量（性能优化）
	history := make([]models.UserLoginHistory, 0, limit)
	for rows.Next() {
		var h models.UserLoginHistory
		var province, city sql.NullString
		err := rows.Scan(
			&h.ID,
			&h.UserID,
			&h.Username,
			&h.LoginTime,
			&h.LoginIP,
			&h.UserAgent,
			&h.LoginStatus,
			&province,
			&city,
			&h.CreatedAt,
		)
		if err != nil {
			r.logger.Error("扫描登录历史数据失败", "error", err.Error())
			continue
		}
		if province.Valid {
			h.Province = province.String
		}
		if city.Valid {
			h.City = city.String
		}
		history = append(history, h)
	}

	return history, nil
}

// GetOperationHistory 获取操作历史
func (r *HistoryRepository) GetOperationHistory(userID uint, limit int) ([]models.UserOperationHistory, error) {
	query := `SELECT id, user_id, username, operation_type, operation_desc, operation_time, ip_address, created_at 
			  FROM user_operation_history 
			  WHERE user_id = ?
			  ORDER BY operation_time DESC
			  LIMIT ?`

	ctx, cancel := context.WithTimeout(context.Background(), r.db.GetQueryTimeout())
	defer cancel()

	rows, err := r.db.DB.QueryContext(ctx, query, userID, limit)
	if err != nil {
		r.logger.Error("查询操作历史失败",
			"userID", userID,
			"error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}
	defer rows.Close()

	// 预分配slice容量（性能优化）
	history := make([]models.UserOperationHistory, 0, limit)
	for rows.Next() {
		var h models.UserOperationHistory
		err := rows.Scan(
			&h.ID,
			&h.UserID,
			&h.Username,
			&h.OperationType,
			&h.OperationDesc,
			&h.OperationTime,
			&h.IPAddress,
			&h.CreatedAt,
		)
		if err != nil {
			r.logger.Error("扫描操作历史数据失败", "error", err.Error())
			continue
		}
		history = append(history, h)
	}

	return history, nil
}

// GetProfileChangeHistory 获取资料修改历史
func (r *HistoryRepository) GetProfileChangeHistory(userID uint, limit int) ([]models.ProfileChangeHistory, error) {
	query := `SELECT id, user_id, field_name, old_value, new_value, change_time, ip_address, created_at 
			  FROM profile_change_history 
			  WHERE user_id = ?
			  ORDER BY change_time DESC
			  LIMIT ?`

	ctx, cancel := context.WithTimeout(context.Background(), r.db.GetQueryTimeout())
	defer cancel()

	rows, err := r.db.DB.QueryContext(ctx, query, userID, limit)
	if err != nil {
		r.logger.Error("查询资料修改历史失败",
			"userID", userID,
			"error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}
	defer rows.Close()

	// 预分配slice容量（性能优化）
	history := make([]models.ProfileChangeHistory, 0, limit)
	for rows.Next() {
		var h models.ProfileChangeHistory
		err := rows.Scan(
			&h.ID,
			&h.UserID,
			&h.FieldName,
			&h.OldValue,
			&h.NewValue,
			&h.ChangeTime,
			&h.IPAddress,
			&h.CreatedAt,
		)
		if err != nil {
			r.logger.Error("扫描资料修改历史数据失败", "error", err.Error())
			continue
		}
		history = append(history, h)
	}

	return history, nil
}

// GetLocationDistribution 获取用户地区分布统计
func (r *HistoryRepository) GetLocationDistribution() (*models.LocationDistribution, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.db.GetQueryTimeout())
	defer cancel()

	// 按省份统计
	provinceQuery := `SELECT province, '' as city, COUNT(DISTINCT user_id) as user_count, COUNT(*) as login_count
					  FROM user_login_history
					  WHERE province IS NOT NULL AND province != ''
					  GROUP BY province
					  ORDER BY user_count DESC`

	provinceRows, err := r.db.DB.QueryContext(ctx, provinceQuery)
	if err != nil {
		r.logger.Error("查询省份统计失败", "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}
	defer provinceRows.Close()

	// 初始化为空数组，避免返回null
	provinceStats := make([]models.LocationStats, 0)
	uniqueProvinces := make(map[string]bool)

	for provinceRows.Next() {
		var stat models.LocationStats
		if err := provinceRows.Scan(&stat.Province, &stat.City, &stat.UserCount, &stat.LoginCount); err != nil {
			continue
		}
		provinceStats = append(provinceStats, stat)
		uniqueProvinces[stat.Province] = true
	}

	// 统计总城市数
	cityCountQuery := `SELECT COUNT(DISTINCT city) 
					   FROM user_login_history 
					   WHERE city IS NOT NULL AND city != ''`

	var totalCities int
	err = r.db.DB.QueryRowContext(ctx, cityCountQuery).Scan(&totalCities)
	if err != nil {
		r.logger.Error("查询城市总数失败", "error", err.Error())
		// 不影响主流程，继续返回
		totalCities = 0
	}

	return &models.LocationDistribution{
		ProvinceStats:  provinceStats,
		TotalProvinces: len(uniqueProvinces),
		TotalCities:    totalCities,
	}, nil
}
