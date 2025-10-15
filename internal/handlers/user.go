package handlers

import (
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
	config      *config.Config
	logger      utils.Logger
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService services.UserServiceInterface, cfg *config.Config) *UserHandler {
	return &UserHandler{
		userService: userService,
		config:      cfg,
		logger:      utils.GetLogger(),
	}
}

// GetProfile 获取用户信息
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		h.logger.Warn("获取用户信息失败：用户未认证", "ip", c.ClientIP())
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	h.logger.Info("获取用户信息请求", "userID", userID, "ip", c.ClientIP())

	// 调用服务层获取用户信息
	ctx := c.Request.Context()
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Warn("获取用户信息失败", "userID", userID, "error", err.Error(), "ip", c.ClientIP())
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, err.Error())
		return
	}

	// 获取扩展资料（头像、昵称、简介）
	extra, _ := h.userService.GetUserProfile(ctx, userID)

	// 构建带时间戳的头像URL（防止缓存）
	avatarURL := h.buildAvatarURL(user.Username)

	h.logger.Info("获取用户信息成功", "userID", userID, "username", user.Username, "ip", c.ClientIP())
	utils.SuccessResponse(c, 200, "获取用户信息成功", gin.H{
		"user": user,
		"profile": gin.H{
			"nickname":   extra.Nickname,
			"bio":        extra.Bio,
			"avatar_url": avatarURL,
		},
	})
}

