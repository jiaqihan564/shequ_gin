package handlers

import (
	"context"
	"gin/internal/models"
	"gin/internal/services"
)

// UserWithProfile 用户基础信息和扩展信息的组合
type UserWithProfile struct {
	User     *models.User
	Profile  *models.UserExtraProfile
	Nickname string // 计算后的昵称（优先使用profile.nickname，否则使用username）
	Avatar   string // 头像URL
}

// GetUserWithProfile 获取用户基础信息和扩展信息
func GetUserWithProfile(ctx context.Context, userRepo *services.UserRepository, userID uint) (*UserWithProfile, error) {
	// 获取用户基础信息（必须成功）
	user, err := userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 获取用户扩展信息（失败时使用默认值）
	profile, _ := userRepo.GetUserProfile(ctx, userID)

	// 计算昵称和头像
	nickname := user.Username // 默认使用username
	avatar := ""

	if profile != nil {
		if profile.Nickname != "" {
			nickname = profile.Nickname
		}
		avatar = profile.AvatarURL
	}

	return &UserWithProfile{
		User:     user,
		Profile:  profile,
		Nickname: nickname,
		Avatar:   avatar,
	}, nil
}

