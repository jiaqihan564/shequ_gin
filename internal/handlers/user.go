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

// UpdateMe 更新当前用户信息
func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, "未认证")
		return
	}

	var payload struct {
		Avatar  string `json:"avatar"`
		Profile struct {
			Nickname string `json:"nickname"`
			Bio      string `json:"bio"`
			Province string `json:"province"`
			City     string `json:"city"`
		} `json:"profile"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.ValidationErrorResponse(c, "请求参数错误")
		return
	}

	// 更新头像
	if payload.Avatar != "" {
		if !utils.ValidateURL(payload.Avatar) {
			utils.ValidationErrorResponse(c, "无效的头像URL")
			return
		}
		if err := h.userService.UpdateUserAvatar(c.Request.Context(), &models.UserExtraProfile{
			UserID:    userID,
			AvatarURL: payload.Avatar,
		}); err != nil {
			utils.ErrorResponse(c, utils.GetHTTPStatusCode(err), err.Error())
			return
		}
	}

	// 更新个人资料
	if payload.Profile.Nickname != "" || payload.Profile.Bio != "" {
		if payload.Profile.Nickname != "" && !utils.ValidateNickname(payload.Profile.Nickname) {
			utils.ValidationErrorResponse(c, "昵称格式不正确")
			return
		}
		if payload.Profile.Bio != "" && !utils.ValidateBio(payload.Profile.Bio) {
			utils.ValidationErrorResponse(c, "简介过长")
			return
		}

		if err := h.userService.UpsertUserProfile(c.Request.Context(), &models.UserExtraProfile{
			UserID:   userID,
			Nickname: utils.SanitizeString(payload.Profile.Nickname),
			Bio:      utils.SanitizeString(payload.Profile.Bio),
		}); err != nil {
			utils.ErrorResponse(c, utils.GetHTTPStatusCode(err), err.Error())
			return
		}
	}

	// 返回更新后的用户信息
	h.returnUserProfile(c, userID)
}

// GetMe 获取当前用户信息
func (h *UserHandler) GetMe(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, "未认证")
		return
	}
	h.returnUserProfile(c, userID)
}

// returnUserProfile 返回用户完整信息（统一格式）
func (h *UserHandler) returnUserProfile(c *gin.Context, userID uint) {
	// 获取基本用户信息
	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		utils.ErrorResponse(c, utils.GetHTTPStatusCode(err), err.Error())
		return
	}

	// 获取扩展资料（可能为空）
	extra, _ := h.userService.GetUserProfile(c.Request.Context(), userID)

	// 安全获取扩展资料字段
	avatarURL, nickname, bio := "", "", ""
	if extra != nil {
		// 如果数据库中有头像URL，修正并添加时间戳防缓存
		if extra.AvatarURL != "" {
			avatarURL = fmt.Sprintf("%s?t=%d", h.fixAvatarURL(extra.AvatarURL, user.Username), time.Now().Unix())
		}
		nickname, bio = extra.Nickname, extra.Bio
	}

	// 返回前端期望的响应格式
	utils.SuccessResponse(c, 200, "成功", gin.H{
		"id":             user.ID,
		"username":       user.Username,
		"email":          user.Email,
		"avatar":         avatarURL,
		"auth_status":    user.AuthStatus,
		"account_status": user.AccountStatus,
		"profile": gin.H{
			"nickname": nickname,
			"bio":      bio,
			"province": "",
			"city":     "",
		},
		"updatedAt": user.UpdatedAt,
	})
}

// fixAvatarURL 修正头像URL中的base地址
func (h *UserHandler) fixAvatarURL(oldURL, username string) string {
	base := h.config.Assets.PublicBaseURL
	if base == "" || username == "" {
		return oldURL
	}
	if len(base) > 0 && base[len(base)-1] == '/' {
		base = base[:len(base)-1]
	}
	return fmt.Sprintf("%s/%s/avatar.png", base, username)
}