// UpdateProfile 更新用户信息
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		h.logger.Warn("更新用户信息失败：用户未认证", "ip", c.ClientIP())
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	var payload struct {
		Profile struct {
			Nickname string `json:"nickname"`
			Bio      string `json:"bio"`
		} `json:"profile"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		h.logger.Warn("更新用户信息请求参数绑定失败", "userID", userID, "error", err.Error(), "ip", c.ClientIP())
		utils.ValidationErrorResponse(c, "请求参数错误: "+err.Error())
		return
	}

	// 持久化昵称/简介到 user_profile
	if payload.Profile.Nickname != "" || payload.Profile.Bio != "" {
		// 验证昵称和简介
		if payload.Profile.Nickname != "" && !utils.ValidateNickname(payload.Profile.Nickname) {
			utils.ValidationErrorResponse(c, "昵称格式不正确，长度应为1-50个字符")
			return
		}
		if payload.Profile.Bio != "" && !utils.ValidateBio(payload.Profile.Bio) {
			utils.ValidationErrorResponse(c, "简介过长，最多500个字符")
			return
		}

		// 清理输入
		nickname := utils.SanitizeString(payload.Profile.Nickname)
		bio := utils.SanitizeString(payload.Profile.Bio)

		prof := &models.UserExtraProfile{UserID: userID, Nickname: nickname, Bio: bio}
		if err := h.userService.UpsertUserProfile(c.Request.Context(), prof); err != nil {
			statusCode := utils.GetHTTPStatusCode(err)
			utils.ErrorResponse(c, statusCode, err.Error())
			return
		}
	}

	// 返回当前用户信息与扩展资料
	ctx := c.Request.Context()
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Warn("获取用户信息失败", "userID", userID, "error", err.Error(), "ip", c.ClientIP())
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, err.Error())
		return
	}

	extra, _ := h.userService.GetUserProfile(ctx, userID)

	// 构建带时间戳的头像URL（防止缓存）
	avatarURL := h.buildAvatarURL(user.Username)

	h.logger.Info("更新用户信息成功", "userID", userID, "nickname", payload.Profile.Nickname, "ip", c.ClientIP())
	utils.SuccessResponse(c, 200, "更新成功", gin.H{
		"user": user,
		"profile": gin.H{
			"nickname":   extra.Nickname,
			"bio":        extra.Bio,
			"avatar_url": avatarURL,
		},
	})
}

// UpdateMe 更新当前用户信息（前端统一接口）
func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		h.logger.Warn("更新用户信息失败：用户未认证", "ip", c.ClientIP())
		utils.UnauthorizedResponse(c, err.Error())
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

	if err := c.ShouldBindJSON(&payload); err != nil {
		h.logger.Warn("更新用户信息请求参数错误", "userID", userID, "error", err.Error(), "ip", c.ClientIP())
		utils.ValidationErrorResponse(c, "请求参数错误: "+err.Error())
		return
	}

	h.logger.Info("收到用户信息更新请求", "userID", userID, "ip", c.ClientIP())

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
		if err := h.userService.UpdateUserAvatar(c.Request.Context(), prof); err != nil {
			h.logger.Error("更新头像失败", "userID", userID, "error", err.Error())
			statusCode := utils.GetHTTPStatusCode(err)
			utils.ErrorResponse(c, statusCode, err.Error())
			return
		}
	}

	// 如果有个人资料，更新个人资料
	if payload.Profile.Nickname != "" || payload.Profile.Bio != "" {
		// 验证昵称和简介
		if payload.Profile.Nickname != "" && !utils.ValidateNickname(payload.Profile.Nickname) {
			utils.ValidationErrorResponse(c, "昵称格式不正确，长度应为1-50个字符")
			return
		}
		if payload.Profile.Bio != "" && !utils.ValidateBio(payload.Profile.Bio) {
			utils.ValidationErrorResponse(c, "简介过长，最多500个字符")
			return
		}

		// 清理输入
		nickname := utils.SanitizeString(payload.Profile.Nickname)
		bio := utils.SanitizeString(payload.Profile.Bio)

		prof := &models.UserExtraProfile{
			UserID:   userID,
			Nickname: nickname,
			Bio:      bio,
		}
		if err := h.userService.UpsertUserProfile(c.Request.Context(), prof); err != nil {
			statusCode := utils.GetHTTPStatusCode(err)
			utils.ErrorResponse(c, statusCode, err.Error())
			return
		}
	}

	// 返回更新后的完整用户信息
	h.returnUserProfile(c, userID)
}

// GetMe 获取当前用户信息（前端统一接口）
func (h *UserHandler) GetMe(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		h.logger.Warn("获取用户信息失败：用户未认证", "ip", c.ClientIP())
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	h.logger.Info("获取当前用户信息请求", "userID", userID, "ip", c.ClientIP())
	h.returnUserProfile(c, userID)
}

// returnUserProfile 返回用户完整信息（统一格式）
func (h *UserHandler) returnUserProfile(c *gin.Context, userID uint) {
	ctx := c.Request.Context()

	// 获取基本用户信息
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Warn("获取用户信息失败", "userID", userID, "error", err.Error(), "ip", c.ClientIP())
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

	// 构建前端期望的响应格式
	response := gin.H{
		"id":             user.ID,
		"username":       user.Username,
		"email":          user.Email,
		"avatar":         avatarURL, // 前端期望字段名为 avatar
		"auth_status":    user.AuthStatus,
		"account_status": user.AccountStatus,
		"profile": gin.H{
			"nickname": nickname,
			"bio":      bio,
			// 省份和城市暂时为空，可以后续扩展
			"province": "",
			"city":     "",
		},
		"updatedAt": user.UpdatedAt,
	}

	h.logger.Info("获取用户信息成功", "userID", userID, "username", user.Username, "ip", c.ClientIP())
	utils.SuccessResponse(c, 200, "获取成功", response)
}

// GetUserByID 根据ID获取用户信息（管理员功能）
func (h *UserHandler) GetUserByID(c *gin.Context) {
	// 检查当前用户是否有权限查看其他用户信息
	currentUserID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		h.logger.Warn("获取用户信息失败：用户未认证", "ip", c.ClientIP())
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	// 解析目标用户ID
	targetUserID, err := utils.ParseUintParam(c, "id")
	if err != nil {
		h.logger.Warn("获取用户信息失败：无效的用户ID", "targetID", c.Param("id"), "currentUserID", currentUserID, "ip", c.ClientIP())
		utils.BadRequestResponse(c, "无效的用户ID")
		return
	}

	h.logger.Info("获取用户信息请求", "currentUserID", currentUserID, "targetUserID", targetUserID, "ip", c.ClientIP())

	// 调用服务层获取用户信息
	ctx := c.Request.Context()
	user, err := h.userService.GetUserByID(ctx, targetUserID)
	if err != nil {
		h.logger.Warn("获取用户信息失败", "currentUserID", currentUserID, "targetUserID", targetUserID, "error", err.Error(), "ip", c.ClientIP())
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, err.Error())
		return
	}

	h.logger.Info("获取用户信息成功", "currentUserID", currentUserID, "targetUserID", targetUserID, "username", user.Username, "ip", c.ClientIP())
	utils.SuccessResponse(c, 200, "获取用户信息成功", user)
}

// buildAvatarURL 构建带时间戳的头像URL（防止浏览器缓存）
func (h *UserHandler) buildAvatarURL(username string) string {
	base := h.config.Assets.PublicBaseURL
	if base == "" {
		return ""
	}
	// 移除末尾斜杠
	if base[len(base)-1] == '/' {
		base = base[:len(base)-1]
	}
	// 添加时间戳参数，确保每次都能获取最新头像
	return fmt.Sprintf("%s/%s/avatar.png?t=%d", base, username, time.Now().Unix())
}

// fixAvatarURL 修正头像URL中的IP地址（处理配置变更）
func (h *UserHandler) fixAvatarURL(oldURL, username string) string {
	// 如果数据库中的URL使用了错误的IP，重新构建正确的URL
	currentBase := h.config.Assets.PublicBaseURL
	if currentBase == "" {
		return oldURL
	}

	// 移除末尾斜杠
	if currentBase[len(currentBase)-1] == '/' {
		currentBase = currentBase[:len(currentBase)-1]
	}

	// 重新构建正确的URL（不带时间戳）
	return fmt.Sprintf("%s/%s/avatar.png", currentBase, username)
}
