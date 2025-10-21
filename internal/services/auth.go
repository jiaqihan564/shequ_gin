package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"gin/internal/config"
	"gin/internal/models"
	"gin/internal/utils"

	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v4"
)

// AuthService 认证服务
type AuthService struct {
	config      *config.Config
	userRepo    *UserRepository
	historyRepo *HistoryRepository
	logger      utils.Logger
}

// NewAuthService 创建认证服务
func NewAuthService(cfg *config.Config, userRepo *UserRepository, historyRepo *HistoryRepository) *AuthService {
	return &AuthService{
		config:      cfg,
		userRepo:    userRepo,
		historyRepo: historyRepo,
		logger:      utils.GetLogger(),
	}
}

// Login 用户登录
func (s *AuthService) Login(ctx context.Context, username, password, clientIP, province, city string) (*models.LoginResponse, error) {
	startTime := time.Now()
	s.logger.Debug("【AuthService.Login】开始处理登录业务逻辑",
		"username", username,
		"ip", clientIP,
		"province", province,
		"city", city)

	// 获取用户信息
	s.logger.Debug("【AuthService.Login】查询用户信息", "username", username)
	userQueryStart := time.Now()
	user, err := s.userRepo.GetUserByUsername(ctx, username)
	userQueryLatency := time.Since(userQueryStart)

	if err != nil {
		s.logger.Warn("【AuthService.Login】登录失败：用户不存在",
			"username", username,
			"ip", clientIP,
			"userQueryLatency", userQueryLatency,
			"totalDuration", time.Since(startTime))
		return nil, utils.ErrInvalidCredentials
	}

	s.logger.Debug("【AuthService.Login】用户信息查询成功",
		"userID", user.ID,
		"username", username,
		"authStatus", user.AuthStatus,
		"accountStatus", user.AccountStatus,
		"failedLoginCount", user.FailedLoginCount,
		"userQueryLatency", userQueryLatency)

	// 检查账户状态
	s.logger.Debug("【AuthService.Login】检查账户状态",
		"userID", user.ID,
		"accountStatus", user.AccountStatus)
	if user.AccountStatus != 1 {
		s.logger.Warn("【AuthService.Login】登录失败：账户已被禁用",
			"userID", user.ID,
			"username", username,
			"accountStatus", user.AccountStatus,
			"ip", clientIP,
			"duration", time.Since(startTime))
		return nil, utils.ErrAccountDisabled
	}

	// 检查登录失败次数
	s.logger.Debug("【AuthService.Login】检查登录失败次数",
		"userID", user.ID,
		"failedLoginCount", user.FailedLoginCount,
		"maxAttempts", s.config.Security.MaxLoginAttempts)
	if user.FailedLoginCount >= s.config.Security.MaxLoginAttempts {
		s.logger.Warn("【AuthService.Login】登录失败：登录尝试次数过多",
			"userID", user.ID,
			"username", username,
			"failedCount", user.FailedLoginCount,
			"maxAttempts", s.config.Security.MaxLoginAttempts,
			"ip", clientIP,
			"duration", time.Since(startTime))
		return nil, utils.ErrTooManyLoginAttempts
	}

	// 验证密码
	s.logger.Debug("【AuthService.Login】验证密码", "userID", user.ID)
	passwordCheckStart := time.Now()
	passwordValid := utils.CheckPasswordHash(password, user.PasswordHash)
	passwordCheckLatency := time.Since(passwordCheckStart)

	if !passwordValid {
		s.logger.Debug("【AuthService.Login】密码验证失败，增加失败计数",
			"userID", user.ID,
			"passwordCheckLatency", passwordCheckLatency)

		// 增加登录失败次数
		if err := s.userRepo.IncrementFailedLoginCount(ctx, user.ID); err != nil {
			s.logger.Error("【AuthService.Login】更新登录失败次数失败",
				"userID", user.ID,
				"error", err.Error())
		}
		s.logger.Warn("【AuthService.Login】登录失败：密码错误",
			"userID", user.ID,
			"username", username,
			"ip", clientIP,
			"passwordCheckLatency", passwordCheckLatency,
			"duration", time.Since(startTime))
		return nil, utils.ErrInvalidCredentials
	}

	s.logger.Debug("【AuthService.Login】密码验证通过",
		"userID", user.ID,
		"passwordCheckLatency", passwordCheckLatency)

	// 更新登录信息
	now := time.Now()
	s.logger.Debug("【AuthService.Login】更新登录信息",
		"userID", user.ID,
		"loginTime", now,
		"loginIP", clientIP)
	updateLoginStart := time.Now()
	err = s.userRepo.UpdateLoginInfo(ctx, user.ID, now, clientIP)
	updateLoginLatency := time.Since(updateLoginStart)

	if err != nil {
		// 登录信息更新失败不影响登录流程，只记录错误
		s.logger.Error("【AuthService.Login】更新登录信息失败",
			"userID", user.ID,
			"error", err.Error(),
			"updateLoginLatency", updateLoginLatency)
	} else {
		s.logger.Debug("【AuthService.Login】登录信息更新成功",
			"userID", user.ID,
			"updateLoginLatency", updateLoginLatency)
	}

	// 生成JWT token
	s.logger.Debug("【AuthService.Login】生成JWT token", "userID", user.ID)
	tokenGenStart := time.Now()
	token, err := s.generateJWT(user.ID, user.Username)
	tokenGenLatency := time.Since(tokenGenStart)

	if err != nil {
		s.logger.Error("【AuthService.Login】生成JWT token失败",
			"userID", user.ID,
			"error", err.Error(),
			"tokenGenLatency", tokenGenLatency,
			"duration", time.Since(startTime))
		return nil, utils.ErrInternalServerError
	}

	s.logger.Debug("【AuthService.Login】JWT token生成成功",
		"userID", user.ID,
		"tokenLength", len(token),
		"tokenGenLatency", tokenGenLatency)

	// 读取扩展资料（昵称/简介）
	s.logger.Debug("【AuthService.Login】读取用户扩展资料", "userID", user.ID)
	profileQueryStart := time.Now()
	extra, _ := s.userRepo.GetUserProfile(ctx, user.ID)
	profileQueryLatency := time.Since(profileQueryStart)
	s.logger.Debug("【AuthService.Login】扩展资料读取完成",
		"userID", user.ID,
		"hasNickname", extra.Nickname != "",
		"hasBio", extra.Bio != "",
		"hasAvatar", extra.AvatarURL != "",
		"profileQueryLatency", profileQueryLatency)

	// 检查用户是否为管理员
	isAdmin := false
	for _, adminUsername := range s.config.Admin.Usernames {
		if adminUsername == user.Username {
			isAdmin = true
			break
		}
	}

	// 确定用户角色
	role := "user"
	if isAdmin {
		role = "admin"
	}

	s.logger.Debug("【AuthService.Login】角色检查完成",
		"userID", user.ID,
		"username", user.Username,
		"role", role)

	// 返回登录成功响应（匹配前端期望格式）
	response := &models.LoginResponse{
		Code:    200,
		Message: "登录成功",
		Data: struct {
			Token string             `json:"token"`
			User  models.UserProfile `json:"user"`
		}{
			Token: token,
			User: models.UserProfile{
				ID:            user.ID,
				Username:      user.Username,
				Email:         user.Email,
				AuthStatus:    user.AuthStatus,
				AccountStatus: user.AccountStatus,
				AvatarURL:     extra.AvatarURL, // 使用数据库中的头像URL
				Nickname:      extra.Nickname,
				Bio:           extra.Bio,
				Role:          role, // 添加角色字段
			},
		},
	}

	s.logger.Info("【AuthService.Login】用户登录成功",
		"userID", user.ID,
		"username", username,
		"ip", clientIP,
		"totalDuration", time.Since(startTime),
		"breakdown", map[string]interface{}{
			"userQuery":     userQueryLatency.Milliseconds(),
			"passwordCheck": passwordCheckLatency.Milliseconds(),
			"updateLogin":   updateLoginLatency.Milliseconds(),
			"tokenGen":      tokenGenLatency.Milliseconds(),
			"profileQuery":  profileQueryLatency.Milliseconds(),
		})

	// 使用 Worker Pool 异步记录登录历史（不影响登录性能）
	if s.historyRepo != nil {
		userID := user.ID
		userName := username
		userIP := clientIP
		prov := province
		ct := city

		err := utils.SubmitTask(
			fmt.Sprintf("login-history-%d-%d", userID, time.Now().Unix()),
			func(ctx context.Context) error {
				userAgentStr := "" // 需要从上下文获取，这里简化处理
				if err := s.historyRepo.RecordLoginHistory(userID, userName, userIP, userAgentStr, prov, ct, 1); err != nil {
					s.logger.Error("记录登录历史失败", "userID", userID, "error", err.Error())
					return err
				}
				s.logger.Info("记录登录历史成功",
					"userID", userID,
					"province", prov,
					"city", ct)

				// 记录操作历史
				if err := s.historyRepo.RecordOperationHistory(userID, userName, "登录", "用户登录系统", userIP); err != nil {
					s.logger.Error("记录操作历史失败", "userID", userID, "error", err.Error())
					return err
				}
				return nil
			},
			10*time.Second,
		)
		if err != nil {
			s.logger.Warn("提交登录历史记录任务失败", "error", err.Error())
		}
	}

	return response, nil
}

