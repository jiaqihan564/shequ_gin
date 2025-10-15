package services

import (
	"context"
	"time"

	"gin/internal/config"
	"gin/internal/models"
	"gin/internal/utils"

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
		return nil, utils.ErrInvalidCredentials
	}

	// 检查账户状态
	if user.AccountStatus != 1 {
		return nil, utils.ErrAccountDisabled
	}

	// 检查登录失败次数
	if user.FailedLoginCount >= s.config.Security.MaxLoginAttempts {
		return nil, utils.ErrTooManyLoginAttempts
	}

	// 验证密码
	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		_ = s.userRepo.IncrementFailedLoginCount(ctx, user.ID)
		return nil, utils.ErrInvalidCredentials
	}

	// 更新登录信息
	now := time.Now()
	if err := s.userRepo.UpdateLoginInfo(ctx, user.ID, now, clientIP); err != nil {
		s.logger.Error("更新登录信息失败", "userID", user.ID, "error", err.Error())
	}

	// 生成JWT token
	token, err := s.generateJWT(user.ID, user.Username)
	if err != nil {
		return nil, utils.ErrInternalServerError
	}

	// 读取扩展资料
	extra, _ := s.userRepo.GetUserProfile(ctx, user.ID)

	return &models.LoginResponse{
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
				AvatarURL:     extra.AvatarURL,
				Nickname:      extra.Nickname,
				Bio:           extra.Bio,
			},
		},
	}, nil
}

// Register 用户注册
func (s *AuthService) Register(ctx context.Context, username, password, email string) (*models.LoginResponse, error) {
	// 检查用户名和邮箱是否存在
	if exists, _ := s.userRepo.CheckUsernameExists(ctx, username); exists {
		return nil, utils.ErrUserAlreadyExists
	}
	if exists, _ := s.userRepo.CheckEmailExists(ctx, email); exists {
		return nil, utils.ErrEmailAlreadyExists
	}

	// 加密密码
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, utils.ErrInternalServerError
	}

	// 创建用户
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

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, utils.ErrDatabaseInsert
	}

	// 重新获取用户以获取ID
	user, err = s.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, utils.ErrDatabaseQuery
	}

	// 生成JWT token
	token, err := s.generateJWT(user.ID, user.Username)
	if err != nil {
		return nil, utils.ErrInternalServerError
	}

	// 读取扩展资料
	extra, _ := s.userRepo.GetUserProfile(ctx, user.ID)

	return &models.LoginResponse{
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
				AvatarURL:     extra.AvatarURL,
				Nickname:      extra.Nickname,
				Bio:           extra.Bio,
			},
		},
	}, nil
}

// generateJWT 生成JWT token
func (s *AuthService) generateJWT(userID uint, username string) (string, error) {
	claims := models.CreateClaims(userID, username, s.config.JWT.Issuer, s.config.JWT.ExpireHours)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWT.SecretKey))
}
