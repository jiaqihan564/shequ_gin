package services

import (
	"fmt"
	"time"

	"gin/internal/models"
	"gin/internal/utils"
)

// UserService 用户服务
type UserService struct{}

// NewUserService 创建用户服务
func NewUserService() *UserService {
	return &UserService{}
}

// GetUserByID 根据ID获取用户信息
func (s *UserService) GetUserByID(id uint) (*models.User, error) {
	// 实际应用中应该从数据库查询用户信息
	// 这里使用示例数据
	if id == 1 {
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

// UpdateUser 更新用户信息
func (s *UserService) UpdateUser(id uint, email string) (*models.User, error) {
	// 获取当前用户信息
	user, err := s.GetUserByID(id)
	if err != nil {
		return nil, err
	}

	// 更新用户信息
	if email != "" {
		user.Email = email
	}
	user.UpdatedAt = time.Now()

	// 保存到数据库
	saveUser(user)

	return user, nil
}

// GetAllUsers 获取所有用户（管理员功能）
func (s *UserService) GetAllUsers() ([]*models.User, error) {
	// 实际应用中应该从数据库查询所有用户
	// 这里返回示例数据
	adminUser, _ := s.GetUserByID(1)
	return []*models.User{adminUser}, nil
}

// DeleteUser 删除用户（管理员功能）
func (s *UserService) DeleteUser(id uint) error {
	// 实际应用中应该从数据库删除用户
	// 这里只是模拟删除操作
	fmt.Printf("用户已删除: ID=%d\n", id)
	return nil
}
