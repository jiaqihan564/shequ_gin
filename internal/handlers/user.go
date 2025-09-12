package handlers

import (
	"net/http"

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
		prof := &models.UserExtraProfile{UserID: userID, Nickname: payload.Profile.Nickname, Bio: payload.Profile.Bio}
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
	// 禁止通过编辑页面（JSON）修改头像，提示改用上传接口
	h.logger.Warn("更新头像被拒绝：不支持通过编辑页面修改", "ip", c.ClientIP(), "path", c.FullPath())
	utils.ErrorResponse(c, http.StatusMethodNotAllowed, "头像不支持通过编辑页面更改，请使用上传接口")
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

// 旧的邮箱校验逻辑已移除（邮箱不支持修改）
