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
	user, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {
		s.logger.Warn("获取用户信息失败", "userID", id, "error", err.Error())
		return nil, err
	}
	return user, nil
}

// GetUserProfile 读取扩展资料
func (s *UserService) GetUserProfile(ctx context.Context, userID uint) (*models.UserExtraProfile, error) {
	profile, err := s.userRepo.GetUserProfile(ctx, userID)
	if err != nil {
		s.logger.Warn("读取用户扩展资料失败", "userID", userID, "error", err.Error())
		return nil, err
	}
	return profile, nil
}

// UpsertUserProfile 创建或更新扩展资料
func (s *UserService) UpsertUserProfile(ctx context.Context, profile *models.UserExtraProfile) error {
	err := s.userRepo.UpsertUserProfile(ctx, profile)
	if err != nil {
		s.logger.Error("更新用户扩展资料失败", "userID", profile.UserID, "error", err.Error())
		return err
	}
	s.logger.Info("更新用户扩展资料成功", "userID", profile.UserID, "nickname", profile.Nickname)
	return nil
}

// UpdateUserAvatar 更新用户头像URL
func (s *UserService) UpdateUserAvatar(ctx context.Context, profile *models.UserExtraProfile) error {
	err := s.userRepo.UpdateUserAvatar(ctx, profile)
	if err != nil {
		s.logger.Error("更新用户头像失败", "userID", profile.UserID, "error", err.Error())
		return err
	}
	s.logger.Info("更新用户头像成功", "userID", profile.UserID)
	return nil
}
