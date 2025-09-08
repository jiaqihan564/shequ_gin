package services

import (
	"database/sql"
	"fmt"
	"time"

	"gin/internal/models"
)

// UserRepository 用户数据访问层
type UserRepository struct {
	db *Database
}

// NewUserRepository 创建用户数据访问层
func NewUserRepository(db *Database) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser 创建用户
func (r *UserRepository) CreateUser(user *models.User) error {
	query := `INSERT INTO user_auth (username, password_hash, email, auth_status, account_status, created_at, updated_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.DB.Exec(query,
		user.Username,
		user.PasswordHash,
		user.Email,
		user.AuthStatus,
		user.AccountStatus,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("创建用户失败: %v", err)
	}

	return nil
}

// GetUserByUsername 根据用户名获取用户
func (r *UserRepository) GetUserByUsername(username string) (*models.User, error) {
	query := `SELECT id, username, password_hash, email, auth_status, account_status, 
			  last_login_time, last_login_ip, failed_login_count, created_at, updated_at 
			  FROM user_auth WHERE username = ?`

	user := &models.User{}
	err := r.db.DB.QueryRow(query, username).Scan(
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

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("查询用户失败: %v", err)
	}

	return user, nil
}

// GetUserByEmail 根据邮箱获取用户
func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	query := `SELECT id, username, password_hash, email, auth_status, account_status, 
			  last_login_time, last_login_ip, failed_login_count, created_at, updated_at 
			  FROM user_auth WHERE email = ?`

	user := &models.User{}
	err := r.db.DB.QueryRow(query, email).Scan(
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

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("查询用户失败: %v", err)
	}

	return user, nil
}

// GetUserByID 根据ID获取用户
func (r *UserRepository) GetUserByID(id uint) (*models.User, error) {
	query := `SELECT id, username, password_hash, email, auth_status, account_status, 
			  last_login_time, last_login_ip, failed_login_count, created_at, updated_at 
			  FROM user_auth WHERE id = ?`

	user := &models.User{}
	err := r.db.DB.QueryRow(query, id).Scan(
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

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("查询用户失败: %v", err)
	}

	return user, nil
}

// UpdateUser 更新用户信息
func (r *UserRepository) UpdateUser(user *models.User) error {
	query := `UPDATE user_auth SET email = ?, auth_status = ?, account_status = ?, 
			  last_login_time = ?, last_login_ip = ?, failed_login_count = ?, updated_at = ? 
			  WHERE id = ?`

	_, err := r.db.DB.Exec(query,
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
		return fmt.Errorf("更新用户失败: %v", err)
	}

	return nil
}

// UpdateLoginInfo 更新登录信息
func (r *UserRepository) UpdateLoginInfo(userID uint, loginTime time.Time, loginIP string) error {
	query := `UPDATE user_auth SET last_login_time = ?, last_login_ip = ?, failed_login_count = 0, updated_at = ? WHERE id = ?`

	_, err := r.db.DB.Exec(query, loginTime, loginIP, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("更新登录信息失败: %v", err)
	}

	return nil
}

// IncrementFailedLoginCount 增加登录失败次数
func (r *UserRepository) IncrementFailedLoginCount(userID uint) error {
	query := `UPDATE user_auth SET failed_login_count = failed_login_count + 1, updated_at = ? WHERE id = ?`

	_, err := r.db.DB.Exec(query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("更新登录失败次数失败: %v", err)
	}

	return nil
}

// CheckUsernameExists 检查用户名是否存在
func (r *UserRepository) CheckUsernameExists(username string) (bool, error) {
	query := `SELECT COUNT(*) FROM user_auth WHERE username = ?`

	var count int
	err := r.db.DB.QueryRow(query, username).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查用户名失败: %v", err)
	}

	return count > 0, nil
}

// CheckEmailExists 检查邮箱是否存在
func (r *UserRepository) CheckEmailExists(email string) (bool, error) {
	query := `SELECT COUNT(*) FROM user_auth WHERE email = ?`

	var count int
	err := r.db.DB.QueryRow(query, email).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查邮箱失败: %v", err)
	}

	return count > 0, nil
}
