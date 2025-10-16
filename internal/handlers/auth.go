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
	startTime := time.Now()
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()

	h.logger.Debug("【Login】开始处理登录请求",
		"ip", clientIP,
		"userAgent", userAgent,
		"method", c.Request.Method,
		"path", c.Request.URL.Path)

	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("【Login】登录请求参数绑定失败",
			"error", err.Error(),
			"ip", clientIP,
			"userAgent", userAgent,
			"duration", time.Since(startTime))
		utils.ValidationErrorResponse(c, "请求参数错误: "+err.Error())
		return
	}

	h.logger.Debug("【Login】请求参数解析成功",
		"username", req.Username,
		"usernameLength", len(req.Username),
		"passwordLength", len(req.Password),
		"ip", clientIP)

	// 验证输入
	h.logger.Debug("【Login】开始验证登录请求参数", "username", req.Username)
	if err := h.validateLoginRequest(&req); err != nil {
		h.logger.Warn("【Login】登录请求验证失败",
			"username", req.Username,
			"error", err.Error(),
			"ip", clientIP,
			"duration", time.Since(startTime))
		utils.ValidationErrorResponse(c, err.Error())
		return
	}
	h.logger.Debug("【Login】登录请求参数验证通过", "username", req.Username)

	h.logger.Info("【Login】收到登录请求",
		"username", req.Username,
		"ip", clientIP,
		"userAgent", userAgent)

	// 调用服务层进行登录验证
	h.logger.Debug("【Login】调用服务层进行登录验证",
		"username", req.Username,
		"province", req.Province,
		"city", req.City)
	ctx := c.Request.Context()
	serviceStartTime := time.Now()
	response, err := h.authService.Login(ctx, req.Username, req.Password, clientIP, req.Province, req.City)
	serviceLatency := time.Since(serviceStartTime)

	if err != nil {
		h.logger.Warn("【Login】登录验证失败",
			"username", req.Username,
			"error", err.Error(),
			"ip", clientIP,
			"serviceLatency", serviceLatency,
			"totalDuration", time.Since(startTime))

		// 根据错误类型返回不同的状态码
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, err.Error())
		return
	}

	h.logger.Info("【Login】登录成功",
		"userID", response.Data.User.ID,
		"username", req.Username,
		"email", utils.SanitizeEmail(response.Data.User.Email),
		"ip", clientIP,
		"serviceLatency", serviceLatency,
		"totalDuration", time.Since(startTime),
		"tokenLength", len(response.Data.Token))

	utils.SuccessResponse(c, 200, "登录成功", response.Data)
}

// Register 处理用户注册请求
func (h *AuthHandler) Register(c *gin.Context) {
	startTime := time.Now()
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()

	h.logger.Debug("【Register】开始处理注册请求",
		"ip", clientIP,
		"userAgent", userAgent,
		"method", c.Request.Method,
		"path", c.Request.URL.Path)

	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("【Register】注册请求参数绑定失败",
			"error", err.Error(),
			"ip", clientIP,
			"userAgent", userAgent,
			"duration", time.Since(startTime))
		utils.ValidationErrorResponse(c, "请求参数错误: "+err.Error())
		return
	}

	h.logger.Debug("【Register】请求参数解析成功",
		"username", req.Username,
		"email", utils.SanitizeEmail(req.Email),
		"usernameLength", len(req.Username),
		"passwordLength", len(req.Password),
		"emailLength", len(req.Email),
		"ip", clientIP)

	// 验证输入
	h.logger.Debug("【Register】开始验证注册请求参数",
		"username", req.Username,
		"email", utils.SanitizeEmail(req.Email))
	if err := h.validateRegisterRequest(&req); err != nil {
		h.logger.Warn("【Register】注册请求验证失败",
			"username", req.Username,
			"email", utils.SanitizeEmail(req.Email),
			"error", err.Error(),
			"ip", clientIP,
			"duration", time.Since(startTime))
		utils.ValidationErrorResponse(c, err.Error())
		return
	}
	h.logger.Debug("【Register】注册请求参数验证通过",
		"username", req.Username,
		"email", utils.SanitizeEmail(req.Email))

	h.logger.Info("【Register】收到注册请求",
		"username", req.Username,
		"email", utils.SanitizeEmail(req.Email),
		"ip", clientIP,
		"userAgent", userAgent)

	// 调用服务层进行用户注册
	h.logger.Debug("【Register】调用服务层进行用户注册",
		"username", req.Username,
		"email", utils.SanitizeEmail(req.Email))
	ctx := c.Request.Context()
	serviceStartTime := time.Now()
	response, err := h.authService.Register(ctx, req.Username, req.Password, req.Email)
	serviceLatency := time.Since(serviceStartTime)

	if err != nil {
		h.logger.Warn("【Register】用户注册失败",
			"username", req.Username,
			"email", utils.SanitizeEmail(req.Email),
			"error", err.Error(),
			"ip", clientIP,
			"serviceLatency", serviceLatency,
			"totalDuration", time.Since(startTime))

		// 根据错误类型返回不同的状态码
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, err.Error())
		return
	}

	h.logger.Info("【Register】用户注册成功",
		"userID", response.Data.User.ID,
		"username", req.Username,
		"email", utils.SanitizeEmail(req.Email),
		"ip", clientIP,
		"serviceLatency", serviceLatency,
		"totalDuration", time.Since(startTime),
		"tokenLength", len(response.Data.Token))

	utils.SuccessResponse(c, 201, "注册成功", response.Data)
}

