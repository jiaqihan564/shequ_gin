package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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

	query := `INSERT INTO user_auth (username, password_hash, email, auth_status, account_status, created_at, updated_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	ctx, cancel := context.WithTimeout(ctx, r.db.GetUpdateTimeout())
	defer cancel()

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
			"error", err.Error())
		return utils.ErrDatabaseInsert
	}

	id, err := result.LastInsertId()
	if err != nil {
		r.logger.Error("获取用户ID失败", "username", user.Username, "error", err.Error())
		return utils.ErrDatabaseInsert
	}

	user.ID = uint(id)
	r.logger.Info("用户创建成功",
		"userID", user.ID,
		"username", user.Username,
		"duration", time.Since(start))
	return nil
}

// GetUserByUsername 根据用户名获取用户
func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `SELECT id, username, password_hash, email, auth_status, account_status, 
			  last_login_time, last_login_ip, failed_login_count, created_at, updated_at 
			  FROM user_auth WHERE username = ?`

	ctx, cancel := context.WithTimeout(ctx, r.db.GetQueryTimeout())
	defer cancel()

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

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, utils.ErrUserNotFound
		}
		r.logger.Error("查询用户失败", "username", username, "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}

	return user, nil
}

// GetUserByEmail 根据邮箱获取用户
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, username, password_hash, email, auth_status, account_status, 
			  last_login_time, last_login_ip, failed_login_count, created_at, updated_at 
			  FROM user_auth WHERE email = ?`

	ctx, cancel := context.WithTimeout(ctx, r.db.GetQueryTimeout())
	defer cancel()

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

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, utils.ErrUserNotFound
		}
		r.logger.Error("查询用户失败", "email", email, "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}

	return user, nil
}

// GetUserByID 根据ID获取用户
func (r *UserRepository) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	query := `SELECT id, username, password_hash, email, auth_status, account_status, 
			  last_login_time, last_login_ip, failed_login_count, created_at, updated_at 
			  FROM user_auth WHERE id = ?`

	ctx, cancel := context.WithTimeout(ctx, r.db.GetQueryTimeout())
	defer cancel()

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

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, utils.ErrUserNotFound
		}
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

	ctx, cancel := context.WithTimeout(ctx, r.db.GetUpdateTimeout())
	defer cancel()

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

