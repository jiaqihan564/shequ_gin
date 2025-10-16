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

// GetProfile 获取用户信息
func (h *UserHandler) GetProfile(c *gin.Context) {
	startTime := time.Now()
	clientIP := c.ClientIP()

	h.logger.Debug("【GetProfile】开始处理获取用户信息请求",
		"ip", clientIP,
		"path", c.Request.URL.Path)

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		h.logger.Warn("【GetProfile】获取用户信息失败：用户未认证",
			"ip", clientIP,
			"error", err.Error(),
			"duration", time.Since(startTime))
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	h.logger.Info("【GetProfile】获取用户信息请求",
		"userID", userID,
		"ip", clientIP)

	// 调用服务层获取用户信息
	ctx := c.Request.Context()
	getUserStart := time.Now()
	user, err := h.userService.GetUserByID(ctx, userID)
	getUserLatency := time.Since(getUserStart)

	if err != nil {
		h.logger.Warn("【GetProfile】获取用户信息失败",
			"userID", userID,
			"error", err.Error(),
			"ip", clientIP,
			"getUserLatency", getUserLatency,
			"duration", time.Since(startTime))
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, err.Error())
		return
	}

	h.logger.Debug("【GetProfile】用户信息查询成功",
		"userID", userID,
		"username", user.Username,
		"getUserLatency", getUserLatency)

	// 获取扩展资料（头像、昵称、简介）
	profileStart := time.Now()
	extra, _ := h.userService.GetUserProfile(ctx, userID)
	profileLatency := time.Since(profileStart)

	h.logger.Debug("【GetProfile】用户扩展资料查询完成",
		"userID", userID,
		"hasNickname", extra.Nickname != "",
		"hasBio", extra.Bio != "",
		"profileLatency", profileLatency)

	// 构建带时间戳的头像URL（防止缓存）
	avatarURL := h.buildAvatarURL(user.Username)

	h.logger.Info("【GetProfile】获取用户信息成功",
		"userID", userID,
		"username", user.Username,
		"ip", clientIP,
		"totalDuration", time.Since(startTime),
		"breakdown", map[string]interface{}{
			"getUser":    getUserLatency.Milliseconds(),
			"getProfile": profileLatency.Milliseconds(),
		})

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
	startTime := time.Now()
	clientIP := c.ClientIP()

	h.logger.Debug("【UpdateMe】开始处理用户信息更新请求",
		"ip", clientIP,
		"path", c.Request.URL.Path)

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		h.logger.Warn("【UpdateMe】更新用户信息失败：用户未认证",
			"ip", clientIP,
			"error", err.Error(),
			"duration", time.Since(startTime))
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
		h.logger.Warn("【UpdateMe】更新用户信息请求参数错误",
			"userID", userID,
			"error", err.Error(),
			"ip", clientIP,
			"duration", time.Since(startTime))
		utils.ValidationErrorResponse(c, "请求参数错误: "+err.Error())
		return
	}

	h.logger.Info("【UpdateMe】收到用户信息更新请求",
		"userID", userID,
		"hasAvatar", payload.Avatar != "",
		"hasNickname", payload.Profile.Nickname != "",
		"hasBio", payload.Profile.Bio != "",
		"ip", clientIP)

	// 如果有头像URL，验证格式并更新
	if payload.Avatar != "" {
		h.logger.Debug("【UpdateMe】开始验证并更新头像",
			"userID", userID,
			"avatarURL", payload.Avatar)

		if !utils.ValidateURL(payload.Avatar) {
			h.logger.Warn("【UpdateMe】头像URL格式错误",
				"userID", userID,
				"url", payload.Avatar,
				"duration", time.Since(startTime))
			utils.ValidationErrorResponse(c, "无效的头像URL格式")
			return
		}

		updateAvatarStart := time.Now()
		prof := &models.UserExtraProfile{
			UserID:    userID,
			AvatarURL: payload.Avatar,
		}
		err := h.userService.UpdateUserAvatar(c.Request.Context(), prof)
		updateAvatarLatency := time.Since(updateAvatarStart)

		if err != nil {
			h.logger.Error("【UpdateMe】更新头像失败",
				"userID", userID,
				"error", err.Error(),
				"updateAvatarLatency", updateAvatarLatency,
				"duration", time.Since(startTime))
			statusCode := utils.GetHTTPStatusCode(err)
			utils.ErrorResponse(c, statusCode, err.Error())
			return
		}

		h.logger.Debug("【UpdateMe】头像更新成功",
			"userID", userID,
			"updateAvatarLatency", updateAvatarLatency)
	}

	// 如果有个人资料，更新个人资料
	if payload.Profile.Nickname != "" || payload.Profile.Bio != "" {
		h.logger.Debug("【UpdateMe】开始验证并更新个人资料",
			"userID", userID,
			"nickname", payload.Profile.Nickname,
			"bioLength", len(payload.Profile.Bio))

		// 验证昵称和简介
		if payload.Profile.Nickname != "" && !utils.ValidateNickname(payload.Profile.Nickname) {
			h.logger.Warn("【UpdateMe】昵称格式不正确",
				"userID", userID,
				"nickname", payload.Profile.Nickname,
				"duration", time.Since(startTime))
			utils.ValidationErrorResponse(c, "昵称格式不正确，长度应为1-50个字符")
			return
		}
		if payload.Profile.Bio != "" && !utils.ValidateBio(payload.Profile.Bio) {
			h.logger.Warn("【UpdateMe】简介过长",
				"userID", userID,
				"bioLength", len(payload.Profile.Bio),
				"duration", time.Since(startTime))
			utils.ValidationErrorResponse(c, "简介过长，最多500个字符")
			return
		}

		// 先获取当前用户信息（用于历史记录）
		updateProfileStart := time.Now()
		currentUser, _ := h.userService.GetUserByID(c.Request.Context(), userID)
		currentProfile, _ := h.userService.GetUserProfile(c.Request.Context(), userID)

		// 只更新非空字段，保留原有数据
		prof := &models.UserExtraProfile{
			UserID:   userID,
			Nickname: currentProfile.Nickname, // 先用原值
			Bio:      currentProfile.Bio,      // 先用原值
		}

		// 只在payload有值时才更新
		if payload.Profile.Nickname != "" {
			prof.Nickname = utils.SanitizeString(payload.Profile.Nickname)
		}
		if payload.Profile.Bio != "" {
			prof.Bio = utils.SanitizeString(payload.Profile.Bio)
		}

		err := h.userService.UpsertUserProfile(c.Request.Context(), prof)
		updateProfileLatency := time.Since(updateProfileStart)

		if err != nil {
			h.logger.Error("【UpdateMe】更新个人资料失败",
				"userID", userID,
				"error", err.Error(),
				"updateProfileLatency", updateProfileLatency,
				"duration", time.Since(startTime))
			statusCode := utils.GetHTTPStatusCode(err)
			utils.ErrorResponse(c, statusCode, err.Error())
			return
		}

		h.logger.Debug("【UpdateMe】个人资料更新成功",
			"userID", userID,
			"nickname", prof.Nickname,
			"updateProfileLatency", updateProfileLatency)

		// 异步记录资料修改历史
		if h.historyRepo != nil && currentUser != nil {
			username := currentUser.Username
			go func() {
				// 记录昵称修改
				if payload.Profile.Nickname != "" && payload.Profile.Nickname != currentProfile.Nickname {
					h.historyRepo.RecordProfileChange(userID, "nickname", currentProfile.Nickname, prof.Nickname, clientIP)
					h.historyRepo.RecordOperationHistory(userID, username, "修改昵称", fmt.Sprintf("从'%s'改为'%s'", currentProfile.Nickname, prof.Nickname), clientIP)
				}
				// 记录简介修改
				if payload.Profile.Bio != "" && payload.Profile.Bio != currentProfile.Bio {
					h.historyRepo.RecordProfileChange(userID, "bio", currentProfile.Bio, prof.Bio, clientIP)
					h.historyRepo.RecordOperationHistory(userID, username, "修改简介", "修改个人简介", clientIP)
				}
			}()
		}
	}

	h.logger.Info("【UpdateMe】用户信息更新完成，开始返回最新信息",
		"userID", userID,
		"duration", time.Since(startTime))

	// 返回更新后的完整用户信息
	h.returnUserProfile(c, userID)
}

