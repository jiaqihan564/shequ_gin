package handlers

import (
	"gin/internal/models"
	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户处理器
type UserHandler struct {
	userService services.UserServiceInterface
	logger      utils.Logger
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService services.UserServiceInterface) *UserHandler {
	return &UserHandler{
		userService: userService,
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

	h.logger.Info("获取用户信息成功", "userID", userID, "username", user.Username, "ip", c.ClientIP())
	utils.SuccessResponse(c, 200, "获取用户信息成功", user)
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
	h.logger.Info("更新用户信息成功", "userID", userID, "nickname", payload.Profile.Nickname, "ip", c.ClientIP())
	utils.SuccessResponse(c, 200, "OK", gin.H{
		"user":    user,
		"profile": gin.H{"nickname": extra.Nickname, "bio": extra.Bio},
	})
}

// UpdateAvatar 使用 JSON 提交的头像 URL 更新用户头像（兼容前端协议）
func (h *UserHandler) UpdateAvatar(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		h.logger.Warn("更新头像失败：用户未认证", "ip", c.ClientIP())
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	var payload struct {
		AvatarURL string `json:"avatarUrl" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		h.logger.Warn("更新头像请求参数错误", "userID", userID, "error", err.Error(), "ip", c.ClientIP())
		utils.ValidationErrorResponse(c, "请求参数错误: "+err.Error())
		return
	}

	// 验证URL格式
	if !utils.ValidateURL(payload.AvatarURL) {
		h.logger.Warn("更新头像失败：URL格式错误", "userID", userID, "url", payload.AvatarURL)
		utils.ValidationErrorResponse(c, "无效的URL格式")
		return
	}

	// 更新 user_profile 中的 avatar_url
	prof := &models.UserExtraProfile{
		UserID:    userID,
		AvatarURL: payload.AvatarURL,
	}
	if err := h.userService.UpdateUserAvatar(c.Request.Context(), prof); err != nil {
		h.logger.Error("更新头像失败", "userID", userID, "error", err.Error())
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, err.Error())
		return
	}

	h.logger.Info("更新头像成功", "userID", userID, "avatarUrl", payload.AvatarURL, "ip", c.ClientIP())
	utils.SuccessResponse(c, 200, "更新头像成功", gin.H{
		"avatarUrl": payload.AvatarURL,
	})
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