// Register 用户注册
func (s *AuthService) Register(ctx context.Context, username, password, email, clientIP, userAgent, province, city string) (*models.LoginResponse, error) {
	startTime := time.Now()
	s.logger.Debug("【AuthService.Register】开始处理注册业务逻辑",
		"username", username,
		"email", utils.SanitizeEmail(email),
		"clientIP", clientIP,
		"province", province,
		"city", city)

	// 检查用户名是否已存在
	s.logger.Debug("【AuthService.Register】检查用户名是否已存在", "username", username)
	usernameCheckStart := time.Now()
	usernameExists, err := s.userRepo.CheckUsernameExists(ctx, username)
	usernameCheckLatency := time.Since(usernameCheckStart)

	if err != nil {
		s.logger.Error("【AuthService.Register】检查用户名失败",
			"username", username,
			"error", err.Error(),
			"usernameCheckLatency", usernameCheckLatency,
			"duration", time.Since(startTime))
		return nil, utils.ErrDatabaseQuery
	}

	s.logger.Debug("【AuthService.Register】用户名检查完成",
		"username", username,
		"exists", usernameExists,
		"usernameCheckLatency", usernameCheckLatency)

	if usernameExists {
		s.logger.Warn("【AuthService.Register】注册失败：用户名已存在",
			"username", username,
			"duration", time.Since(startTime))
		return nil, utils.ErrUserAlreadyExists
	}

	// 检查邮箱是否已存在
	s.logger.Debug("【AuthService.Register】检查邮箱是否已存在",
		"email", utils.SanitizeEmail(email))
	emailCheckStart := time.Now()
	emailExists, err := s.userRepo.CheckEmailExists(ctx, email)
	emailCheckLatency := time.Since(emailCheckStart)

	if err != nil {
		s.logger.Error("【AuthService.Register】检查邮箱失败",
			"email", utils.SanitizeEmail(email),
			"error", err.Error(),
			"emailCheckLatency", emailCheckLatency,
			"duration", time.Since(startTime))
		return nil, utils.ErrDatabaseQuery
	}

	s.logger.Debug("【AuthService.Register】邮箱检查完成",
		"email", utils.SanitizeEmail(email),
		"exists", emailExists,
		"emailCheckLatency", emailCheckLatency)

	if emailExists {
		s.logger.Warn("【AuthService.Register】注册失败：邮箱已被注册",
			"email", utils.SanitizeEmail(email),
			"duration", time.Since(startTime))
		return nil, utils.ErrEmailAlreadyExists
	}

	// 加密密码
	s.logger.Debug("【AuthService.Register】开始加密密码",
		"username", username,
		"passwordLength", len(password))
	hashPasswordStart := time.Now()
	hashedPassword, err := utils.HashPassword(password)
	hashPasswordLatency := time.Since(hashPasswordStart)

	if err != nil {
		s.logger.Error("【AuthService.Register】密码加密失败",
			"username", username,
			"error", err.Error(),
			"hashPasswordLatency", hashPasswordLatency,
			"duration", time.Since(startTime))
		return nil, utils.ErrInternalServerError
	}

	s.logger.Debug("【AuthService.Register】密码加密成功",
		"username", username,
		"hashedPasswordLength", len(hashedPassword),
		"hashPasswordLatency", hashPasswordLatency)

	// 创建新用户
	now := time.Now()
	user := &models.User{
		Username:      username,
		PasswordHash:  hashedPassword,
		Email:         email,
		AuthStatus:    1, // 已验证
		AccountStatus: 1, // 正常
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	s.logger.Debug("【AuthService.Register】准备创建用户记录",
		"username", username,
		"email", utils.SanitizeEmail(email),
		"authStatus", user.AuthStatus,
		"accountStatus", user.AccountStatus)

	// 保存用户到数据库
	createUserStart := time.Now()
	err = s.userRepo.CreateUser(ctx, user)
	createUserLatency := time.Since(createUserStart)

	if err != nil {
		s.logger.Error("【AuthService.Register】创建用户失败",
			"username", username,
			"email", utils.SanitizeEmail(email),
			"error", err.Error(),
			"createUserLatency", createUserLatency,
			"duration", time.Since(startTime))
		return nil, utils.ErrDatabaseInsert
	}

	s.logger.Debug("【AuthService.Register】用户创建成功",
		"username", username,
		"createUserLatency", createUserLatency)

	// 重新获取用户信息以获取生成的ID
	s.logger.Debug("【AuthService.Register】重新获取用户信息以确认ID", "username", username)
	getUserStart := time.Now()
	user, err = s.userRepo.GetUserByUsername(ctx, username)
	getUserLatency := time.Since(getUserStart)

	if err != nil {
		s.logger.Error("【AuthService.Register】获取用户信息失败",
			"username", username,
			"error", err.Error(),
			"getUserLatency", getUserLatency,
			"duration", time.Since(startTime))
		return nil, utils.ErrDatabaseQuery
	}

	s.logger.Debug("【AuthService.Register】用户信息获取成功",
		"userID", user.ID,
		"username", username,
		"getUserLatency", getUserLatency)

	// 生成JWT token
	s.logger.Debug("【AuthService.Register】生成JWT token", "userID", user.ID)
	tokenGenStart := time.Now()
	token, err := s.generateJWT(user.ID, user.Username)
	tokenGenLatency := time.Since(tokenGenStart)

	if err != nil {
		s.logger.Error("【AuthService.Register】生成JWT token失败",
			"userID", user.ID,
			"error", err.Error(),
			"tokenGenLatency", tokenGenLatency,
			"duration", time.Since(startTime))
		return nil, utils.ErrInternalServerError
	}

	s.logger.Debug("【AuthService.Register】JWT token生成成功",
		"userID", user.ID,
		"tokenLength", len(token),
		"tokenGenLatency", tokenGenLatency)

	// 读取扩展资料（可能为空）
	s.logger.Debug("【AuthService.Register】读取用户扩展资料", "userID", user.ID)
	profileQueryStart := time.Now()
	extra, _ := s.userRepo.GetUserProfile(ctx, user.ID)
	profileQueryLatency := time.Since(profileQueryStart)
	s.logger.Debug("【AuthService.Register】扩展资料读取完成（注册时通常为空）",
		"userID", user.ID,
		"profileQueryLatency", profileQueryLatency)

	// 检查用户是否为管理员
	isAdmin := false
	for _, adminUsername := range s.config.Admin.Usernames {
		if adminUsername == user.Username {
			isAdmin = true
			break
		}
	}

	// 确定用户角色
	role := "user"
	if isAdmin {
		role = "admin"
	}

	s.logger.Debug("【AuthService.Register】角色检查完成",
		"userID", user.ID,
		"username", user.Username,
		"role", role)

	// 返回注册成功响应（匹配前端期望格式）
	response := &models.LoginResponse{
		Code:    201,
		Message: "注册成功",
		Data: struct {
			Token string             `json:"token"`
			User  models.UserProfile `json:"user"`
		}{
			Token: token,
			User: models.UserProfile{
				ID:            user.ID,
				Username:      user.Username,
				Email:         user.Email,
				AuthStatus:    user.AuthStatus,
				AccountStatus: user.AccountStatus,
				AvatarURL:     extra.AvatarURL, // 注册时通常为空
				Nickname:      extra.Nickname,
				Bio:           extra.Bio,
				Role:          role, // 添加角色字段
			},
		},
	}

	s.logger.Info("【AuthService.Register】用户注册成功",
		"userID", user.ID,
		"username", username,
		"email", utils.SanitizeEmail(email),
		"totalDuration", time.Since(startTime),
		"breakdown", map[string]interface{}{
			"usernameCheck": usernameCheckLatency.Milliseconds(),
			"emailCheck":    emailCheckLatency.Milliseconds(),
			"hashPassword":  hashPasswordLatency.Milliseconds(),
			"createUser":    createUserLatency.Milliseconds(),
			"getUser":       getUserLatency.Milliseconds(),
			"tokenGen":      tokenGenLatency.Milliseconds(),
			"profileQuery":  profileQueryLatency.Milliseconds(),
		})

	// 使用 Worker Pool 异步记录注册和登录历史
	if s.historyRepo != nil {
		userID := user.ID
		userName := username
		userIP := clientIP
		userAgentStr := userAgent
		prov := province
		ct := city

		err := utils.SubmitTask(
			fmt.Sprintf("register-history-%d-%d", userID, time.Now().Unix()),
			func(ctx context.Context) error {
				// 记录操作历史
				if err := s.historyRepo.RecordOperationHistory(userID, userName, "注册", "用户注册账号", userIP); err != nil {
					s.logger.Error("记录操作历史失败", "userID", userID, "error", err.Error())
					return err
				}

				// 记录登录历史（注册后自动登录）
				if err := s.historyRepo.RecordLoginHistory(userID, userName, userIP, userAgentStr, prov, ct, 1); err != nil {
					s.logger.Error("记录登录历史失败", "userID", userID, "error", err.Error())
					return err
				}
				s.logger.Info("记录注册登录历史成功",
					"userID", userID,
					"province", prov,
					"city", ct)

				return nil
			},
			10*time.Second,
		)
		if err != nil {
			s.logger.Warn("提交注册历史记录任务失败", "error", err.Error())
		}
	}

	return response, nil
}

// generateJWT 生成JWT token
func (s *AuthService) generateJWT(userID uint, username string) (string, error) {
	s.logger.Debug("【generateJWT】开始生成JWT token",
		"userID", userID,
		"username", username,
		"issuer", s.config.JWT.Issuer,
		"expireHours", s.config.JWT.ExpireHours)

	claims := models.CreateClaims(userID, username, s.config.JWT.Issuer, s.config.JWT.ExpireHours)

	s.logger.Debug("【generateJWT】JWT claims创建完成",
		"userID", userID,
		"subject", claims.Subject,
		"issuer", claims.Issuer,
		"expiresAt", claims.ExpiresAt)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(s.config.JWT.SecretKey))

	if err != nil {
		s.logger.Error("【generateJWT】token签名失败",
			"userID", userID,
			"error", err.Error())
		return "", err
	}

	s.logger.Debug("【generateJWT】JWT token签名成功",
		"userID", userID,
		"tokenLength", len(signedToken))

	return signedToken, nil
}

