package services

import (
	"context"

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

// GetUserProfile 读取扩展资料
func (s *UserService) GetUserProfile(ctx context.Context, userID uint) (*models.UserExtraProfile, error) {
	return s.userRepo.GetUserProfile(ctx, userID)
}

// UpsertUserProfile 创建或更新扩展资料
func (s *UserService) UpsertUserProfile(ctx context.Context, profile *models.UserExtraProfile) error {
	return s.userRepo.UpsertUserProfile(ctx, profile)
}

// UpdateUserAvatar 更新用户头像URL
func (s *UserService) UpdateUserAvatar(ctx context.Context, profile *models.UserExtraProfile) error {
	return s.userRepo.UpdateUserAvatar(ctx, profile)
}