// Logout 处理退出登录请求
func (h *AuthHandler) Logout(c *gin.Context) {
	startTime := time.Now()
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()

	h.logger.Debug("【Logout】开始处理退出登录请求",
		"ip", clientIP,
		"userAgent", userAgent,
		"method", c.Request.Method,
		"path", c.Request.URL.Path)

	// 对于基于JWT的无状态认证，服务端无法强制使现有token失效
	// 此处仅返回成功，客户端应删除本地保存的token
	userID, _ := utils.GetUserIDFromContext(c)
	username, _ := c.Get("username")

	h.logger.Info("【Logout】收到退出登录请求",
		"userID", userID,
		"username", username,
		"ip", clientIP,
		"userAgent", userAgent)

	h.logger.Debug("【Logout】退出登录处理完成（无状态JWT，仅返回成功）",
		"userID", userID,
		"username", username,
		"duration", time.Since(startTime))

	utils.SuccessResponse(c, 200, "退出登录成功", gin.H{"ok": true})
}

// validateLoginRequest 验证登录请求
func (h *AuthHandler) validateLoginRequest(req *models.LoginRequest) error {
	h.logger.Debug("【validateLoginRequest】开始验证登录参数", "username", req.Username)

	if req.Username == "" {
		h.logger.Debug("【validateLoginRequest】验证失败: 用户名为空")
		return utils.ErrMissingParameter
	}
	if req.Password == "" {
		h.logger.Debug("【validateLoginRequest】验证失败: 密码为空", "username", req.Username)
		return utils.ErrMissingParameter
	}

	h.logger.Debug("【validateLoginRequest】参数非空检查通过",
		"username", req.Username,
		"hasPassword", req.Password != "")

	// 清理输入
	originalUsername := req.Username
	req.Username = utils.SanitizeString(req.Username)
	if originalUsername != req.Username {
		h.logger.Debug("【validateLoginRequest】用户名已清理",
			"original", originalUsername,
			"sanitized", req.Username)
	}

	// 验证用户名格式
	if !utils.ValidateUsername(req.Username) {
		h.logger.Debug("【validateLoginRequest】验证失败: 用户名格式不正确",
			"username", req.Username)
		return utils.ErrInvalidUsername
	}

	h.logger.Debug("【validateLoginRequest】登录参数验证通过", "username", req.Username)
	return nil
}

