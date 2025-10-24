package handlers

import (
	"time"

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
	reqCtx := extractRequestContext(c)

	var req models.LoginRequest
	if !bindJSONOrFail(c, &req, h.logger, "Login") {
		return
	}

	// 验证输入
	if err := h.validateLoginRequest(&req); err != nil {
		h.logger.Warn("登录请求验证失败",
			"username", req.Username,
			"error", err.Error(),
			"ip", reqCtx.ClientIP)
		utils.ValidationErrorResponse(c, err.Error())
		return
	}

	h.logger.Info("收到登录请求",
		"username", req.Username,
		"ip", reqCtx.ClientIP)

	// 调用服务层进行登录验证
	ctx := c.Request.Context()
	response, err := h.authService.Login(ctx, req.Username, req.Password, reqCtx.ClientIP, req.Province, req.City)
	if err != nil {
		h.logger.Warn("登录验证失败",
			"username", req.Username,
			"error", err.Error(),
			"ip", reqCtx.ClientIP)
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, err.Error())
		return
	}

	h.logger.Info("登录成功",
		"userID", response.Data.User.ID,
		"username", req.Username,
		"ip", reqCtx.ClientIP,
		"duration", time.Since(reqCtx.StartTime))

	utils.SuccessResponse(c, 200, "登录成功", response.Data)
}

// Register 处理用户注册请求
func (h *AuthHandler) Register(c *gin.Context) {
	reqCtx := extractRequestContext(c)

	var req models.RegisterRequest
	if !bindJSONOrFail(c, &req, h.logger, "Register") {
		return
	}

	// 验证输入
	if err := h.validateRegisterRequest(&req); err != nil {
		h.logger.Warn("注册请求验证失败",
			"username", req.Username,
			"email", utils.SanitizeEmail(req.Email),
			"error", err.Error(),
			"ip", reqCtx.ClientIP)
		utils.ValidationErrorResponse(c, err.Error())
		return
	}

	h.logger.Info("收到注册请求",
		"username", req.Username,
		"email", utils.SanitizeEmail(req.Email),
		"ip", reqCtx.ClientIP)

	// 调用服务层进行用户注册
	ctx := c.Request.Context()
	response, err := h.authService.Register(ctx, req.Username, req.Password, req.Email, reqCtx.ClientIP, reqCtx.UserAgent, req.Province, req.City)
	if err != nil {
		h.logger.Warn("用户注册失败",
			"username", req.Username,
			"email", utils.SanitizeEmail(req.Email),
			"error", err.Error(),
			"ip", reqCtx.ClientIP)
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, err.Error())
		return
	}

	h.logger.Info("用户注册成功",
		"userID", response.Data.User.ID,
		"username", req.Username,
		"email", utils.SanitizeEmail(req.Email),
		"ip", reqCtx.ClientIP,
		"duration", time.Since(reqCtx.StartTime))

	utils.SuccessResponse(c, 201, "注册成功", response.Data)
}

// Logout 处理退出登录请求
func (h *AuthHandler) Logout(c *gin.Context) {
	reqCtx := extractRequestContext(c)
	userID, _ := utils.GetUserIDFromContext(c)
	username, _ := c.Get("username")

	h.logger.Info("收到退出登录请求",
		"userID", userID,
		"username", username,
		"ip", reqCtx.ClientIP)

	utils.SuccessResponse(c, 200, "退出登录成功", gin.H{"ok": true})
}

// validateLoginRequest 验证登录请求
func (h *AuthHandler) validateLoginRequest(req *models.LoginRequest) error {
	if req.Username == "" || req.Password == "" {
		return utils.ErrMissingParameter
	}

	// 清理输入
	req.Username = utils.SanitizeString(req.Username)

	// 安全检查：防止SQL注入和XSS
	if utils.DetectSQLInjection(req.Username) {
		h.logger.Warn("检测到SQL注入尝试", "username", req.Username)
		return utils.ErrInvalidUsername
	}
	if utils.DetectXSS(req.Username) {
		h.logger.Warn("检测到XSS攻击尝试", "username", req.Username)
		return utils.ErrInvalidUsername
	}

	// 验证用户名格式
	if !utils.ValidateUsername(req.Username) {
		return utils.ErrInvalidUsername
	}

	// 验证密码长度（不验证复杂度，因为是登录不是注册）
	if len(req.Password) < 6 || len(req.Password) > 100 {
		return utils.ErrInvalidPassword
	}

	return nil
}

