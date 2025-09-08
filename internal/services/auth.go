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
	config *config.Config
}

// NewAuthService 创建认证服务
func NewAuthService(cfg *config.Config) *AuthService {
	return &AuthService{
		config: cfg,
	}
}

// Login 用户登录
func (s *AuthService) Login(username, password string) (*models.LoginResponse, error) {
	// 获取用户信息
	user, err := getUserByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("用户名或密码错误")
	}

	// 验证密码
	if !utils.CheckPasswordHash(password, user.Password) {
		return nil, fmt.Errorf("用户名或密码错误")
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
				ID:        user.ID,
				Username:  user.Username,
				Email:     user.Email,
				CreatedAt: user.CreatedAt,
				UpdatedAt: user.UpdatedAt,
			},
		},
	}

	return response, nil
}

// Register 用户注册
func (s *AuthService) Register(username, password, email string) (*models.LoginResponse, error) {
	// 检查用户名是否已存在
	existingUser, _ := getUserByUsername(username)
	if existingUser != nil {
		return nil, fmt.Errorf("用户名已存在")
	}

	// 检查邮箱是否已存在
	existingUserByEmail, _ := getUserByEmail(email)
	if existingUserByEmail != nil {
		return nil, fmt.Errorf("邮箱已被注册")
	}

	// 加密密码
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败")
	}

	// 创建新用户
	user := &models.User{
		Username:  username,
		Password:  hashedPassword,
		Email:     email,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 保存用户到数据库
	user.ID = uint(time.Now().Unix()) // 模拟ID生成
	saveUser(user)

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
				ID:        user.ID,
				Username:  user.Username,
				Email:     user.Email,
				CreatedAt: user.CreatedAt,
				UpdatedAt: user.UpdatedAt,
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

// getUserByUsername 根据用户名获取用户信息（示例实现）
func getUserByUsername(username string) (*models.User, error) {
	// 实际应用中应该从数据库查询用户信息
	// 这里使用示例数据
	if username == "admin" {
		// admin用户的密码是"password"的哈希值
		hashedPassword, _ := utils.HashPassword("password")
		return &models.User{
			ID:        1,
			Username:  "admin",
			Password:  hashedPassword,
			Email:     "admin@example.com",
			CreatedAt: time.Now().Add(-24 * time.Hour),
			UpdatedAt: time.Now().Add(-1 * time.Hour),
		}, nil
	}
	return nil, fmt.Errorf("用户不存在")
}

// getUserByEmail 根据邮箱获取用户信息（示例实现）
func getUserByEmail(email string) (*models.User, error) {
	// 实际应用中应该从数据库查询用户信息
	// 这里使用示例数据
	if email == "admin@example.com" {
		hashedPassword, _ := utils.HashPassword("password")
		return &models.User{
			ID:        1,
			Username:  "admin",
			Password:  hashedPassword,
			Email:     "admin@example.com",
			CreatedAt: time.Now().Add(-24 * time.Hour),
			UpdatedAt: time.Now().Add(-1 * time.Hour),
		}, nil
	}
	return nil, fmt.Errorf("用户不存在")
}

// saveUser 保存用户到数据库（示例实现）
func saveUser(user *models.User) {
	// 实际应用中应该保存到数据库
	// 这里只是模拟保存操作
	fmt.Printf("用户已保存: %+v\n", user)
}