// validateRegisterRequest 验证注册请求
func (h *AuthHandler) validateRegisterRequest(req *models.RegisterRequest) error {
	h.logger.Debug("【validateRegisterRequest】开始验证注册参数",
		"username", req.Username,
		"email", utils.SanitizeEmail(req.Email))

	if req.Username == "" {
		h.logger.Debug("【validateRegisterRequest】验证失败: 用户名为空")
		return utils.ErrMissingParameter
	}
	if req.Password == "" {
		h.logger.Debug("【validateRegisterRequest】验证失败: 密码为空", "username", req.Username)
		return utils.ErrMissingParameter
	}
	if req.Email == "" {
		h.logger.Debug("【validateRegisterRequest】验证失败: 邮箱为空", "username", req.Username)
		return utils.ErrMissingParameter
	}

	h.logger.Debug("【validateRegisterRequest】参数非空检查通过",
		"username", req.Username,
		"email", utils.SanitizeEmail(req.Email),
		"hasPassword", req.Password != "")

	// 清理输入
	originalUsername := req.Username
	originalEmail := req.Email
	req.Username = utils.SanitizeString(req.Username)
	req.Email = utils.SanitizeString(req.Email)

	if originalUsername != req.Username || originalEmail != req.Email {
		h.logger.Debug("【validateRegisterRequest】输入已清理",
			"usernameChanged", originalUsername != req.Username,
			"emailChanged", originalEmail != req.Email)
	}

	// 验证用户名格式
	if !utils.ValidateUsername(req.Username) {
		h.logger.Debug("【validateRegisterRequest】验证失败: 用户名格式不正确",
			"username", req.Username)
		return utils.ErrInvalidUsername
	}
	h.logger.Debug("【validateRegisterRequest】用户名格式验证通过", "username", req.Username)

	// 验证密码强度
	if !utils.ValidatePassword(req.Password) {
		h.logger.Debug("【validateRegisterRequest】验证失败: 密码强度不够",
			"username", req.Username,
			"passwordLength", len(req.Password))
		return utils.ErrInvalidPassword
	}
	h.logger.Debug("【validateRegisterRequest】密码强度验证通过",
		"username", req.Username,
		"passwordLength", len(req.Password))

	// 验证邮箱格式
	if !utils.ValidateEmail(req.Email) {
		h.logger.Debug("【validateRegisterRequest】验证失败: 邮箱格式不正确",
			"username", req.Username,
			"email", utils.SanitizeEmail(req.Email))
		return utils.ErrInvalidEmail
	}
	h.logger.Debug("【validateRegisterRequest】邮箱格式验证通过",
		"email", utils.SanitizeEmail(req.Email))

	h.logger.Debug("【validateRegisterRequest】注册参数验证全部通过",
		"username", req.Username,
		"email", utils.SanitizeEmail(req.Email))
	return nil
}

// ChangePassword 处理修改密码请求
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	startTime := time.Now()
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()

	h.logger.Debug("【ChangePassword】开始处理修改密码请求",
		"ip", clientIP,
		"userAgent", userAgent,
		"method", c.Request.Method,
		"path", c.Request.URL.Path)

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("【ChangePassword】修改密码请求参数绑定失败",
			"error", err.Error(),
			"ip", clientIP,
			"userAgent", userAgent,
			"duration", time.Since(startTime))
		utils.ValidationErrorResponse(c, "请求参数错误: "+err.Error())
		return
	}

	h.logger.Debug("【ChangePassword】请求参数解析成功",
		"currentPasswordLength", len(req.CurrentPassword),
		"newPasswordLength", len(req.NewPassword),
		"ip", clientIP)

	// 获取用户ID从上下文（由认证中间件设置）
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		h.logger.Warn("【ChangePassword】无法获取用户ID",
			"ip", clientIP,
			"error", err.Error(),
			"duration", time.Since(startTime))
		utils.UnauthorizedResponse(c, "未授权访问")
		return
	}

	h.logger.Info("【ChangePassword】收到修改密码请求",
		"userID", userID,
		"ip", clientIP,
		"userAgent", userAgent)

	// 调用服务层进行密码修改
	h.logger.Debug("【ChangePassword】调用服务层进行密码修改", "userID", userID)
	ctx := c.Request.Context()
	serviceStartTime := time.Now()
	err = h.authService.ChangePassword(ctx, userID, req.CurrentPassword, req.NewPassword)
	serviceLatency := time.Since(serviceStartTime)

	if err != nil {
		h.logger.Warn("【ChangePassword】密码修改失败",
			"userID", userID,
			"error", err.Error(),
			"ip", clientIP,
			"serviceLatency", serviceLatency,
			"totalDuration", time.Since(startTime))

		// 根据错误类型返回不同的状态码
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, err.Error())
		return
	}

	h.logger.Info("【ChangePassword】密码修改成功",
		"userID", userID,
		"ip", clientIP,
		"serviceLatency", serviceLatency,
		"totalDuration", time.Since(startTime))

	utils.SuccessResponse(c, 200, "密码修改成功", gin.H{"ok": true})
}

