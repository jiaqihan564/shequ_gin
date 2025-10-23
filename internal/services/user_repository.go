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
	start := time.Now()
	r.logger.Debug("开始创建用户",
		"username", user.Username,
		"email", utils.SanitizeEmail(user.Email),
		"authStatus", user.AuthStatus,
		"accountStatus", user.AccountStatus)

	query := `INSERT INTO user_auth (username, password_hash, email, auth_status, account_status, created_at, updated_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	r.logger.Debug("执行用户创建SQL",
		"operation", "INSERT",
		"table", "user_auth",
		"username", user.Username)

	// 使用缓存的prepared statement
	result, err := r.db.ExecWithCache(ctx, query,
		user.Username,
		user.PasswordHash,
		user.Email,
		user.AuthStatus,
		user.AccountStatus,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("创建用户失败",
			"username", user.Username,
			"email", utils.SanitizeEmail(user.Email),
			"error", err.Error(),
			"duration", time.Since(start))
		return utils.ErrDatabaseInsert
	}

	// 获取插入的ID
	id, err := result.LastInsertId()
	if err != nil {
		r.logger.Error("获取用户ID失败",
			"username", user.Username,
			"error", err.Error(),
			"duration", time.Since(start))
		return utils.ErrDatabaseInsert
	}

	user.ID = uint(id)
	r.logger.Info("用户创建成功",
		"userID", user.ID,
		"username", user.Username,
		"email", utils.SanitizeEmail(user.Email),
		"duration", time.Since(start),
		"durationMs", time.Since(start).Milliseconds())
	return nil
}

// GetUserByUsername 根据用户名获取用户
func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	start := time.Now()
	r.logger.Debug("开始查询用户(按用户名)",
		"username", username,
		"operation", "SELECT",
		"table", "user_auth")

	query := `SELECT id, username, password_hash, email, auth_status, account_status, 
			  last_login_time, last_login_ip, failed_login_count, created_at, updated_at 
			  FROM user_auth WHERE username = ?`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	user := &models.User{}
	// 使用缓存的prepared statement
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

	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Debug("用户不存在",
				"username", username,
				"duration", time.Since(start))
			return nil, utils.ErrUserNotFound
		}
		r.logger.Error("查询用户失败",
			"username", username,
			"error", err.Error(),
			"duration", time.Since(start))
		return nil, utils.ErrDatabaseQuery
	}

	r.logger.Debug("查询用户成功",
		"userID", user.ID,
		"username", username,
		"email", utils.SanitizeEmail(user.Email),
		"authStatus", user.AuthStatus,
		"accountStatus", user.AccountStatus,
		"failedLoginCount", user.FailedLoginCount,
		"duration", time.Since(start))

	return user, nil
}

// GetUserByEmail 根据邮箱获取用户
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, username, password_hash, email, auth_status, account_status, 
			  last_login_time, last_login_ip, failed_login_count, created_at, updated_at 
			  FROM user_auth WHERE email = ?`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	user := &models.User{}
	// 使用缓存的prepared statement
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

	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Debug("用户不存在", "email", email)
			return nil, utils.ErrUserNotFound
		}
		r.logger.Error("查询用户失败", "email", email, "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}

	return user, nil
}

// GetUserByID 根据ID获取用户
func (r *UserRepository) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	start := time.Now()
	r.logger.Debug("开始查询用户(按ID)",
		"userID", id,
		"operation", "SELECT",
		"table", "user_auth")

	query := `SELECT id, username, password_hash, email, auth_status, account_status, 
			  last_login_time, last_login_ip, failed_login_count, created_at, updated_at 
			  FROM user_auth WHERE id = ?`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	user := &models.User{}
	// 使用缓存的prepared statement
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

	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Debug("用户不存在",
				"userID", id,
				"duration", time.Since(start))
			return nil, utils.ErrUserNotFound
		}
		r.logger.Error("查询用户失败",
			"userID", id,
			"error", err.Error(),
			"duration", time.Since(start))
		return nil, utils.ErrDatabaseQuery
	}

	r.logger.Debug("查询用户成功",
		"userID", user.ID,
		"username", user.Username,
		"email", utils.SanitizeEmail(user.Email),
		"authStatus", user.AuthStatus,
		"accountStatus", user.AccountStatus,
		"duration", time.Since(start))

	return user, nil
}

