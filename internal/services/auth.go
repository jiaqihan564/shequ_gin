package services

import (
	"context"
	"time"

	"gin/internal/config"
	"gin/internal/models"
	"gin/internal/utils"

	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v4"
)

// AuthService 认证服务
type AuthService struct {
	config   *config.Config
	userRepo *UserRepository
	logger   utils.Logger
}

// NewAuthService 创建认证服务
func NewAuthService(cfg *config.Config, userRepo *UserRepository) *AuthService {
	return &AuthService{
		config:   cfg,
		userRepo: userRepo,
		logger:   utils.GetLogger(),
	}
}

// Login 用户登录
func (s *AuthService) Login(ctx context.Context, username, password, clientIP string) (*models.LoginResponse, error) {
	// 获取用户信息
	user, err := s.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		s.logger.Warn("登录失败：用户不存在", "username", username, "ip", clientIP)
		return nil, utils.ErrInvalidCredentials
	}

	// 检查账户状态
	if user.AccountStatus != 1 {
		s.logger.Warn("登录失败：账户已被禁用", "userID", user.ID, "username", username, "ip", clientIP)
		return nil, utils.ErrAccountDisabled
	}

	// 检查登录失败次数
	if user.FailedLoginCount >= s.config.Security.MaxLoginAttempts {
		s.logger.Warn("登录失败：登录尝试次数过多", "userID", user.ID, "username", username, "failedCount", user.FailedLoginCount, "ip", clientIP)
		return nil, utils.ErrTooManyLoginAttempts
	}

	// 验证密码
	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		// 增加登录失败次数
		if err := s.userRepo.IncrementFailedLoginCount(ctx, user.ID); err != nil {
			s.logger.Error("更新登录失败次数失败", "userID", user.ID, "error", err.Error())
		}
		s.logger.Warn("登录失败：密码错误", "userID", user.ID, "username", username, "ip", clientIP)
		return nil, utils.ErrInvalidCredentials
	}

	// 更新登录信息
	now := time.Now()
	err = s.userRepo.UpdateLoginInfo(ctx, user.ID, now, clientIP)
	if err != nil {
		// 登录信息更新失败不影响登录流程，只记录错误
		s.logger.Error("更新登录信息失败", "userID", user.ID, "error", err.Error())
	}

	// 生成JWT token
	token, err := s.generateJWT(user.ID, user.Username)
	if err != nil {
		s.logger.Error("生成JWT token失败", "userID", user.ID, "error", err.Error())
		return nil, utils.ErrInternalServerError
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
				AvatarURL:     s.buildAvatarURL(user.Username),
			},
		},
	}

	s.logger.Info("用户登录成功", "userID", user.ID, "username", username, "ip", clientIP)
	return response, nil
}

// Register 用户注册
func (s *AuthService) Register(ctx context.Context, username, password, email string) (*models.LoginResponse, error) {
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
		s.logger.Error("检查邮箱失败", "email", email, "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}
	if emailExists {
		s.logger.Warn("注册失败：邮箱已被注册", "email", email)
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
		AuthStatus:    1, // 已验证
		AccountStatus: 1, // 正常
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// 保存用户到数据库
	err = s.userRepo.CreateUser(ctx, user)
	if err != nil {
		s.logger.Error("创建用户失败", "username", username, "email", email, "error", err.Error())
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
				AvatarURL:     s.buildAvatarURL(user.Username),
			},
		},
	}

	s.logger.Info("用户注册成功", "userID", user.ID, "username", username, "email", email)
	return response, nil
}

// generateJWT 生成JWT token
func (s *AuthService) generateJWT(userID uint, username string) (string, error) {
	claims := models.CreateClaims(userID, username, s.config.JWT.Issuer, s.config.JWT.ExpireHours)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWT.SecretKey))
}

// buildAvatarURL 根据用户名生成头像URL: {base}/{username}/avatar.png
func (s *AuthService) buildAvatarURL(username string) string {
	base := s.config.Assets.PublicBaseURL
	if base == "" {
		return ""
	}
	base = strings.TrimRight(base, "/")
	return fmt.Sprintf("%s/%s/avatar.png", base, username)
}