// ForgotPassword 处理忘记密码请求
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	startTime := time.Now()
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()

	h.logger.Debug("【ForgotPassword】开始处理忘记密码请求",
		"ip", clientIP,
		"userAgent", userAgent,
		"method", c.Request.Method,
		"path", c.Request.URL.Path)

	var req models.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("【ForgotPassword】忘记密码请求参数绑定失败",
			"error", err.Error(),
			"ip", clientIP,
			"userAgent", userAgent,
			"duration", time.Since(startTime))
		utils.ValidationErrorResponse(c, "请求参数错误: "+err.Error())
		return
	}

	h.logger.Debug("【ForgotPassword】请求参数解析成功",
		"email", utils.SanitizeEmail(req.Email),
		"ip", clientIP)

	// 验证邮箱格式
	if !utils.ValidateEmail(req.Email) {
		h.logger.Warn("【ForgotPassword】邮箱格式不正确",
			"email", utils.SanitizeEmail(req.Email),
			"ip", clientIP,
			"duration", time.Since(startTime))
		utils.ValidationErrorResponse(c, "邮箱格式不正确")
		return
	}

	h.logger.Info("【ForgotPassword】收到忘记密码请求",
		"email", utils.SanitizeEmail(req.Email),
		"ip", clientIP,
		"userAgent", userAgent)

	// 调用服务层生成重置token
	h.logger.Debug("【ForgotPassword】调用服务层生成重置token",
		"email", utils.SanitizeEmail(req.Email))
	ctx := c.Request.Context()
	serviceStartTime := time.Now()
	token, err := h.authService.ForgotPassword(ctx, req.Email)
	serviceLatency := time.Since(serviceStartTime)

	if err != nil {
		h.logger.Warn("【ForgotPassword】生成重置token失败",
			"email", utils.SanitizeEmail(req.Email),
			"error", err.Error(),
			"ip", clientIP,
			"serviceLatency", serviceLatency,
			"totalDuration", time.Since(startTime))

		// 根据错误类型返回不同的状态码
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, err.Error())
		return
	}

	h.logger.Info("【ForgotPassword】重置token生成成功",
		"email", utils.SanitizeEmail(req.Email),
		"ip", clientIP,
		"serviceLatency", serviceLatency,
		"totalDuration", time.Since(startTime),
		"tokenLength", len(token))

	// 返回成功响应（在实际生产环境中不应该返回token，而是发送邮件）
	// 这里为了测试方便，暂时返回token
	utils.SuccessResponse(c, 200, "重置链接已生成（生产环境将发送到邮箱）", gin.H{
		"token":   token,
		"message": "请保存此token用于重置密码（有效期15分钟）",
	})
}

// ResetPassword 处理重置密码请求
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	startTime := time.Now()
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()

	h.logger.Debug("【ResetPassword】开始处理重置密码请求",
		"ip", clientIP,
		"userAgent", userAgent,
		"method", c.Request.Method,
		"path", c.Request.URL.Path)

	var req models.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("【ResetPassword】重置密码请求参数绑定失败",
			"error", err.Error(),
			"ip", clientIP,
			"userAgent", userAgent,
			"duration", time.Since(startTime))
		utils.ValidationErrorResponse(c, "请求参数错误: "+err.Error())
		return
	}

	h.logger.Debug("【ResetPassword】请求参数解析成功",
		"tokenLength", len(req.Token),
		"newPasswordLength", len(req.NewPassword),
		"ip", clientIP)

	h.logger.Info("【ResetPassword】收到重置密码请求",
		"ip", clientIP,
		"userAgent", userAgent)

	// 调用服务层重置密码
	h.logger.Debug("【ResetPassword】调用服务层重置密码")
	ctx := c.Request.Context()
	serviceStartTime := time.Now()
	err := h.authService.ResetPassword(ctx, req.Token, req.NewPassword)
	serviceLatency := time.Since(serviceStartTime)

	if err != nil {
		h.logger.Warn("【ResetPassword】重置密码失败",
			"error", err.Error(),
			"ip", clientIP,
			"serviceLatency", serviceLatency,
			"totalDuration", time.Since(startTime))

		// 根据错误类型返回不同的状态码
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, err.Error())
		return
	}

	h.logger.Info("【ResetPassword】密码重置成功",
		"ip", clientIP,
		"serviceLatency", serviceLatency,
		"totalDuration", time.Since(startTime))

	utils.SuccessResponse(c, 200, "密码重置成功，请使用新密码登录", gin.H{"ok": true})
}