// UpdateUser 更新用户信息
func (r *UserRepository) UpdateUser(ctx context.Context, user *models.User) error {
	query := `UPDATE user_auth SET email = ?, auth_status = ?, account_status = ?, 
			  last_login_time = ?, last_login_ip = ?, failed_login_count = ?, updated_at = ? 
			  WHERE id = ?`

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 使用缓存的prepared statement
	result, err := r.db.ExecWithCache(ctx, query,
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

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	prof := &models.UserExtraProfile{}

	// 使用 sql.NullString 处理可能为NULL的字段
	var nickname, bio, avatarURL sql.NullString

	// 使用缓存的prepared statement
	err := r.db.QueryRowWithCache(ctx, query, userID).Scan(
		&prof.UserID,
		&nickname,  // 使用 NullString
		&bio,       // 使用 NullString
		&avatarURL, // 使用 NullString
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

	// 安全地转换 NullString 为 string（NULL -> 空字符串）
	prof.Nickname = nickname.String
	prof.Bio = bio.String
	prof.AvatarURL = avatarURL.String

	return prof, nil
}

// UpsertUserProfile 创建或更新扩展资料（昵称/简介）
func (r *UserRepository) UpsertUserProfile(ctx context.Context, profile *models.UserExtraProfile) error {
	start := time.Now()
	r.logger.Debug("开始保存用户扩展资料",
		"userID", profile.UserID,
		"nickname", profile.Nickname,
		"bioLength", len(profile.Bio),
		"operation", "UPSERT",
		"table", "user_profile")

	query := `INSERT INTO user_profile (user_id, nickname, bio, avatar_url, created_at, updated_at)
              VALUES (?, ?, ?, COALESCE(?, NULL), NOW(), NOW())
              ON DUPLICATE KEY UPDATE nickname = VALUES(nickname), bio = VALUES(bio), updated_at = NOW()`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 使用缓存的prepared statement
	result, err := r.db.ExecWithCache(ctx, query, profile.UserID, profile.Nickname, profile.Bio, profile.AvatarURL)
	if err != nil {
		r.logger.Error("保存用户扩展资料失败",
			"userID", profile.UserID,
			"nickname", profile.Nickname,
			"error", err.Error(),
			"duration", time.Since(start))
		return utils.ErrDatabaseUpdate
	}

	rowsAffected, _ := result.RowsAffected()
	r.logger.Info("保存用户扩展资料成功",
		"userID", profile.UserID,
		"nickname", profile.Nickname,
		"bioLength", len(profile.Bio),
		"rowsAffected", rowsAffected,
		"duration", time.Since(start))
	return nil
}

// UpdateUserAvatar 仅更新头像URL
func (r *UserRepository) UpdateUserAvatar(ctx context.Context, profile *models.UserExtraProfile) error {
	query := `INSERT INTO user_profile (user_id, avatar_url, created_at, updated_at)
              VALUES (?, ?, NOW(), NOW())
              ON DUPLICATE KEY UPDATE avatar_url = VALUES(avatar_url), updated_at = NOW()`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 使用缓存的prepared statement
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
	start := time.Now()
	r.logger.Debug("开始更新登录信息",
		"userID", userID,
		"loginIP", loginIP,
		"loginTime", loginTime,
		"operation", "UPDATE",
		"table", "user_auth")

	query := `UPDATE user_auth SET last_login_time = ?, last_login_ip = ?, failed_login_count = 0, updated_at = ? WHERE id = ?`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 使用缓存的prepared statement
	result, err := r.db.ExecWithCache(ctx, query, loginTime, loginIP, time.Now(), userID)
	if err != nil {
		r.logger.Error("更新登录信息失败",
			"userID", userID,
			"loginIP", loginIP,
			"error", err.Error(),
			"duration", time.Since(start))
		return utils.ErrDatabaseUpdate
	}

	rowsAffected, _ := result.RowsAffected()
	r.logger.Info("更新登录信息成功",
		"userID", userID,
		"loginIP", loginIP,
		"rowsAffected", rowsAffected,
		"duration", time.Since(start))

	return nil
}

// IncrementFailedLoginCount 增加登录失败次数
func (r *UserRepository) IncrementFailedLoginCount(ctx context.Context, userID uint) error {
	query := `UPDATE user_auth SET failed_login_count = failed_login_count + 1, updated_at = ? WHERE id = ?`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 使用缓存的prepared statement
	_, err := r.db.ExecWithCache(ctx, query, time.Now(), userID)
	if err != nil {
		r.logger.Error("更新登录失败次数失败", "userID", userID, "error", err.Error())
		return utils.ErrDatabaseUpdate
	}

	return nil
}

// CheckUsernameExists 检查用户名是否存在
func (r *UserRepository) CheckUsernameExists(ctx context.Context, username string) (bool, error) {
	start := time.Now()
	r.logger.Debug("开始检查用户名是否存在",
		"username", username,
		"operation", "SELECT COUNT",
		"table", "user_auth")

	query := `SELECT COUNT(*) FROM user_auth WHERE username = ?`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var count int
	// 使用缓存的prepared statement
	err := r.db.QueryRowWithCache(ctx, query, username).Scan(&count)
	if err != nil {
		r.logger.Error("检查用户名失败",
			"username", username,
			"error", err.Error(),
			"duration", time.Since(start))
		return false, utils.ErrDatabaseQuery
	}

	exists := count > 0
	r.logger.Debug("检查用户名完成",
		"username", username,
		"exists", exists,
		"count", count,
		"duration", time.Since(start))

	return exists, nil
}

// CheckEmailExists 检查邮箱是否存在
func (r *UserRepository) CheckEmailExists(ctx context.Context, email string) (bool, error) {
	start := time.Now()
	r.logger.Debug("开始检查邮箱是否存在",
		"email", utils.SanitizeEmail(email),
		"operation", "SELECT COUNT",
		"table", "user_auth")

	query := `SELECT COUNT(*) FROM user_auth WHERE email = ?`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var count int
	// 使用缓存的prepared statement
	err := r.db.QueryRowWithCache(ctx, query, email).Scan(&count)
	if err != nil {
		r.logger.Error("检查邮箱失败",
			"email", utils.SanitizeEmail(email),
			"error", err.Error(),
			"duration", time.Since(start))
		return false, utils.ErrDatabaseQuery
	}

	exists := count > 0
	r.logger.Debug("检查邮箱完成",
		"email", utils.SanitizeEmail(email),
		"exists", exists,
		"count", count,
		"duration", time.Since(start))

	return exists, nil
}

// UpdatePassword 更新用户密码
func (r *UserRepository) UpdatePassword(ctx context.Context, userID uint, newPasswordHash string) error {
	start := time.Now()
	r.logger.Debug("开始更新用户密码",
		"userID", userID,
		"operation", "UPDATE",
		"table", "user_auth")

	query := `UPDATE user_auth SET password_hash = ?, updated_at = ? WHERE id = ?`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 使用缓存的prepared statement
	result, err := r.db.ExecWithCache(ctx, query, newPasswordHash, time.Now(), userID)
	if err != nil {
		r.logger.Error("更新密码失败",
			"userID", userID,
			"error", err.Error(),
			"duration", time.Since(start))
		return utils.ErrDatabaseUpdate
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		r.logger.Warn("用户不存在或密码未更新",
			"userID", userID,
			"duration", time.Since(start))
		return utils.ErrUserNotFound
	}

	r.logger.Info("更新密码成功",
		"userID", userID,
		"rowsAffected", rowsAffected,
		"duration", time.Since(start))

	return nil
}

// CreatePasswordResetToken 创建密码重置token
func (r *UserRepository) CreatePasswordResetToken(ctx context.Context, token *models.PasswordResetToken) error {
	start := time.Now()
	r.logger.Debug("开始创建密码重置token",
		"email", utils.SanitizeEmail(token.Email),
		"expiresAt", token.ExpiresAt,
		"operation", "INSERT",
		"table", "password_reset_tokens")

	query := `INSERT INTO password_reset_tokens (email, token, expires_at, used, created_at) 
			  VALUES (?, ?, ?, ?, ?)`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 使用缓存的prepared statement
	result, err := r.db.ExecWithCache(ctx, query,
		token.Email,
		token.Token,
		token.ExpiresAt,
		token.Used,
		token.CreatedAt,
	)

	if err != nil {
		r.logger.Error("创建密码重置token失败",
			"email", utils.SanitizeEmail(token.Email),
			"error", err.Error(),
			"duration", time.Since(start))
		return utils.ErrDatabaseInsert
	}

	id, _ := result.LastInsertId()
	token.ID = uint(id)

	r.logger.Info("创建密码重置token成功",
		"tokenID", token.ID,
		"email", utils.SanitizeEmail(token.Email),
		"duration", time.Since(start))

	return nil
}

// GetPasswordResetToken 根据token获取密码重置记录
func (r *UserRepository) GetPasswordResetToken(ctx context.Context, token string) (*models.PasswordResetToken, error) {
	start := time.Now()
	r.logger.Debug("开始查询密码重置token",
		"operation", "SELECT",
		"table", "password_reset_tokens")

	query := `SELECT id, email, token, expires_at, used, created_at 
			  FROM password_reset_tokens 
			  WHERE token = ? AND used = false`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resetToken := &models.PasswordResetToken{}
	// 使用缓存的prepared statement
	err := r.db.QueryRowWithCache(ctx, query, token).Scan(
		&resetToken.ID,
		&resetToken.Email,
		&resetToken.Token,
		&resetToken.ExpiresAt,
		&resetToken.Used,
		&resetToken.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Debug("密码重置token不存在或已使用",
				"duration", time.Since(start))
			return nil, utils.ErrInvalidToken
		}
		r.logger.Error("查询密码重置token失败",
			"error", err.Error(),
			"duration", time.Since(start))
		return nil, utils.ErrDatabaseQuery
	}

	r.logger.Debug("查询密码重置token成功",
		"tokenID", resetToken.ID,
		"email", utils.SanitizeEmail(resetToken.Email),
		"duration", time.Since(start))

	return resetToken, nil
}

// MarkPasswordResetTokenAsUsed 标记token为已使用
func (r *UserRepository) MarkPasswordResetTokenAsUsed(ctx context.Context, tokenID uint) error {
	start := time.Now()
	r.logger.Debug("开始标记密码重置token为已使用",
		"tokenID", tokenID,
		"operation", "UPDATE",
		"table", "password_reset_tokens")

	query := `UPDATE password_reset_tokens SET used = true WHERE id = ?`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 使用缓存的prepared statement
	_, err := r.db.ExecWithCache(ctx, query, tokenID)
	if err != nil {
		r.logger.Error("标记密码重置token失败",
			"tokenID", tokenID,
			"error", err.Error(),
			"duration", time.Since(start))
		return utils.ErrDatabaseUpdate
	}

	r.logger.Info("标记密码重置token成功",
		"tokenID", tokenID,
		"duration", time.Since(start))

	return nil
}
