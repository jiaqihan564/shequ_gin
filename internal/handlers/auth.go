package handlers

import (
	"gin/internal/models"
	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService services.AuthServiceInterface
	logger      utils.Logger
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(authService services.AuthServiceInterface) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      utils.GetLogger(),
	}
}

// Login 处理登录请求
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, "请求参数错误")
		return
	}

	// 清理和验证输入
	req.Username = utils.SanitizeString(req.Username)
	if !utils.ValidateUsername(req.Username) {
		utils.ValidationErrorResponse(c, "用户名格式错误")
		return
	}

	// 调用服务层进行登录验证
	response, err := h.authService.Login(c.Request.Context(), req.Username, req.Password, c.ClientIP())
	if err != nil {
		utils.ErrorResponse(c, utils.GetHTTPStatusCode(err), err.Error())
		return
	}

	h.logger.Info("登录成功", "userID", response.Data.User.ID, "username", req.Username)
	utils.SuccessResponse(c, 200, "登录成功", response.Data)
}

// Register 处理用户注册请求
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, "请求参数错误")
		return
	}

	// 清理输入
	req.Username = utils.SanitizeString(req.Username)
	req.Email = utils.SanitizeString(req.Email)

	// 验证输入格式
	if !utils.ValidateUsername(req.Username) {
		utils.ValidationErrorResponse(c, "用户名格式错误")
		return
	}
	if !utils.ValidatePassword(req.Password) {
		utils.ValidationErrorResponse(c, "密码格式错误")
		return
	}
	if !utils.ValidateEmail(req.Email) {
		utils.ValidationErrorResponse(c, "邮箱格式错误")
		return
	}

	// 调用服务层进行用户注册
	response, err := h.authService.Register(c.Request.Context(), req.Username, req.Password, req.Email)
	if err != nil {
		utils.ErrorResponse(c, utils.GetHTTPStatusCode(err), err.Error())
		return
	}

	h.logger.Info("注册成功", "userID", response.Data.User.ID, "username", req.Username)
	utils.SuccessResponse(c, 201, "注册成功", response.Data)
}

// Logout 处理退出登录请求
func (h *AuthHandler) Logout(c *gin.Context) {
	userID, _ := utils.GetUserIDFromContext(c)
	h.logger.Info("退出登录", "userID", userID)
	utils.SuccessResponse(c, 200, "退出成功", nil)
}
