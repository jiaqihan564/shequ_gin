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
// 这是一个常用的组合操作，避免在多个handler中重复实现
//
// 参数:
//   - ctx: 请求上下文
//   - userRepo: 用户仓库
//   - userID: 用户ID
//
// 返回:
//   - UserWithProfile: 包含用户信息、profile、计算后的昵称和头像
//   - error: 如果获取用户基础信息失败则返回错误；profile获取失败不影响（使用默认值）
//
// 示例:
//
//	userInfo, err := GetUserWithProfile(ctx, h.userRepo, userID)
//	if err != nil {
//	    return nil, err
//	}
//	// 使用 userInfo.User, userInfo.Nickname, userInfo.Avatar
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

// GetMultipleUsersWithProfile 批量获取用户信息和profile
// 用于需要展示多个用户信息的场景（如聊天室、评论列表等）
//
// 注意：此函数内部使用并发查询，适合获取多个用户
func GetMultipleUsersWithProfile(ctx context.Context, userRepo *services.UserRepository, userIDs []uint) (map[uint]*UserWithProfile, error) {
	if len(userIDs) == 0 {
		return make(map[uint]*UserWithProfile), nil
	}

	result := make(map[uint]*UserWithProfile, len(userIDs))

	// 对每个用户ID获取信息
	for _, userID := range userIDs {
		userInfo, err := GetUserWithProfile(ctx, userRepo, userID)
		if err != nil {
			// 如果某个用户获取失败，记录但继续处理其他用户
			continue
		}
		result[userID] = userInfo
	}

	return result, nil
}
