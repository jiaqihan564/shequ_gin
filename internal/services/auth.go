package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"gin/internal/config"
	"gin/internal/models"
	"gin/internal/utils"

	"github.com/golang-jwt/jwt/v4"
)

// AuthService 认证服务
type AuthService struct {
	config      *config.Config
	userRepo    *UserRepository
	historyRepo *HistoryRepository
	logger      utils.Logger
}

// NewAuthService 创建认证服务
func NewAuthService(cfg *config.Config, userRepo *UserRepository, historyRepo *HistoryRepository) *AuthService {
	return &AuthService{
		config:      cfg,
		userRepo:    userRepo,
		historyRepo: historyRepo,
		logger:      utils.GetLogger(),
	}
}

// Login 用户登录
func (s *AuthService) Login(ctx context.Context, username, password, clientIP, province, city string) (*models.LoginResponse, error) {
	startTime := time.Now()

	// 获取用户信息
	user, err := s.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		s.logger.Warn("登录失败：用户不存在", "username", username, "ip", clientIP)
		return nil, utils.ErrInvalidCredentials
	}

	// 检查账户状态
	if user.AccountStatus != 1 {
		s.logger.Warn("登录失败：账户已被禁用",
			"userID", user.ID,
			"username", username,
			"ip", clientIP)
		return nil, utils.ErrAccountDisabled
	}

	// 检查登录失败次数
	if user.FailedLoginCount >= s.config.Security.MaxLoginAttempts {
		s.logger.Warn("登录失败：登录尝试次数过多",
			"userID", user.ID,
			"username", username,
			"failedCount", user.FailedLoginCount,
			"ip", clientIP)
		return nil, utils.ErrTooManyLoginAttempts
	}

	// 验证密码
	passwordValid := utils.CheckPasswordHash(password, user.PasswordHash)
	if !passwordValid {
		// 增加登录失败次数
		if err := s.userRepo.IncrementFailedLoginCount(ctx, user.ID); err != nil {
			s.logger.Error("更新登录失败次数失败", "userID", user.ID, "error", err.Error())
		}
		s.logger.Warn("登录失败：密码错误",
			"userID", user.ID,
			"username", username,
			"ip", clientIP)
		return nil, utils.ErrInvalidCredentials
	}

	// 更新登录信息
	now := time.Now()
	err = s.userRepo.UpdateLoginInfo(ctx, user.ID, now, clientIP)
	if err != nil {
		s.logger.Error("更新登录信息失败", "userID", user.ID, "error", err.Error())
	}

	// 生成JWT token
	token, err := s.generateJWT(user.ID, user.Username)
	if err != nil {
		s.logger.Error("生成JWT token失败", "userID", user.ID, "error", err.Error())
		return nil, utils.ErrInternalServerError
	}

	// 读取扩展资料
	extra, _ := s.userRepo.GetUserProfile(ctx, user.ID)

	// 检查用户是否为管理员（优化：使用AdminChecker，O(1)查找）
	role := utils.GetUserRole(s.config, user.Username)

	// 安全获取扩展资料字段
	avatarURL, nickname, bio := "", "", ""
	if extra != nil {
		avatarURL = extra.AvatarURL
		nickname = extra.Nickname
		bio = extra.Bio
	}

	// 返回登录成功响应
	response := &models.LoginResponse{
		Code:    200,
		Message: "登录成功",
		Data: struct {
			Token string             `json:"token"`
			User  models.UserProfile `json:"user"`
		}{
			Token: token,
			User: models.UserProfile{
				ID:            user.ID,
				Username:      user.Username,
				Email:         user.Email,
				AuthStatus:    user.AuthStatus,
				AccountStatus: user.AccountStatus,
				AvatarURL:     avatarURL,
				Nickname:      nickname,
				Bio:           bio,
				Role:          role,
			},
		},
	}

	s.logger.Info("用户登录成功",
		"userID", user.ID,
		"username", username,
		"ip", clientIP,
		"duration", time.Since(startTime))

	// 异步记录登录历史
	if s.historyRepo != nil {
		userID := user.ID
		userName := username
		userIP := clientIP
		prov := province
		ct := city

		err := utils.SubmitTask(
			fmt.Sprintf("login-history-%d-%d", userID, time.Now().Unix()),
			func(ctx context.Context) error {
				userAgentStr := ""
				if err := s.historyRepo.RecordLoginHistory(userID, userName, userIP, userAgentStr, prov, ct, 1); err != nil {
					s.logger.Error("记录登录历史失败", "userID", userID, "error", err.Error())
					return err
				}
				if err := s.historyRepo.RecordOperationHistory(userID, userName, "登录", "用户登录系统", userIP); err != nil {
					s.logger.Error("记录操作历史失败", "userID", userID, "error", err.Error())
					return err
				}
				return nil
			},
			10*time.Second,
		)
		if err != nil {
			s.logger.Warn("提交登录历史记录任务失败", "error", err.Error())
		}
	}

	return response, nil
}