// ChangePassword 修改密码
func (s *AuthService) ChangePassword(ctx context.Context, userID uint, currentPassword, newPassword string) error {
	startTime := time.Now()
	s.logger.Debug("【AuthService.ChangePassword】开始处理修改密码业务逻辑",
		"userID", userID)

	// 获取用户信息
	s.logger.Debug("【AuthService.ChangePassword】查询用户信息", "userID", userID)
	userQueryStart := time.Now()
	user, err := s.userRepo.GetUserByID(ctx, userID)
	userQueryLatency := time.Since(userQueryStart)

	if err != nil {
		s.logger.Warn("【AuthService.ChangePassword】修改密码失败：用户不存在",
			"userID", userID,
			"userQueryLatency", userQueryLatency,
			"totalDuration", time.Since(startTime))
		return utils.ErrUserNotFound
	}

	s.logger.Debug("【AuthService.ChangePassword】用户信息查询成功",
		"userID", user.ID,
		"username", user.Username,
		"userQueryLatency", userQueryLatency)

	// 验证当前密码
	s.logger.Debug("【AuthService.ChangePassword】验证当前密码", "userID", userID)
	passwordCheckStart := time.Now()
	passwordValid := utils.CheckPasswordHash(currentPassword, user.PasswordHash)
	passwordCheckLatency := time.Since(passwordCheckStart)

	if !passwordValid {
		s.logger.Warn("【AuthService.ChangePassword】修改密码失败：当前密码错误",
			"userID", userID,
			"passwordCheckLatency", passwordCheckLatency,
			"totalDuration", time.Since(startTime))
		return utils.ErrInvalidCredentials
	}

	s.logger.Debug("【AuthService.ChangePassword】当前密码验证通过",
		"userID", userID,
		"passwordCheckLatency", passwordCheckLatency)

	// 验证新密码强度
	s.logger.Debug("【AuthService.ChangePassword】验证新密码强度",
		"userID", userID,
		"newPasswordLength", len(newPassword))

	if !utils.ValidatePassword(newPassword) {
		s.logger.Warn("【AuthService.ChangePassword】修改密码失败：新密码强度不够",
			"userID", userID,
			"newPasswordLength", len(newPassword),
			"totalDuration", time.Since(startTime))
		return utils.ErrInvalidPassword
	}

	// 加密新密码
	s.logger.Debug("【AuthService.ChangePassword】开始加密新密码",
		"userID", userID,
		"newPasswordLength", len(newPassword))
	hashPasswordStart := time.Now()
	hashedPassword, err := utils.HashPassword(newPassword)
	hashPasswordLatency := time.Since(hashPasswordStart)

	if err != nil {
		s.logger.Error("【AuthService.ChangePassword】新密码加密失败",
			"userID", userID,
			"error", err.Error(),
			"hashPasswordLatency", hashPasswordLatency,
			"totalDuration", time.Since(startTime))
		return utils.ErrInternalServerError
	}

	s.logger.Debug("【AuthService.ChangePassword】新密码加密成功",
		"userID", userID,
		"hashedPasswordLength", len(hashedPassword),
		"hashPasswordLatency", hashPasswordLatency)

	// 更新密码到数据库
	s.logger.Debug("【AuthService.ChangePassword】开始更新密码到数据库", "userID", userID)
	updatePasswordStart := time.Now()
	err = s.userRepo.UpdatePassword(ctx, userID, hashedPassword)
	updatePasswordLatency := time.Since(updatePasswordStart)

	if err != nil {
		s.logger.Error("【AuthService.ChangePassword】更新密码失败",
			"userID", userID,
			"error", err.Error(),
			"updatePasswordLatency", updatePasswordLatency,
			"totalDuration", time.Since(startTime))
		return err
	}

	s.logger.Info("【AuthService.ChangePassword】密码修改成功",
		"userID", userID,
		"totalDuration", time.Since(startTime),
		"breakdown", map[string]interface{}{
			"userQuery":      userQueryLatency.Milliseconds(),
			"passwordCheck":  passwordCheckLatency.Milliseconds(),
			"hashPassword":   hashPasswordLatency.Milliseconds(),
			"updatePassword": updatePasswordLatency.Milliseconds(),
		})

	return nil
}

