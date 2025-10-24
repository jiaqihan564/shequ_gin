package handlers

import (
	"context"
	"strconv"
	"time"

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

	// 从数据库获取用户基本信息
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	user, err := h.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Error("获取用户信息失败", "userID", userID, "error", err.Error())
		utils.ErrorResponse(c, 500, "获取用户信息失败")
		return
	}

	// 获取用户扩展信息（昵称和头像）
	profile, err := h.userRepo.GetUserProfile(ctx, userID)
	nickname := ""
	avatar := ""
	if err == nil && profile != nil {
		nickname = profile.Nickname
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

	// 更新在线用户心跳
	_ = h.chatRepo.UpdateOnlineUser(userID, user.Username)

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

	// 更新在线用户心跳（轮询视为在线）
	userID, err := utils.GetUserIDFromContext(c)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if user, err := h.userRepo.GetUserByID(ctx, userID); err == nil {
			_ = h.chatRepo.UpdateOnlineUser(userID, user.Username)
		}
	}

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

// GetOnlineCount 获取在线用户数
func (h *ChatHandler) GetOnlineCount(c *gin.Context) {
	count, err := h.chatRepo.GetOnlineCount()
	if err != nil {
		h.logger.Error("获取在线用户数失败", "error", err.Error())
		utils.ErrorResponse(c, 500, "获取在线用户数失败")
		return
	}

	utils.SuccessResponse(c, 200, "获取成功", models.OnlineCountResponse{
		Count: count,
	})
}
