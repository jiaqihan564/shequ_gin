package handlers

import (
	"gin/internal/models"
	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户处理器
type UserHandler struct {
	userService *services.UserService
	logger      utils.Logger
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService *services.UserService) *UserHandler {
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

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("更新用户信息请求参数绑定失败", "userID", userID, "error", err.Error(), "ip", c.ClientIP())
		utils.ValidationErrorResponse(c, "请求参数错误: "+err.Error())
		return
	}

	// 验证输入
	if err := h.validateUpdateRequest(&req); err != nil {
		h.logger.Warn("更新用户信息请求验证失败", "userID", userID, "error", err.Error(), "ip", c.ClientIP())
		utils.ValidationErrorResponse(c, err.Error())
		return
	}

	h.logger.Info("更新用户信息请求", "userID", userID, "email", req.Email, "ip", c.ClientIP())

	// 调用服务层更新用户信息
	ctx := c.Request.Context()
	user, err := h.userService.UpdateUser(ctx, userID, req.Email)
	if err != nil {
		h.logger.Warn("更新用户信息失败", "userID", userID, "error", err.Error(), "ip", c.ClientIP())
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, err.Error())
		return
	}

	h.logger.Info("更新用户信息成功", "userID", userID, "username", user.Username, "ip", c.ClientIP())
	utils.SuccessResponse(c, 200, "用户信息更新成功", user)
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

// validateUpdateRequest 验证更新请求
func (h *UserHandler) validateUpdateRequest(req *models.UpdateUserRequest) error {
	if req.Email != "" {
		// 清理输入
		req.Email = utils.SanitizeString(req.Email)

		// 验证邮箱格式
		if !utils.ValidateEmail(req.Email) {
			return utils.ErrInvalidEmail
		}
	}

	return nil
}