// Register 用户注册
func (s *AuthService) Register(ctx context.Context, username, password, email, clientIP, userAgent, province, city string) (*models.LoginResponse, error) {
	startTime := time.Now()

	// 检查用户名是否已存在
	usernameExists, err := s.userRepo.CheckUsernameExists(ctx, username)
	if err != nil {
		s.logger.Error("检查用户名失败", "username", username, "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}

	if usernameExists {
		s.logger.Warn("注册失败：用户名已存在", "username", username)
		return nil, utils.ErrUserAlreadyExists
	}

	// 检查邮箱是否已存在
	emailExists, err := s.userRepo.CheckEmailExists(ctx, email)
	if err != nil {
		s.logger.Error("检查邮箱失败", "email", utils.SanitizeEmail(email), "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}

	if emailExists {
		s.logger.Warn("注册失败：邮箱已被注册", "email", utils.SanitizeEmail(email))
		return nil, utils.ErrEmailAlreadyExists
	}

	// 加密密码
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		s.logger.Error("密码加密失败", "username", username, "error", err.Error())
		return nil, utils.ErrInternalServerError
	}

	// 创建新用户
	now := time.Now()
	user := &models.User{
		Username:      username,
		PasswordHash:  hashedPassword,
		Email:         email,
		AuthStatus:    1,
		AccountStatus: 1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// 保存用户到数据库
	err = s.userRepo.CreateUser(ctx, user)
	if err != nil {
		s.logger.Error("创建用户失败", "username", username, "error", err.Error())
		return nil, utils.ErrDatabaseInsert
	}

	// 重新获取用户信息以获取生成的ID
	user, err = s.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		s.logger.Error("获取用户信息失败", "username", username, "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}

	// 生成JWT token
	token, err := s.generateJWT(user.ID, user.Username)
	if err != nil {
		s.logger.Error("生成JWT token失败", "userID", user.ID, "error", err.Error())
		return nil, utils.ErrInternalServerError
	}

	// 读取扩展资料
	extra, _ := s.userRepo.GetUserProfile(ctx, user.ID)

	// 检查用户是否为管理员（优化：使用AdminChecker，O(1)查找）
	role := utils.GetUserRole(s.config, user.Username)

	// 安全获取扩展资料字段
	regAvatarURL, regNickname, regBio := "", "", ""
	if extra != nil {
		regAvatarURL = extra.AvatarURL
		regNickname = extra.Nickname
		regBio = extra.Bio
	}

	// 返回注册成功响应
	response := &models.LoginResponse{
		Code:    201,
		Message: "注册成功",
		Data: struct {
			Token string             `json:"token"`
			User  models.UserProfile `json:"user"`
		}{
			Token: token,
			User: models.UserProfile{
				ID:            user.ID,
				Username:      user.Username,
				Email:         user.Email,
				AuthStatus:    user.AuthStatus,
				AccountStatus: user.AccountStatus,
				AvatarURL:     regAvatarURL,
				Nickname:      regNickname,
				Bio:           regBio,
				Role:          role,
			},
		},
	}

	s.logger.Info("用户注册成功",
		"userID", user.ID,
		"username", username,
		"email", utils.SanitizeEmail(email),
		"duration", time.Since(startTime))

	// 异步记录注册和登录历史
	if s.historyRepo != nil {
		userID := user.ID
		userName := username
		userIP := clientIP
		userAgentStr := userAgent
		prov := province
		ct := city

		err := utils.SubmitTask(
			fmt.Sprintf("register-history-%d-%d", userID, time.Now().Unix()),
			func(ctx context.Context) error {
				if err := s.historyRepo.RecordOperationHistory(userID, userName, "注册", "用户注册账号", userIP); err != nil {
					s.logger.Error("记录操作历史失败", "userID", userID, "error", err.Error())
					return err
				}
				if err := s.historyRepo.RecordLoginHistory(userID, userName, userIP, userAgentStr, prov, ct, 1); err != nil {
					s.logger.Error("记录登录历史失败", "userID", userID, "error", err.Error())
					return err
				}
				return nil
			},
			10*time.Second,
		)
		if err != nil {
			s.logger.Warn("提交注册历史记录任务失败", "error", err.Error())
		}
	}

	return response, nil
}

// generateJWT 生成JWT token
func (s *AuthService) generateJWT(userID uint, username string) (string, error) {
	claims := models.CreateClaims(userID, username, s.config.JWT.Issuer, s.config.JWT.ExpireHours)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(s.config.JWT.SecretKey))
	if err != nil {
		s.logger.Error("token签名失败", "userID", userID, "error", err.Error())
		return "", err
	}
	return signedToken, nil
}

// ChangePassword 修改密码
func (s *AuthService) ChangePassword(ctx context.Context, userID uint, currentPassword, newPassword string) error {
	startTime := time.Now()

	// 获取用户信息
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.Warn("修改密码失败：用户不存在", "userID", userID)
		return utils.ErrUserNotFound
	}

	// 验证当前密码
	passwordValid := utils.CheckPasswordHash(currentPassword, user.PasswordHash)
	if !passwordValid {
		s.logger.Warn("修改密码失败：当前密码错误", "userID", userID)
		return utils.ErrInvalidCredentials
	}

	// 验证新密码强度
	if !utils.ValidatePassword(newPassword) {
		s.logger.Warn("修改密码失败：新密码强度不够", "userID", userID)
		return utils.ErrInvalidPassword
	}

	// 加密新密码
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		s.logger.Error("新密码加密失败", "userID", userID, "error", err.Error())
		return utils.ErrInternalServerError
	}

	// 更新密码到数据库
	err = s.userRepo.UpdatePassword(ctx, userID, hashedPassword)
	if err != nil {
		s.logger.Error("更新密码失败", "userID", userID, "error", err.Error())
		return err
	}

	s.logger.Info("密码修改成功", "userID", userID, "duration", time.Since(startTime))
	return nil
}

// ForgotPassword 忘记密码 - 生成重置token
func (s *AuthService) ForgotPassword(ctx context.Context, email string) (string, error) {
	startTime := time.Now()

	// 检查邮箱是否存在
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		s.logger.Warn("邮箱不存在", "email", utils.SanitizeEmail(email))
		return "", utils.ErrUserNotFound
	}

	// 生成重置token
	resetToken := generateResetToken()

	// 设置过期时间（15分钟）
	expiresAt := time.Now().Add(15 * time.Minute)

	// 保存token到数据库
	tokenRecord := &models.PasswordResetToken{
		Email:     email,
		Token:     resetToken,
		ExpiresAt: expiresAt,
		Used:      false,
		CreatedAt: time.Now(),
	}

	err = s.userRepo.CreatePasswordResetToken(ctx, tokenRecord)
	if err != nil {
		s.logger.Error("保存重置token失败",
			"email", utils.SanitizeEmail(email),
			"error", err.Error())
		return "", utils.ErrInternalServerError
	}

	s.logger.Info("忘记密码处理成功",
		"userID", user.ID,
		"email", utils.SanitizeEmail(email),
		"tokenID", tokenRecord.ID,
		"duration", time.Since(startTime))

	return resetToken, nil
}

// ResetPassword 重置密码
func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	startTime := time.Now()

	// 验证token
	tokenRecord, err := s.userRepo.GetPasswordResetToken(ctx, token)
	if err != nil {
		s.logger.Warn("重置token无效", "error", err.Error())
		return utils.ErrInvalidToken
	}

	// 检查token是否过期
	if time.Now().After(tokenRecord.ExpiresAt) {
		s.logger.Warn("重置token已过期",
			"tokenID", tokenRecord.ID,
			"email", utils.SanitizeEmail(tokenRecord.Email),
			"expiresAt", tokenRecord.ExpiresAt)
		return utils.ErrTokenExpired
	}

	// 获取用户信息
	user, err := s.userRepo.GetUserByEmail(ctx, tokenRecord.Email)
	if err != nil {
		s.logger.Error("获取用户信息失败",
			"email", utils.SanitizeEmail(tokenRecord.Email),
			"error", err.Error())
		return utils.ErrUserNotFound
	}

	// 验证新密码强度
	if !utils.ValidatePassword(newPassword) {
		s.logger.Warn("新密码强度不够", "userID", user.ID)
		return utils.ErrInvalidPassword
	}

	// 加密新密码
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		s.logger.Error("密码加密失败", "error", err.Error())
		return utils.ErrInternalServerError
	}

	// 更新密码
	err = s.userRepo.UpdatePassword(ctx, user.ID, hashedPassword)
	if err != nil {
		s.logger.Error("更新密码失败", "userID", user.ID, "error", err.Error())
		return err
	}

	// 标记token为已使用
	err = s.userRepo.MarkPasswordResetTokenAsUsed(ctx, tokenRecord.ID)
	if err != nil {
		s.logger.Error("标记token失败", "tokenID", tokenRecord.ID, "error", err.Error())
	}

	s.logger.Info("密码重置成功",
		"userID", user.ID,
		"email", utils.SanitizeEmail(tokenRecord.Email),
		"duration", time.Since(startTime))

	return nil
}

// generateResetToken 生成加密安全的重置token
func generateResetToken() string {
	const tokenBytes = 48
	b := make([]byte, tokenBytes)

	_, err := rand.Read(b)
	if err != nil {
		// 如果crypto/rand失败，使用时间戳作为后备方案
		return fmt.Sprintf("%d-%d-%d-%d",
			time.Now().UnixNano(),
			time.Now().Unix(),
			time.Now().UnixMicro(),
			time.Now().UnixMilli())
	}

	token := base64.URLEncoding.EncodeToString(b)
	token = strings.ReplaceAll(token, "=", "")
	return token
}