// validateRegisterRequest 验证注册请求
func (h *AuthHandler) validateRegisterRequest(req *models.RegisterRequest) error {
	if req.Username == "" || req.Password == "" || req.Email == "" {
		return utils.ErrMissingParameter
	}

	// 清理输入
	req.Username = utils.SanitizeString(req.Username)
	req.Email = utils.SanitizeString(req.Email)

	// 安全检查：防止SQL注入和XSS
	if utils.DetectSQLInjection(req.Username) || utils.DetectSQLInjection(req.Email) {
		h.logger.Warn("检测到SQL注入尝试",
			"username", req.Username,
			"email", utils.SanitizeEmail(req.Email))
		return utils.ErrInvalidParameter
	}
	if utils.DetectXSS(req.Username) || utils.DetectXSS(req.Email) {
		h.logger.Warn("检测到XSS攻击尝试",
			"username", req.Username,
			"email", utils.SanitizeEmail(req.Email))
		return utils.ErrInvalidParameter
	}

	// 验证用户名格式
	if !utils.ValidateUsername(req.Username) {
		return utils.ErrInvalidUsername
	}

	// 验证密码强度
	if !utils.ValidatePassword(req.Password) {
		return utils.NewAppError(utils.ErrInvalidPassword,
			"密码必须至少6位，并包含字母和数字", 400)
	}

	// 验证邮箱格式
	if !utils.ValidateEmail(req.Email) {
		return utils.ErrInvalidEmail
	}

	return nil
}

// ChangePassword 处理修改密码请求
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	reqCtx := extractRequestContext(c)

	var req models.ChangePasswordRequest
	if !bindJSONOrFail(c, &req, h.logger, "ChangePassword") {
		return
	}

	userID, isOK := getUserIDOrFail(c)
	if !isOK {
		return
	}

	h.logger.Info("收到修改密码请求",
		"userID", userID,
		"ip", reqCtx.ClientIP)

	// 调用服务层进行密码修改
	ctx := c.Request.Context()
	err := h.authService.ChangePassword(ctx, userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		h.logger.Warn("密码修改失败",
			"userID", userID,
			"error", err.Error(),
			"ip", reqCtx.ClientIP)
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, err.Error())
		return
	}

	h.logger.Info("密码修改成功",
		"userID", userID,
		"ip", reqCtx.ClientIP,
		"duration", time.Since(reqCtx.StartTime))

	utils.SuccessResponse(c, 200, "密码修改成功", gin.H{"ok": true})
}

// ForgotPassword 处理忘记密码请求
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	reqCtx := extractRequestContext(c)

	var req models.ForgotPasswordRequest
	if !bindJSONOrFail(c, &req, h.logger, "ForgotPassword") {
		return
	}

	// 验证邮箱格式
	if !utils.ValidateEmail(req.Email) {
		h.logger.Warn("忘记密码请求邮箱格式不正确",
			"email", utils.SanitizeEmail(req.Email),
			"ip", reqCtx.ClientIP)
		utils.ValidationErrorResponse(c, "邮箱格式不正确")
		return
	}

	h.logger.Info("收到忘记密码请求",
		"email", utils.SanitizeEmail(req.Email),
		"ip", reqCtx.ClientIP)

	// 调用服务层生成重置token
	ctx := c.Request.Context()
	token, err := h.authService.ForgotPassword(ctx, req.Email)
	if err != nil {
		h.logger.Warn("生成重置token失败",
			"email", utils.SanitizeEmail(req.Email),
			"error", err.Error(),
			"ip", reqCtx.ClientIP)
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, err.Error())
		return
	}

	h.logger.Info("重置token生成成功",
		"email", utils.SanitizeEmail(req.Email),
		"ip", reqCtx.ClientIP,
		"duration", time.Since(reqCtx.StartTime))

	// 返回成功响应（在实际生产环境中不应该返回token，而是发送邮件）
	utils.SuccessResponse(c, 200, "重置链接已生成（生产环境将发送到邮箱）", gin.H{
		"token":   token,
		"message": "请保存此token用于重置密码（有效期15分钟）",
	})
}

// ResetPassword 处理重置密码请求
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	reqCtx := extractRequestContext(c)

	var req models.ResetPasswordRequest
	if !bindJSONOrFail(c, &req, h.logger, "ResetPassword") {
		return
	}

	h.logger.Info("收到重置密码请求", "ip", reqCtx.ClientIP)

	// 调用服务层重置密码
	ctx := c.Request.Context()
	err := h.authService.ResetPassword(ctx, req.Token, req.NewPassword)
	if err != nil {
		h.logger.Warn("重置密码失败",
			"error", err.Error(),
			"ip", reqCtx.ClientIP)
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, err.Error())
		return
	}

	h.logger.Info("密码重置成功",
		"ip", reqCtx.ClientIP,
		"duration", time.Since(reqCtx.StartTime))

	utils.SuccessResponse(c, 200, "密码重置成功，请使用新密码登录", gin.H{"ok": true})
}