// ForgotPassword 忘记密码 - 生成重置token
func (s *AuthService) ForgotPassword(ctx context.Context, email string) (string, error) {
	startTime := time.Now()
	s.logger.Debug("【AuthService.ForgotPassword】开始处理忘记密码业务逻辑",
		"email", utils.SanitizeEmail(email))

	// 检查邮箱是否存在
	s.logger.Debug("【AuthService.ForgotPassword】检查邮箱是否存在",
		"email", utils.SanitizeEmail(email))
	emailCheckStart := time.Now()
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	emailCheckLatency := time.Since(emailCheckStart)

	if err != nil {
		s.logger.Warn("【AuthService.ForgotPassword】邮箱不存在",
			"email", utils.SanitizeEmail(email),
			"emailCheckLatency", emailCheckLatency,
			"totalDuration", time.Since(startTime))
		// 为了安全起见，不直接告诉用户邮箱不存在
		return "", utils.ErrUserNotFound
	}

	s.logger.Debug("【AuthService.ForgotPassword】邮箱验证成功",
		"userID", user.ID,
		"email", utils.SanitizeEmail(email),
		"emailCheckLatency", emailCheckLatency)

	// 生成重置token（使用UUID或随机字符串）
	s.logger.Debug("【AuthService.ForgotPassword】生成重置token")
	tokenGenStart := time.Now()
	resetToken := generateResetToken()
	tokenGenLatency := time.Since(tokenGenStart)

	s.logger.Debug("【AuthService.ForgotPassword】重置token生成成功",
		"tokenLength", len(resetToken),
		"tokenGenLatency", tokenGenLatency)

	// 设置过期时间（15分钟）
	expiresAt := time.Now().Add(15 * time.Minute)

	// 保存token到数据库
	tokenRecord := &models.PasswordResetToken{
		Email:     email,
		Token:     resetToken,
		ExpiresAt: expiresAt,
		Used:      false,
		CreatedAt: time.Now(),
	}

	s.logger.Debug("【AuthService.ForgotPassword】保存重置token到数据库",
		"email", utils.SanitizeEmail(email),
		"expiresAt", expiresAt)
	saveTokenStart := time.Now()
	err = s.userRepo.CreatePasswordResetToken(ctx, tokenRecord)
	saveTokenLatency := time.Since(saveTokenStart)

	if err != nil {
		s.logger.Error("【AuthService.ForgotPassword】保存重置token失败",
			"email", utils.SanitizeEmail(email),
			"error", err.Error(),
			"saveTokenLatency", saveTokenLatency,
			"totalDuration", time.Since(startTime))
		return "", utils.ErrInternalServerError
	}

	s.logger.Info("【AuthService.ForgotPassword】忘记密码处理成功",
		"email", utils.SanitizeEmail(email),
		"tokenID", tokenRecord.ID,
		"expiresAt", expiresAt,
		"totalDuration", time.Since(startTime),
		"breakdown", map[string]interface{}{
			"emailCheck": emailCheckLatency.Milliseconds(),
			"tokenGen":   tokenGenLatency.Milliseconds(),
			"saveToken":  saveTokenLatency.Milliseconds(),
		})

	// 返回token（在实际生产环境中应该发送邮件）
	return resetToken, nil
}

