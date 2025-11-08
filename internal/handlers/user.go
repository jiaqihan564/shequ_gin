package handlers

import (
	"context"
	"fmt"
	"time"

	"gin/internal/config"
	"gin/internal/models"
	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户处理器
type UserHandler struct {
	userService services.UserServiceInterface
	historyRepo *services.HistoryRepository
	config      *config.Config
	logger      utils.Logger
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService services.UserServiceInterface, historyRepo *services.HistoryRepository, cfg *config.Config) *UserHandler {
	return &UserHandler{
		userService: userService,
		historyRepo: historyRepo,
		config:      cfg,
		logger:      utils.GetLogger(),
	}
}

// UpdateMe 更新当前用户信息（前端统一接口）
func (h *UserHandler) UpdateMe(c *gin.Context) {
	reqCtx := extractRequestContext(c)
	userID, isOK := getUserIDOrFail(c)
	if !isOK {
		return
	}

	// 定义请求载荷结构，所有字段都是可选的
	var payload struct {
		Avatar  string `json:"avatar"` // 头像URL
		Profile struct {
			Nickname string `json:"nickname"`
			Bio      string `json:"bio"`
			Province string `json:"province"`
			City     string `json:"city"`
		} `json:"profile"`
	}

	if !bindJSONOrFail(c, &payload, h.logger, "UpdateMe") {
		return
	}

	h.logger.Info("收到用户信息更新请求",
		"userID", userID,
		"hasAvatar", payload.Avatar != "",
		"hasNickname", payload.Profile.Nickname != "",
		"hasBio", payload.Profile.Bio != "",
		"ip", reqCtx.ClientIP)

	// 如果有头像URL，验证格式并更新
	if payload.Avatar != "" {
		if !utils.ValidateURL(payload.Avatar) {
			h.logger.Warn("头像URL格式错误", "userID", userID, "url", payload.Avatar)
			utils.ValidationErrorResponse(c, "无效的头像URL格式")
			return
		}

		prof := &models.UserExtraProfile{
			UserID:    userID,
			AvatarURL: payload.Avatar,
		}
		err := h.userService.UpdateUserAvatar(c.Request.Context(), prof)
		if err != nil {
			h.logger.Error("更新头像失败", "userID", userID, "error", err.Error())
			statusCode := utils.GetHTTPStatusCode(err)
			utils.ErrorResponse(c, statusCode, err.Error())
			return
		}
	}

	// 如果有个人资料，更新个人资料
	if payload.Profile.Nickname != "" || payload.Profile.Bio != "" {
		// 验证昵称和简介
		if payload.Profile.Nickname != "" && !utils.ValidateNicknameWithConfig(payload.Profile.Nickname, &h.config.Validation.Nickname) {
			h.logger.Warn("昵称格式不正确", "userID", userID, "nickname", payload.Profile.Nickname)
			utils.ValidationErrorResponse(c, fmt.Sprintf("昵称格式不正确，长度应为%d-%d个字符", h.config.Validation.Nickname.MinLength, h.config.Validation.Nickname.MaxLength))
			return
		}
		if payload.Profile.Bio != "" && !utils.ValidateBioWithConfig(payload.Profile.Bio, &h.config.Validation.Bio) {
			h.logger.Warn("简介过长", "userID", userID, "bioLength", len(payload.Profile.Bio))
			utils.ValidationErrorResponse(c, fmt.Sprintf("简介过长，最多%d个字符", h.config.Validation.Bio.MaxLength))
			return
		}

		// 先获取当前用户信息（用于历史记录）
		currentUser, _ := h.userService.GetUserByID(c.Request.Context(), userID)
		currentProfile, _ := h.userService.GetUserProfile(c.Request.Context(), userID)

		// 只更新非空字段，保留原有数据
		prof := &models.UserExtraProfile{
			UserID:   userID,
			Nickname: currentProfile.Nickname,
			Bio:      currentProfile.Bio,
		}

		// 只在payload有值时才更新
		if payload.Profile.Nickname != "" {
			prof.Nickname = utils.SanitizeString(payload.Profile.Nickname)
		}
		if payload.Profile.Bio != "" {
			prof.Bio = utils.SanitizeString(payload.Profile.Bio)
		}

		err := h.userService.UpsertUserProfile(c.Request.Context(), prof)
		if err != nil {
			h.logger.Error("更新个人资料失败", "userID", userID, "error", err.Error())
			statusCode := utils.GetHTTPStatusCode(err)
			utils.ErrorResponse(c, statusCode, err.Error())
			return
		}

		// 使用Worker Pool记录资料修改历史（避免goroutine泄漏）
		if h.historyRepo != nil && currentUser != nil {
			username := currentUser.Username
			taskID := fmt.Sprintf("profile_history_%d_%d", userID, time.Now().Unix())
			_ = utils.SubmitTask(taskID, func(taskCtx context.Context) error {
				// 记录昵称修改
				if payload.Profile.Nickname != "" && payload.Profile.Nickname != currentProfile.Nickname {
					h.historyRepo.RecordProfileChange(userID, "nickname", currentProfile.Nickname, prof.Nickname, reqCtx.ClientIP)
					h.historyRepo.RecordOperationHistory(userID, username, "修改昵称", fmt.Sprintf("从'%s'改为'%s'", currentProfile.Nickname, prof.Nickname), reqCtx.ClientIP)
				}
				// 记录简介修改
				if payload.Profile.Bio != "" && payload.Profile.Bio != currentProfile.Bio {
					h.historyRepo.RecordProfileChange(userID, "bio", currentProfile.Bio, prof.Bio, reqCtx.ClientIP)
					h.historyRepo.RecordOperationHistory(userID, username, "修改简介", "修改个人简介", reqCtx.ClientIP)
				}
				return nil
			}, time.Duration(h.config.AsyncTasks.UserUpdateHistoryTimeout)*time.Second)
		}
	}

	h.logger.Info("用户信息更新完成", "userID", userID, "duration", time.Since(reqCtx.StartTime))

	// 返回更新后的完整用户信息
	h.returnUserProfile(c, userID)
}

// GetMe 获取当前用户信息（前端统一接口）
func (h *UserHandler) GetMe(c *gin.Context) {
	userID, isOK := getUserIDOrFail(c)
	if !isOK {
		return
	}

	h.returnUserProfile(c, userID)
}

// returnUserProfile 返回用户完整信息（统一格式）
func (h *UserHandler) returnUserProfile(c *gin.Context, userID uint) {
	ctx := c.Request.Context()

	// 获取基本用户信息
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Warn("获取用户信息失败", "userID", userID, "error", err.Error())
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, err.Error())
		return
	}

	// 获取扩展资料（可能为空）
	extra, _ := h.userService.GetUserProfile(ctx, userID)

	// 安全获取扩展资料字段（防止nil指针）
	avatarURL := ""
	nickname := ""
	bio := ""
	if extra != nil {
		// 如果数据库中有头像URL，修正URL并添加时间戳防缓存
		if extra.AvatarURL != "" {
			// 动态修正URL中的IP地址（如果配置发生变化）
			fixedURL := h.fixAvatarURL(extra.AvatarURL, user.Username)
			avatarURL = fmt.Sprintf("%s?t=%d", fixedURL, time.Now().Unix())
		}
		nickname = extra.Nickname
		bio = extra.Bio
	}

	// 检查用户是否为管理员（优化：使用AdminChecker，O(1)查找）
	role := utils.GetUserRole(h.config, user.Username)

	// 构建前端期望的响应格式
	response := gin.H{
		"id":             user.ID,
		"username":       user.Username,
		"email":          user.Email,
		"avatar":         avatarURL,
		"auth_status":    user.AuthStatus,
		"account_status": user.AccountStatus,
		"role":           role,
		"profile": gin.H{
			"nickname": nickname,
			"bio":      bio,
			"province": "",
			"city":     "",
		},
		"updatedAt": user.UpdatedAt,
	}

	utils.SuccessResponse(c, 200, "获取成功", response)
}

