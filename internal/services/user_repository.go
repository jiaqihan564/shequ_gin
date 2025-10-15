package services

import (
	"context"
	"database/sql"
	"time"

	"gin/internal/models"
	"gin/internal/utils"
)

// UserRepository 用户数据访问层
type UserRepository struct {
	db     *Database
	logger utils.Logger
}

// NewUserRepository 创建用户数据访问层
func NewUserRepository(db *Database) *UserRepository {
	return &UserRepository{
		db:     db,
		logger: utils.GetLogger(),
	}
}

// CreateUser 创建用户
func (r *UserRepository) CreateUser(ctx context.Context, user *models.User) error {
	query := `INSERT INTO user_auth (username, password_hash, email, auth_status, account_status, created_at, updated_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	result, err := r.db.DB.ExecContext(ctx, query,
		user.Username,
		user.PasswordHash,
		user.Email,
		user.AuthStatus,
		user.AccountStatus,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("创建用户失败", "username", user.Username, "error", err.Error())
		return utils.ErrDatabaseInsert
	}

	// 获取插入的ID
	id, err := result.LastInsertId()
	if err != nil {
		r.logger.Error("获取用户ID失败", "username", user.Username, "error", err.Error())
		return utils.ErrDatabaseInsert
	}

	user.ID = uint(id)
	r.logger.Info("用户创建成功", "userID", user.ID, "username", user.Username)
	return nil
}

// GetUserByUsername 根据用户名获取用户（使用prepared statement）
func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `SELECT id, username, password_hash, email, auth_status, account_status, 
			  last_login_time, last_login_ip, failed_login_count, created_at, updated_at 
			  FROM user_auth WHERE username = ?`

	user := &models.User{}
	err := r.db.QueryRowWithCache(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Email,
		&user.AuthStatus,
		&user.AccountStatus,
		&user.LastLoginTime,
		&user.LastLoginIP,
		&user.FailedLoginCount,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, utils.ErrUserNotFound
	}
	if err != nil {
		r.logger.Error("查询用户失败", "username", username, "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}

	return user, nil
}

// GetUserByEmail 根据邮箱获取用户（使用prepared statement）
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, username, password_hash, email, auth_status, account_status, 
			  last_login_time, last_login_ip, failed_login_count, created_at, updated_at 
			  FROM user_auth WHERE email = ?`

	user := &models.User{}
	err := r.db.QueryRowWithCache(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Email,
		&user.AuthStatus,
		&user.AccountStatus,
		&user.LastLoginTime,
		&user.LastLoginIP,
		&user.FailedLoginCount,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, utils.ErrUserNotFound
	}
	if err != nil {
		r.logger.Error("查询用户失败", "email", email, "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}

	return user, nil
}

// GetUserByID 根据ID获取用户（使用prepared statement）
func (r *UserRepository) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	query := `SELECT id, username, password_hash, email, auth_status, account_status, 
			  last_login_time, last_login_ip, failed_login_count, created_at, updated_at 
			  FROM user_auth WHERE id = ?`

	user := &models.User{}
	err := r.db.QueryRowWithCache(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Email,
		&user.AuthStatus,
		&user.AccountStatus,
		&user.LastLoginTime,
		&user.LastLoginIP,
		&user.FailedLoginCount,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, utils.ErrUserNotFound
	}
	if err != nil {
		r.logger.Error("查询用户失败", "userID", id, "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}

	return user, nil
}

// UpdateUser 更新用户信息
func (r *UserRepository) UpdateUser(ctx context.Context, user *models.User) error {
	query := `UPDATE user_auth SET email = ?, auth_status = ?, account_status = ?, 
			  last_login_time = ?, last_login_ip = ?, failed_login_count = ?, updated_at = ? 
			  WHERE id = ?`

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	result, err := r.db.DB.ExecContext(ctx, query,
		user.Email,
		user.AuthStatus,
		user.AccountStatus,
		user.LastLoginTime,
		user.LastLoginIP,
		user.FailedLoginCount,
		user.UpdatedAt,
		user.ID,
	)

	if err != nil {
		r.logger.Error("更新用户失败", "userID", user.ID, "error", err.Error())
		return utils.ErrDatabaseUpdate
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("获取更新行数失败", "userID", user.ID, "error", err.Error())
		return utils.ErrDatabaseUpdate
	}

	if rowsAffected == 0 {
		r.logger.Warn("用户不存在或未更新", "userID", user.ID)
		return utils.ErrUserNotFound
	}

	r.logger.Info("用户更新成功", "userID", user.ID)
	return nil
}

// GetUserProfile 读取扩展资料 user_profile
func (r *UserRepository) GetUserProfile(ctx context.Context, userID uint) (*models.UserExtraProfile, error) {
	query := `SELECT user_id, nickname, bio, avatar_url, created_at, updated_at FROM user_profile WHERE user_id = ?`

	prof := &models.UserExtraProfile{}
	err := r.db.DB.QueryRowContext(ctx, query, userID).Scan(
		&prof.UserID,
		&prof.Nickname,
		&prof.Bio,
		&prof.AvatarURL,
		&prof.CreatedAt,
		&prof.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// 未创建则返回空记录（不视为错误）
			return &models.UserExtraProfile{UserID: userID}, nil
		}
		r.logger.Error("查询用户扩展资料失败", "userID", userID, "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}

	return prof, nil
}

// UpsertUserProfile 创建或更新扩展资料（昵称/简介）
func (r *UserRepository) UpsertUserProfile(ctx context.Context, profile *models.UserExtraProfile) error {
	query := `INSERT INTO user_profile (user_id, nickname, bio, avatar_url, created_at, updated_at)
              VALUES (?, ?, ?, COALESCE(?, NULL), NOW(), NOW())
              ON DUPLICATE KEY UPDATE nickname = VALUES(nickname), bio = VALUES(bio), updated_at = NOW()`

	_, err := r.db.ExecWithCache(ctx, query, profile.UserID, profile.Nickname, profile.Bio, profile.AvatarURL)
	if err != nil {
		r.logger.Error("保存用户扩展资料失败", "userID", profile.UserID, "error", err.Error())
		return utils.ErrDatabaseUpdate
	}
	return nil
}

// UpdateUserAvatar 仅更新头像URL
func (r *UserRepository) UpdateUserAvatar(ctx context.Context, profile *models.UserExtraProfile) error {
	query := `INSERT INTO user_profile (user_id, avatar_url, created_at, updated_at)
              VALUES (?, ?, NOW(), NOW())
              ON DUPLICATE KEY UPDATE avatar_url = VALUES(avatar_url), updated_at = NOW()`

	_, err := r.db.ExecWithCache(ctx, query, profile.UserID, profile.AvatarURL)
	if err != nil {
		r.logger.Error("更新用户头像失败", "userID", profile.UserID, "error", err.Error())
		return utils.ErrDatabaseUpdate
	}
	r.logger.Info("更新用户头像成功", "userID", profile.UserID, "avatarUrl", profile.AvatarURL)
	return nil
}

// UpdateLoginInfo 更新登录信息
func (r *UserRepository) UpdateLoginInfo(ctx context.Context, userID uint, loginTime time.Time, loginIP string) error {
	query := `UPDATE user_auth SET last_login_time = ?, last_login_ip = ?, failed_login_count = 0, updated_at = ? WHERE id = ?`

	_, err := r.db.ExecWithCache(ctx, query, loginTime, loginIP, time.Now(), userID)
	if err != nil {
		r.logger.Error("更新登录信息失败", "userID", userID, "error", err.Error())
		return utils.ErrDatabaseUpdate
	}

	return nil
}

// IncrementFailedLoginCount 增加登录失败次数
func (r *UserRepository) IncrementFailedLoginCount(ctx context.Context, userID uint) error {
	query := `UPDATE user_auth SET failed_login_count = failed_login_count + 1, updated_at = ? WHERE id = ?`

	_, err := r.db.ExecWithCache(ctx, query, time.Now(), userID)
	if err != nil {
		r.logger.Error("更新登录失败次数失败", "userID", userID, "error", err.Error())
		return utils.ErrDatabaseUpdate
	}

	return nil
}

// CheckUsernameExists 检查用户名是否存在（使用prepared statement）
func (r *UserRepository) CheckUsernameExists(ctx context.Context, username string) (bool, error) {
	query := `SELECT 1 FROM user_auth WHERE username = ? LIMIT 1`

	var exists int
	err := r.db.QueryRowWithCache(ctx, query, username).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		r.logger.Error("检查用户名失败", "username", username, "error", err.Error())
		return false, utils.ErrDatabaseQuery
	}
	return true, nil
}

// CheckEmailExists 检查邮箱是否存在（使用prepared statement）
func (r *UserRepository) CheckEmailExists(ctx context.Context, email string) (bool, error) {
	query := `SELECT 1 FROM user_auth WHERE email = ? LIMIT 1`

	var exists int
	err := r.db.QueryRowWithCache(ctx, query, email).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		r.logger.Error("检查邮箱失败", "email", email, "error", err.Error())
		return false, utils.ErrDatabaseQuery
	}
	return true, nil
}
