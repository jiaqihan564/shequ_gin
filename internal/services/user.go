package services

import (
	"context"
	"time"

	"gin/internal/models"
	"gin/internal/utils"
)

// UserService 用户服务
type UserService struct {
	userRepo *UserRepository
	logger   utils.Logger
}

// NewUserService 创建用户服务
func NewUserService(userRepo *UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
		logger:   utils.GetLogger(),
	}
}

// GetUserByID 根据ID获取用户信息
func (s *UserService) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	return s.userRepo.GetUserByID(ctx, id)
}

// UpdateUser 更新用户信息
func (s *UserService) UpdateUser(ctx context.Context, id uint, email string) (*models.User, error) {
	// 获取当前用户信息
	user, err := s.GetUserByID(ctx, id)
	if err != nil {
		s.logger.Error("获取用户信息失败", "userID", id, "error", err.Error())
		return nil, err
	}

	// 更新用户信息
	if email != "" {
		user.Email = email
	}
	user.UpdatedAt = time.Now()

	// 保存到数据库
	err = s.userRepo.UpdateUser(ctx, user)
	if err != nil {
		s.logger.Error("更新用户信息失败", "userID", id, "error", err.Error())
		return nil, err
	}

	s.logger.Info("用户信息更新成功", "userID", id, "email", email)
	return user, nil
}
