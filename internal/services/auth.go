package services

import (
	"fmt"
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
}

// NewAuthService 创建认证服务
func NewAuthService(cfg *config.Config, userRepo *UserRepository) *AuthService {
	return &AuthService{
		config:   cfg,
		userRepo: userRepo,
	}
}

// Login 用户登录
func (s *AuthService) Login(username, password, clientIP string) (*models.LoginResponse, error) {
	// 获取用户信息
	user, err := s.userRepo.GetUserByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("用户名或密码错误")
	}

	// 检查账户状态
	if user.AccountStatus != 1 {
		return nil, fmt.Errorf("账户已被禁用")
	}

	// 验证密码
	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		// 增加登录失败次数
		s.userRepo.IncrementFailedLoginCount(user.ID)
		return nil, fmt.Errorf("用户名或密码错误")
	}

	// 更新登录信息
	now := time.Now()
	err = s.userRepo.UpdateLoginInfo(user.ID, now, clientIP)
	if err != nil {
		// 登录信息更新失败不影响登录流程，只记录错误
		fmt.Printf("更新登录信息失败: %v\n", err)
	}

	// 生成JWT token
	token, err := s.generateJWT(user.ID, user.Username)
	if err != nil {
		return nil, fmt.Errorf("生成token失败")
	}

	// 返回登录成功响应
	response := &models.LoginResponse{
		Code:    200,
		Message: "登录成功",
		Data: struct {
			Token string      `json:"token"`
			User  models.User `json:"user"`
		}{
			Token: token,
			User: models.User{
				ID:               user.ID,
				Username:         user.Username,
				Email:            user.Email,
				AuthStatus:       user.AuthStatus,
				AccountStatus:    user.AccountStatus,
				LastLoginTime:    user.LastLoginTime,
				LastLoginIP:      user.LastLoginIP,
				FailedLoginCount: user.FailedLoginCount,
				CreatedAt:        user.CreatedAt,
				UpdatedAt:        user.UpdatedAt,
			},
		},
	}

	return response, nil
}

// Register 用户注册
func (s *AuthService) Register(username, password, email string) (*models.LoginResponse, error) {
	// 检查用户名是否已存在
	usernameExists, err := s.userRepo.CheckUsernameExists(username)
	if err != nil {
		return nil, fmt.Errorf("检查用户名失败")
	}
	if usernameExists {
		return nil, fmt.Errorf("用户名已存在")
	}

	// 检查邮箱是否已存在
	emailExists, err := s.userRepo.CheckEmailExists(email)
	if err != nil {
		return nil, fmt.Errorf("检查邮箱失败")
	}
	if emailExists {
		return nil, fmt.Errorf("邮箱已被注册")
	}

	// 加密密码
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败")
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
	err = s.userRepo.CreateUser(user)
	if err != nil {
		return nil, fmt.Errorf("创建用户失败")
	}

	// 重新获取用户信息以获取生成的ID
	user, err = s.userRepo.GetUserByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败")
	}

	// 生成JWT token
	token, err := s.generateJWT(user.ID, user.Username)
	if err != nil {
		return nil, fmt.Errorf("生成token失败")
	}

	// 返回注册成功响应
	response := &models.LoginResponse{
		Code:    201,
		Message: "注册成功",
		Data: struct {
			Token string      `json:"token"`
			User  models.User `json:"user"`
		}{
			Token: token,
			User: models.User{
				ID:               user.ID,
				Username:         user.Username,
				Email:            user.Email,
				AuthStatus:       user.AuthStatus,
				AccountStatus:    user.AccountStatus,
				LastLoginTime:    user.LastLoginTime,
				LastLoginIP:      user.LastLoginIP,
				FailedLoginCount: user.FailedLoginCount,
				CreatedAt:        user.CreatedAt,
				UpdatedAt:        user.UpdatedAt,
			},
		},
	}

	return response, nil
}

// generateJWT 生成JWT token
func (s *AuthService) generateJWT(userID uint, username string) (string, error) {
	expirationTime := time.Now().Add(time.Duration(s.config.JWT.ExpireHours) * time.Hour)
	claims := &models.Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWT.SecretKey))
}
