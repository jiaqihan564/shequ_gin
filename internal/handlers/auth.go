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
		h.logger.Warn("登录请求参数绑定失败", "error", err.Error(), "ip", c.ClientIP())
		utils.ValidationErrorResponse(c, "请求参数错误: "+err.Error())
		return
	}

	// 验证输入
	if err := h.validateLoginRequest(&req); err != nil {
		h.logger.Warn("登录请求验证失败", "username", req.Username, "error", err.Error(), "ip", c.ClientIP())
		utils.ValidationErrorResponse(c, err.Error())
		return
	}

	h.logger.Info("收到登录请求", "username", req.Username, "ip", c.ClientIP())

	// 获取客户端IP
	clientIP := c.ClientIP()

	// 调用服务层进行登录验证
	ctx := c.Request.Context()
	response, err := h.authService.Login(ctx, req.Username, req.Password, clientIP)
	if err != nil {
		h.logger.Warn("登录验证失败", "username", req.Username, "error", err.Error(), "ip", clientIP)

		// 根据错误类型返回不同的状态码
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, err.Error())
		return
	}

	h.logger.Info("登录成功", "userID", response.Data.User.ID, "username", req.Username, "ip", clientIP)
	utils.SuccessResponse(c, 200, "登录成功", response.Data)
}

// Register 处理用户注册请求
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("注册请求参数绑定失败", "error", err.Error(), "ip", c.ClientIP())
		utils.ValidationErrorResponse(c, "请求参数错误: "+err.Error())
		return
	}

	// 验证输入
	if err := h.validateRegisterRequest(&req); err != nil {
		h.logger.Warn("注册请求验证失败", "username", req.Username, "email", req.Email, "error", err.Error(), "ip", c.ClientIP())
		utils.ValidationErrorResponse(c, err.Error())
		return
	}

	h.logger.Info("收到注册请求", "username", req.Username, "email", req.Email, "ip", c.ClientIP())

	// 调用服务层进行用户注册
	ctx := c.Request.Context()
	response, err := h.authService.Register(ctx, req.Username, req.Password, req.Email)
	if err != nil {
		h.logger.Warn("用户注册失败", "username", req.Username, "email", req.Email, "error", err.Error(), "ip", c.ClientIP())

		// 根据错误类型返回不同的状态码
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, err.Error())
		return
	}

	h.logger.Info("用户注册成功", "userID", response.Data.User.ID, "username", req.Username, "email", req.Email, "ip", c.ClientIP())
	utils.SuccessResponse(c, 201, "注册成功", response.Data)
}

// Logout 处理退出登录请求
func (h *AuthHandler) Logout(c *gin.Context) {
	// 对于基于JWT的无状态认证，服务端无法强制使现有token失效
	// 此处仅返回成功，客户端应删除本地保存的token
	userID, _ := utils.GetUserIDFromContext(c)
	h.logger.Info("收到退出登录请求", "userID", userID, "ip", c.ClientIP())

	utils.SuccessResponse(c, 200, "退出登录成功", gin.H{"ok": true})
}

// validateLoginRequest 验证登录请求
func (h *AuthHandler) validateLoginRequest(req *models.LoginRequest) error {
	if req.Username == "" {
		return utils.ErrMissingParameter
	}
	if req.Password == "" {
		return utils.ErrMissingParameter
	}

	// 清理输入
	req.Username = utils.SanitizeString(req.Username)

	// 验证用户名格式
	if !utils.ValidateUsername(req.Username) {
		return utils.ErrInvalidUsername
	}

	return nil
}

// validateRegisterRequest 验证注册请求
func (h *AuthHandler) validateRegisterRequest(req *models.RegisterRequest) error {
	if req.Username == "" {
		return utils.ErrMissingParameter
	}
	if req.Password == "" {
		return utils.ErrMissingParameter
	}
	if req.Email == "" {
		return utils.ErrMissingParameter
	}

	// 清理输入
	req.Username = utils.SanitizeString(req.Username)
	req.Email = utils.SanitizeString(req.Email)

	// 验证用户名格式
	if !utils.ValidateUsername(req.Username) {
		return utils.ErrInvalidUsername
	}

	// 验证密码强度
	if !utils.ValidatePassword(req.Password) {
		return utils.ErrInvalidPassword
	}

	// 验证邮箱格式
	if !utils.ValidateEmail(req.Email) {
		return utils.ErrInvalidEmail
	}

	return nil
}