// GetUserProfile 读取扩展资料
func (r *UserRepository) GetUserProfile(ctx context.Context, userID uint) (*models.UserExtraProfile, error) {
	query := `SELECT user_id, nickname, bio, avatar_url, created_at, updated_at FROM user_profile WHERE user_id = ?`

	ctx, cancel := context.WithTimeout(ctx, r.db.GetQueryTimeout())
	defer cancel()

	prof := &models.UserExtraProfile{}
	var nickname, bio, avatarURL sql.NullString

	err := r.db.QueryRowWithCache(ctx, query, userID).Scan(
		&prof.UserID,
		&nickname,
		&bio,
		&avatarURL,
		&prof.CreatedAt,
		&prof.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return &models.UserExtraProfile{UserID: userID}, nil
		}
		r.logger.Error("查询用户扩展资料失败", "userID", userID, "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}

	prof.Nickname = nickname.String
	prof.Bio = bio.String
	prof.AvatarURL = avatarURL.String

	return prof, nil
}

// UpsertUserProfile 创建或更新扩展资料（昵称/简介）
func (r *UserRepository) UpsertUserProfile(ctx context.Context, profile *models.UserExtraProfile) error {
	start := time.Now()

	query := `INSERT INTO user_profile (user_id, nickname, bio, avatar_url, created_at, updated_at)
              VALUES (?, ?, ?, COALESCE(?, NULL), NOW(), NOW())
              ON DUPLICATE KEY UPDATE nickname = VALUES(nickname), bio = VALUES(bio), updated_at = NOW()`

	ctx, cancel := context.WithTimeout(ctx, r.db.GetUpdateTimeout())
	defer cancel()

	result, err := r.db.ExecWithCache(ctx, query, profile.UserID, profile.Nickname, profile.Bio, profile.AvatarURL)
	if err != nil {
		r.logger.Error("保存用户扩展资料失败", "userID", profile.UserID, "error", err.Error())
		return utils.ErrDatabaseUpdate
	}

	rowsAffected, _ := result.RowsAffected()
	r.logger.Info("保存用户扩展资料成功",
		"userID", profile.UserID,
		"nickname", profile.Nickname,
		"rowsAffected", rowsAffected,
		"duration", time.Since(start))
	return nil
}

// UpdateUserAvatar 仅更新头像URL
func (r *UserRepository) UpdateUserAvatar(ctx context.Context, profile *models.UserExtraProfile) error {
	query := `INSERT INTO user_profile (user_id, avatar_url, created_at, updated_at)
              VALUES (?, ?, NOW(), NOW())
              ON DUPLICATE KEY UPDATE avatar_url = VALUES(avatar_url), updated_at = NOW()`

	ctx, cancel := context.WithTimeout(ctx, r.db.GetUpdateTimeout())
	defer cancel()

	_, err := r.db.ExecWithCache(ctx, query, profile.UserID, profile.AvatarURL)
	if err != nil {
		r.logger.Error("更新用户头像失败", "userID", profile.UserID, "error", err.Error())
		return utils.ErrDatabaseUpdate
	}
	r.logger.Info("更新用户头像成功", "userID", profile.UserID)
	return nil
}

// UpdateLoginInfo 更新登录信息
func (r *UserRepository) UpdateLoginInfo(ctx context.Context, userID uint, loginTime time.Time, loginIP string) error {
	query := `UPDATE user_auth SET last_login_time = ?, last_login_ip = ?, failed_login_count = 0, updated_at = ? WHERE id = ?`

	ctx, cancel := context.WithTimeout(ctx, r.db.GetUpdateTimeout())
	defer cancel()

	result, err := r.db.ExecWithCache(ctx, query, loginTime, loginIP, time.Now(), userID)
	if err != nil {
		r.logger.Error("更新登录信息失败", "userID", userID, "error", err.Error())
		return utils.ErrDatabaseUpdate
	}

	rowsAffected, _ := result.RowsAffected()
	r.logger.Info("更新登录信息成功", "userID", userID, "loginIP", loginIP, "rowsAffected", rowsAffected)
	return nil
}

// IncrementFailedLoginCount 增加登录失败次数
func (r *UserRepository) IncrementFailedLoginCount(ctx context.Context, userID uint) error {
	query := `UPDATE user_auth SET failed_login_count = failed_login_count + 1, updated_at = ? WHERE id = ?`

	ctx, cancel := context.WithTimeout(ctx, r.db.GetUpdateTimeout())
	defer cancel()

	_, err := r.db.ExecWithCache(ctx, query, time.Now(), userID)
	if err != nil {
		r.logger.Error("更新登录失败次数失败", "userID", userID, "error", err.Error())
		return utils.ErrDatabaseUpdate
	}

	return nil
}

// CheckUsernameExists 检查用户名是否存在
func (r *UserRepository) CheckUsernameExists(ctx context.Context, username string) (bool, error) {
	query := `SELECT COUNT(*) FROM user_auth WHERE username = ?`

	ctx, cancel := context.WithTimeout(ctx, r.db.GetQueryTimeout())
	defer cancel()

	var count int
	err := r.db.QueryRowWithCache(ctx, query, username).Scan(&count)
	if err != nil {
		r.logger.Error("检查用户名失败", "username", username, "error", err.Error())
		return false, utils.ErrDatabaseQuery
	}

	return count > 0, nil
}

// BatchGetUserProfiles 批量获取用户信息（解决N+1问题）
func (r *UserRepository) BatchGetUserProfiles(ctx context.Context, userIDs []uint) (map[uint]*models.User, error) {
	if len(userIDs) == 0 {
		return make(map[uint]*models.User), nil
	}

	// 去重（预分配容量）
	uniqueIDs := make(map[uint]bool, len(userIDs))
	for _, id := range userIDs {
		uniqueIDs[id] = true
	}

	ids := make([]uint, 0, len(uniqueIDs))
	for id := range uniqueIDs {
		ids = append(ids, id)
	}

	// 构建批量查询（使用JOIN一次性获取用户和profile）
	// 优化：使用strings.Repeat代替循环构建placeholders
	placeholders := "?" + strings.Repeat(",?", len(ids)-1)
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT ua.id, ua.username, ua.email, ua.auth_status, ua.account_status,
		       COALESCE(up.nickname, ua.username) as nickname,
		       COALESCE(up.avatar_url, '') as avatar
		FROM user_auth ua
		LEFT JOIN user_profile up ON ua.id = up.user_id
		WHERE ua.id IN (%s)
	`, placeholders)

	ctx, cancel := context.WithTimeout(ctx, r.db.GetQueryTimeout())
	defer cancel()

	rows, err := r.db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("批量查询用户信息失败", "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}
	defer rows.Close()

	users := make(map[uint]*models.User, len(ids))
	for rows.Next() {
		var user models.User
		var nickname, avatar string
		err := rows.Scan(
			&user.ID, &user.Username, &user.Email,
			&user.AuthStatus, &user.AccountStatus,
			&nickname, &avatar)
		if err != nil {
			r.logger.Warn("扫描用户信息失败", "error", err.Error())
			continue
		}

		// 将nickname和avatar附加到用户对象（虽然User模型没有这些字段）
		// 调用者需要单独处理
		users[user.ID] = &user
	}

	r.logger.Info("批量查询用户信息成功", "count", len(users), "requested", len(ids))
	return users, nil
}

// CheckEmailExists 检查邮箱是否存在
func (r *UserRepository) CheckEmailExists(ctx context.Context, email string) (bool, error) {
	query := `SELECT COUNT(*) FROM user_auth WHERE email = ?`

	ctx, cancel := context.WithTimeout(ctx, r.db.GetQueryTimeout())
	defer cancel()

	var count int
	err := r.db.QueryRowWithCache(ctx, query, email).Scan(&count)
	if err != nil {
		r.logger.Error("检查邮箱失败", "email", utils.SanitizeEmail(email), "error", err.Error())
		return false, utils.ErrDatabaseQuery
	}

	return count > 0, nil
}

// UpdatePassword 更新用户密码
func (r *UserRepository) UpdatePassword(ctx context.Context, userID uint, newPasswordHash string) error {
	start := time.Now()

	query := `UPDATE user_auth SET password_hash = ?, updated_at = ? WHERE id = ?`

	ctx, cancel := context.WithTimeout(ctx, r.db.GetUpdateTimeout())
	defer cancel()

	result, err := r.db.ExecWithCache(ctx, query, newPasswordHash, time.Now(), userID)
	if err != nil {
		r.logger.Error("更新密码失败", "userID", userID, "error", err.Error())
		return utils.ErrDatabaseUpdate
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		r.logger.Warn("用户不存在或密码未更新", "userID", userID)
		return utils.ErrUserNotFound
	}

	r.logger.Info("更新密码成功", "userID", userID, "duration", time.Since(start))
	return nil
}

// CreatePasswordResetToken 创建密码重置token
func (r *UserRepository) CreatePasswordResetToken(ctx context.Context, token *models.PasswordResetToken) error {
	query := `INSERT INTO password_reset_tokens (email, token, expires_at, used, created_at) 
			  VALUES (?, ?, ?, ?, ?)`

	ctx, cancel := context.WithTimeout(ctx, r.db.GetUpdateTimeout())
	defer cancel()

	result, err := r.db.ExecWithCache(ctx, query,
		token.Email,
		token.Token,
		token.ExpiresAt,
		token.Used,
		token.CreatedAt,
	)

	if err != nil {
		r.logger.Error("创建密码重置token失败", "email", utils.SanitizeEmail(token.Email), "error", err.Error())
		return utils.ErrDatabaseInsert
	}

	id, _ := result.LastInsertId()
	token.ID = uint(id)

	r.logger.Info("创建密码重置token成功", "tokenID", token.ID, "email", utils.SanitizeEmail(token.Email))
	return nil
}

// GetPasswordResetToken 根据token获取密码重置记录
func (r *UserRepository) GetPasswordResetToken(ctx context.Context, token string) (*models.PasswordResetToken, error) {
	query := `SELECT id, email, token, expires_at, used, created_at 
			  FROM password_reset_tokens 
			  WHERE token = ? AND used = false`

	ctx, cancel := context.WithTimeout(ctx, r.db.GetQueryTimeout())
	defer cancel()

	resetToken := &models.PasswordResetToken{}
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
			return nil, utils.ErrInvalidToken
		}
		r.logger.Error("查询密码重置token失败", "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}

	return resetToken, nil
}

// MarkPasswordResetTokenAsUsed 标记token为已使用
func (r *UserRepository) MarkPasswordResetTokenAsUsed(ctx context.Context, tokenID uint) error {
	query := `UPDATE password_reset_tokens SET used = true WHERE id = ?`

	ctx, cancel := context.WithTimeout(ctx, r.db.GetUpdateTimeout())
	defer cancel()

	_, err := r.db.ExecWithCache(ctx, query, tokenID)
	if err != nil {
		r.logger.Error("标记密码重置token失败", "tokenID", tokenID, "error", err.Error())
		return utils.ErrDatabaseUpdate
	}

	r.logger.Info("标记密码重置token成功", "tokenID", tokenID)
	return nil
}
