package handlers

import (
	"strconv"

	"gin/internal/models"
	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// ChatHandler 聊天处理器
type ChatHandler struct {
	chatRepo *services.ChatRepository
	userRepo *services.UserRepository
	logger   utils.Logger
}

// NewChatHandler 创建聊天处理器
func NewChatHandler(chatRepo *services.ChatRepository, userRepo *services.UserRepository) *ChatHandler {
	return &ChatHandler{
		chatRepo: chatRepo,
		userRepo: userRepo,
		logger:   utils.GetLogger(),
	}
}

// SendMessage 发送消息
func (h *ChatHandler) SendMessage(c *gin.Context) {
	userID, isOK := getUserIDOrFail(c)
	if !isOK {
		return
	}

	var req models.SendMessageRequest
	if !bindJSONOrFail(c, &req, nil, "") {
		return
	}

	// 从请求上下文获取，避免重复查询
	ctx := c.Request.Context()

	// 使用缓存获取用户信息（减少数据库查询）
	user, err := h.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Error("获取用户信息失败", "userID", userID, "error", err.Error())
		utils.ErrorResponse(c, 500, "获取用户信息失败")
		return
	}

	// 获取用户扩展信息（昵称和头像）
	profile, _ := h.userRepo.GetUserProfile(ctx, userID)
	nickname := user.Username // 默认使用username
	avatar := ""
	if profile != nil {
		if profile.Nickname != "" {
			nickname = profile.Nickname
		}
		avatar = profile.AvatarURL
	}

	// 获取IP地址
	ipAddress := c.ClientIP()

	// 发送消息
	message, err := h.chatRepo.SendMessage(userID, user.Username, nickname, avatar, req.Content, ipAddress)
	if err != nil {
		h.logger.Error("发送消息失败",
			"userID", userID,
			"error", err.Error())
		utils.ErrorResponse(c, 500, "发送消息失败")
		return
	}

	// Online user heartbeat now handled by WebSocket connection

	utils.SuccessResponse(c, 200, "发送成功", models.SendMessageResponse{
		MessageID: message.ID,
		SendTime:  message.SendTime,
	})
}

// GetMessages 获取消息列表
func (h *ChatHandler) GetMessages(c *gin.Context) {
	// 获取查询参数
	limitStr := c.DefaultQuery("limit", "50")
	beforeIDStr := c.DefaultQuery("before_id", "0")

	limit, _ := strconv.Atoi(limitStr)
	beforeID, _ := strconv.ParseUint(beforeIDStr, 10, 32)

	// 限制单次查询数量
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	messages, err := h.chatRepo.GetMessages(limit, uint(beforeID))
	if err != nil {
		h.logger.Error("获取消息列表失败", "error", err.Error())
		utils.ErrorResponse(c, 500, "获取消息列表失败")
		return
	}

	// 判断是否还有更多消息
	hasMore := len(messages) == limit

	utils.SuccessResponse(c, 200, "获取成功", models.GetMessagesResponse{
		Messages: messages,
		HasMore:  hasMore,
	})
}

// GetNewMessages 获取新消息
func (h *ChatHandler) GetNewMessages(c *gin.Context) {
	afterIDStr := c.DefaultQuery("after_id", "0")
	afterID, _ := strconv.ParseUint(afterIDStr, 10, 32)

	messages, err := h.chatRepo.GetNewMessages(uint(afterID))
	if err != nil {
		h.logger.Error("获取新消息失败", "error", err.Error())
		utils.ErrorResponse(c, 500, "获取新消息失败")
		return
	}

	// Online user heartbeat now handled by WebSocket connection

	utils.SuccessResponse(c, 200, "获取成功", models.GetMessagesResponse{
		Messages: messages,
		HasMore:  false,
	})
}

// DeleteMessage 删除消息
func (h *ChatHandler) DeleteMessage(c *gin.Context) {
	userID, isOK := getUserIDOrFail(c)
	if !isOK {
		return
	}

	messageID, isOK := parseUintParam(c, "id", "消息ID格式错误")
	if !isOK {
		return
	}

	err := h.chatRepo.DeleteMessage(messageID, userID)
	if err != nil {
		h.logger.Error("删除消息失败",
			"messageID", messageID,
			"userID", userID,
			"error", err.Error())
		utils.ErrorResponse(c, 500, "删除消息失败")
		return
	}

	utils.SuccessResponse(c, 200, "删除成功", nil)
}

// GetOnlineCount - DEPRECATED: Use WebSocket ConnectionHub instead
// Kept for backward compatibility with old clients
func (h *ChatHandler) GetOnlineCount(c *gin.Context) {
	h.logger.Warn("DEPRECATED: GetOnlineCount called, use WebSocket instead")
	// Return 0 as this is deprecated
	utils.SuccessResponse(c, 200, "Use WebSocket for real-time online count", models.OnlineCountResponse{
		Count: 0,
	})
}

// UserOffline - DEPRECATED: WebSocket automatically handles disconnections
// Kept for backward compatibility with old clients
func (h *ChatHandler) UserOffline(c *gin.Context) {
	h.logger.Warn("DEPRECATED: UserOffline called, WebSocket handles disconnections automatically")
	utils.SuccessResponse(c, 200, "WebSocket handles disconnections automatically", nil)
}