// ResetPassword 重置密码
func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	startTime := time.Now()
	s.logger.Debug("【AuthService.ResetPassword】开始处理重置密码业务逻辑")

	// 验证token
	s.logger.Debug("【AuthService.ResetPassword】验证重置token")
	tokenCheckStart := time.Now()
	tokenRecord, err := s.userRepo.GetPasswordResetToken(ctx, token)
	tokenCheckLatency := time.Since(tokenCheckStart)

	if err != nil {
		s.logger.Warn("【AuthService.ResetPassword】重置token无效",
			"error", err.Error(),
			"tokenCheckLatency", tokenCheckLatency,
			"totalDuration", time.Since(startTime))
		return utils.ErrInvalidToken
	}

	s.logger.Debug("【AuthService.ResetPassword】token验证成功",
		"tokenID", tokenRecord.ID,
		"email", utils.SanitizeEmail(tokenRecord.Email),
		"expiresAt", tokenRecord.ExpiresAt,
		"tokenCheckLatency", tokenCheckLatency)

	// 检查token是否过期
	if time.Now().After(tokenRecord.ExpiresAt) {
		s.logger.Warn("【AuthService.ResetPassword】重置token已过期",
			"tokenID", tokenRecord.ID,
			"email", utils.SanitizeEmail(tokenRecord.Email),
			"expiresAt", tokenRecord.ExpiresAt,
			"totalDuration", time.Since(startTime))
		return utils.ErrTokenExpired
	}

	s.logger.Debug("【AuthService.ResetPassword】token未过期")

	// 获取用户信息
	s.logger.Debug("【AuthService.ResetPassword】获取用户信息",
		"email", utils.SanitizeEmail(tokenRecord.Email))
	getUserStart := time.Now()
	user, err := s.userRepo.GetUserByEmail(ctx, tokenRecord.Email)
	getUserLatency := time.Since(getUserStart)

	if err != nil {
		s.logger.Error("【AuthService.ResetPassword】获取用户信息失败",
			"email", utils.SanitizeEmail(tokenRecord.Email),
			"error", err.Error(),
			"getUserLatency", getUserLatency,
			"totalDuration", time.Since(startTime))
		return utils.ErrUserNotFound
	}

	s.logger.Debug("【AuthService.ResetPassword】用户信息获取成功",
		"userID", user.ID,
		"getUserLatency", getUserLatency)

	// 验证新密码强度
	s.logger.Debug("【AuthService.ResetPassword】验证新密码强度",
		"newPasswordLength", len(newPassword))

	if !utils.ValidatePassword(newPassword) {
		s.logger.Warn("【AuthService.ResetPassword】新密码强度不够",
			"newPasswordLength", len(newPassword),
			"totalDuration", time.Since(startTime))
		return utils.ErrInvalidPassword
	}

	// 加密新密码
	s.logger.Debug("【AuthService.ResetPassword】加密新密码")
	hashPasswordStart := time.Now()
	hashedPassword, err := utils.HashPassword(newPassword)
	hashPasswordLatency := time.Since(hashPasswordStart)

	if err != nil {
		s.logger.Error("【AuthService.ResetPassword】密码加密失败",
			"error", err.Error(),
			"hashPasswordLatency", hashPasswordLatency,
			"totalDuration", time.Since(startTime))
		return utils.ErrInternalServerError
	}

	s.logger.Debug("【AuthService.ResetPassword】密码加密成功",
		"hashedPasswordLength", len(hashedPassword),
		"hashPasswordLatency", hashPasswordLatency)

	// 更新密码
	s.logger.Debug("【AuthService.ResetPassword】更新用户密码",
		"userID", user.ID)
	updatePasswordStart := time.Now()
	err = s.userRepo.UpdatePassword(ctx, user.ID, hashedPassword)
	updatePasswordLatency := time.Since(updatePasswordStart)

	if err != nil {
		s.logger.Error("【AuthService.ResetPassword】更新密码失败",
			"userID", user.ID,
			"error", err.Error(),
			"updatePasswordLatency", updatePasswordLatency,
			"totalDuration", time.Since(startTime))
		return err
	}

	s.logger.Debug("【AuthService.ResetPassword】密码更新成功",
		"userID", user.ID,
		"updatePasswordLatency", updatePasswordLatency)

	// 标记token为已使用
	s.logger.Debug("【AuthService.ResetPassword】标记token为已使用",
		"tokenID", tokenRecord.ID)
	markTokenStart := time.Now()
	err = s.userRepo.MarkPasswordResetTokenAsUsed(ctx, tokenRecord.ID)
	markTokenLatency := time.Since(markTokenStart)

	if err != nil {
		s.logger.Error("【AuthService.ResetPassword】标记token失败",
			"tokenID", tokenRecord.ID,
			"error", err.Error(),
			"markTokenLatency", markTokenLatency)
		// 不影响主流程
	}

	s.logger.Info("【AuthService.ResetPassword】密码重置成功",
		"userID", user.ID,
		"email", utils.SanitizeEmail(tokenRecord.Email),
		"tokenID", tokenRecord.ID,
		"totalDuration", time.Since(startTime),
		"breakdown", map[string]interface{}{
			"tokenCheck":     tokenCheckLatency.Milliseconds(),
			"getUser":        getUserLatency.Milliseconds(),
			"hashPassword":   hashPasswordLatency.Milliseconds(),
			"updatePassword": updatePasswordLatency.Milliseconds(),
			"markToken":      markTokenLatency.Milliseconds(),
		})

	return nil
}

// generateResetToken 生成加密安全的重置token
func generateResetToken() string {
	// 使用crypto/rand生成64位加密安全的随机token（Base64编码后约43个字符）
	const tokenBytes = 48 // 生成48字节随机数据，Base64编码后为64个字符
	b := make([]byte, tokenBytes)

	// 使用crypto/rand而不是math/rand，确保密码学安全
	_, err := randRead(b)
	if err != nil {
		// 如果crypto/rand失败，使用UUID作为后备方案
		return fmt.Sprintf("%d-%d-%d-%d",
			time.Now().UnixNano(),
			time.Now().Unix(),
			time.Now().UnixMicro(),
			time.Now().UnixMilli())
	}

	// Base64 URL编码（替换+/为-_，移除=填充）
	token := base64.URLEncoding.EncodeToString(b)
	token = strings.ReplaceAll(token, "=", "")

	return token
}

// randRead 从crypto/rand读取随机字节（封装以便错误处理）
func randRead(b []byte) (int, error) {
	return rand.Read(b)
}