// GetMe 获取当前用户信息（前端统一接口）
func (h *UserHandler) GetMe(c *gin.Context) {
	startTime := time.Now()
	clientIP := c.ClientIP()

	h.logger.Debug("【GetMe】开始处理获取当前用户信息请求",
		"ip", clientIP,
		"path", c.Request.URL.Path)

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		h.logger.Warn("【GetMe】获取用户信息失败：用户未认证",
			"ip", clientIP,
			"error", err.Error(),
			"duration", time.Since(startTime))
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	h.logger.Info("【GetMe】获取当前用户信息请求",
		"userID", userID,
		"ip", clientIP)

	h.returnUserProfile(c, userID)

	h.logger.Debug("【GetMe】请求处理完成",
		"userID", userID,
		"duration", time.Since(startTime))
}

// returnUserProfile 返回用户完整信息（统一格式）
func (h *UserHandler) returnUserProfile(c *gin.Context, userID uint) {
	startTime := time.Now()
	clientIP := c.ClientIP()

	h.logger.Debug("【returnUserProfile】开始构建用户完整信息",
		"userID", userID)

	ctx := c.Request.Context()

	// 获取基本用户信息
	getUserStart := time.Now()
	user, err := h.userService.GetUserByID(ctx, userID)
	getUserLatency := time.Since(getUserStart)

	if err != nil {
		h.logger.Warn("【returnUserProfile】获取用户信息失败",
			"userID", userID,
			"error", err.Error(),
			"ip", clientIP,
			"getUserLatency", getUserLatency,
			"duration", time.Since(startTime))
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, err.Error())
		return
	}

	h.logger.Debug("【returnUserProfile】基本用户信息获取成功",
		"userID", userID,
		"username", user.Username,
		"getUserLatency", getUserLatency)

	// 获取扩展资料（可能为空）
	profileStart := time.Now()
	extra, _ := h.userService.GetUserProfile(ctx, userID)
	profileLatency := time.Since(profileStart)

	h.logger.Debug("【returnUserProfile】扩展资料获取完成",
		"userID", userID,
		"hasExtra", extra != nil,
		"profileLatency", profileLatency)

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
			h.logger.Debug("【returnUserProfile】头像URL已构建",
				"userID", userID,
				"avatarURL", avatarURL)
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

	h.logger.Info("【returnUserProfile】获取用户信息成功",
		"userID", userID,
		"username", user.Username,
		"ip", clientIP,
		"totalDuration", time.Since(startTime),
		"breakdown", map[string]interface{}{
			"getUser":    getUserLatency.Milliseconds(),
			"getProfile": profileLatency.Milliseconds(),
		})

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

	h.logger.Info("获取用户信息成功", "currentUserID", currentUserID, "targetUserID", targetUserID, "username", user.Username, "ip", c.ClientIP())
	utils.SuccessResponse(c, 200, "获取用户信息成功", response)
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
