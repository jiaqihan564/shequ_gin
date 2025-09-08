package services

import (
	"fmt"
	"time"

	"gin/internal/models"
)

// UserService 用户服务
type UserService struct {
	userRepo *UserRepository
}

// NewUserService 创建用户服务
func NewUserService(userRepo *UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

// GetUserByID 根据ID获取用户信息
func (s *UserService) GetUserByID(id uint) (*models.User, error) {
	return s.userRepo.GetUserByID(id)
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
	err = s.userRepo.UpdateUser(user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetAllUsers 获取所有用户（管理员功能）
func (s *UserService) GetAllUsers() ([]*models.User, error) {
	// TODO: 实现获取所有用户的功能
	return nil, fmt.Errorf("功能暂未实现")
}

// DeleteUser 删除用户（管理员功能）
func (s *UserService) DeleteUser(id uint) error {
	// TODO: 实现删除用户的功能
	return fmt.Errorf("功能暂未实现")
}
