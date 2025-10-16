package services

import (
	"context"
	"database/sql"
	"time"

	"gin/internal/models"
	"gin/internal/utils"
)

// HistoryRepository 历史记录数据访问层
type HistoryRepository struct {
	db     *Database
	logger utils.Logger
}

// NewHistoryRepository 创建历史记录数据访问层
func NewHistoryRepository(db *Database) *HistoryRepository {
	return &HistoryRepository{
		db:     db,
		logger: utils.GetLogger(),
	}
}

// RecordLoginHistory 记录登录历史
func (r *HistoryRepository) RecordLoginHistory(userID uint, username, loginIP, userAgent, province, city string, loginStatus int) error {
	query := `INSERT INTO user_login_history (user_id, username, login_time, login_ip, user_agent, login_status, province, city) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.db.DB.ExecContext(ctx, query, userID, username, time.Now(), loginIP, userAgent, loginStatus, province, city)
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.db.DB.ExecContext(ctx, query, userID, username, operationType, operationDesc, time.Now(), ipAddress)
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.db.DB.ExecContext(ctx, query, userID, fieldName, oldValue, newValue, time.Now(), ipAddress)
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := r.db.DB.QueryContext(ctx, query, userID, limit)
	if err != nil {
		r.logger.Error("查询登录历史失败",
			"userID", userID,
			"error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}
	defer rows.Close()

	var history []models.UserLoginHistory
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := r.db.DB.QueryContext(ctx, query, userID, limit)
	if err != nil {
		r.logger.Error("查询操作历史失败",
			"userID", userID,
			"error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}
	defer rows.Close()

	var history []models.UserOperationHistory
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := r.db.DB.QueryContext(ctx, query, userID, limit)
	if err != nil {
		r.logger.Error("查询资料修改历史失败",
			"userID", userID,
			"error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}
	defer rows.Close()

	var history []models.ProfileChangeHistory
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	var provinceStats []models.LocationStats
	for provinceRows.Next() {
		var stat models.LocationStats
		if err := provinceRows.Scan(&stat.Province, &stat.City, &stat.UserCount, &stat.LoginCount); err != nil {
			continue
		}
		provinceStats = append(provinceStats, stat)
	}

	// 按城市统计（Top 20）
	cityQuery := `SELECT province, city, COUNT(DISTINCT user_id) as user_count, COUNT(*) as login_count
				  FROM user_login_history
				  WHERE city IS NOT NULL AND city != ''
				  GROUP BY province, city
				  ORDER BY user_count DESC
				  LIMIT 20`

	cityRows, err := r.db.DB.QueryContext(ctx, cityQuery)
	if err != nil {
		r.logger.Error("查询城市统计失败", "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}
	defer cityRows.Close()

	var cityStats []models.LocationStats
	uniqueProvinces := make(map[string]bool)
	uniqueCities := make(map[string]bool)

	for cityRows.Next() {
		var stat models.LocationStats
		if err := cityRows.Scan(&stat.Province, &stat.City, &stat.UserCount, &stat.LoginCount); err != nil {
			continue
		}
		cityStats = append(cityStats, stat)
		uniqueProvinces[stat.Province] = true
		uniqueCities[stat.Province+stat.City] = true
	}

	return &models.LocationDistribution{
		ProvinceStats:  provinceStats,
		CityStats:      cityStats,
		TotalProvinces: len(uniqueProvinces),
		TotalCities:    len(uniqueCities),
	}, nil
}