// GetUserByID 根据ID获取用户信息（管理员功能）
func (h *UserHandler) GetUserByID(c *gin.Context) {
	currentUserID, isOK := getUserIDOrFail(c)
	if !isOK {
		return
	}

	// 解析目标用户ID
	targetUserID, isOK := parseUintParam(c, "id", "无效的用户ID")
	if !isOK {
		return
	}

	h.logger.Info("获取用户信息请求", "currentUserID", currentUserID, "targetUserID", targetUserID)

	// 调用服务层获取用户信息
	ctx := c.Request.Context()
	user, err := h.userService.GetUserByID(ctx, targetUserID)
	if err != nil {
		h.logger.Warn("获取用户信息失败", "targetUserID", targetUserID, "error", err.Error())
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, err.Error())
		return
	}

	// 获取用户扩展资料（包括头像）
	profile, _ := h.userService.GetUserProfile(ctx, targetUserID)

	// 构建响应，包含完整的用户信息和profile
	response := gin.H{
		"id":             user.ID,
		"username":       user.Username,
		"email":          user.Email,
		"auth_status":    user.AuthStatus,
		"account_status": user.AccountStatus,
		"created_at":     user.CreatedAt,
		"updated_at":     user.UpdatedAt,
	}

	if profile != nil {
		response["profile"] = gin.H{
			"nickname":   profile.Nickname,
			"bio":        profile.Bio,
			"avatar_url": profile.AvatarURL,
		}
		// 同时在根级提供 avatar 字段（方便前端使用）
		response["avatar"] = profile.AvatarURL
	}

	h.logger.Info("获取用户信息成功", "targetUserID", targetUserID, "username", user.Username)
	utils.SuccessResponse(c, 200, "获取用户信息成功", response)
}

// buildAvatarURL 构建带时间戳的头像URL（7桶架构）
func (h *UserHandler) buildAvatarURL(username string) string {
	base := h.config.BucketUserAvatars.PublicBaseURL
	if base == "" {
		return ""
	}
	// 移除末尾斜杠
	if base[len(base)-1] == '/' {
		base = base[:len(base)-1]
	}
	// 添加时间戳参数，确保每次都能获取最新头像
	return fmt.Sprintf("%s/%s/current.jpg?t=%d", base, username, time.Now().Unix())
}

// fixAvatarURL 修正头像URL中的IP地址（7桶架构）
func (h *UserHandler) fixAvatarURL(oldURL, username string) string {
	// 如果数据库中的URL使用了错误的IP，重新构建正确的URL
	currentBase := h.config.BucketUserAvatars.PublicBaseURL
	if currentBase == "" {
		return oldURL
	}

	// 移除末尾斜杠
	if currentBase[len(currentBase)-1] == '/' {
		currentBase = currentBase[:len(currentBase)-1]
	}

	// 重新构建正确的URL（不带时间戳，7桶架构使用current.jpg）
	return fmt.Sprintf("%s/%s/current.jpg", currentBase, username)
}
