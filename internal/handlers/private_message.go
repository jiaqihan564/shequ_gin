package handlers

import (
	"context"
	"strconv"

	"gin/internal/models"
	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// PrivateMessageHandler 私信处理器
type PrivateMessageHandler struct {
	msgRepo  *services.PrivateMessageRepository
	userRepo *services.UserRepository
	logger   utils.Logger
}

// NewPrivateMessageHandler 创建私信处理器
func NewPrivateMessageHandler(msgRepo *services.PrivateMessageRepository, userRepo *services.UserRepository) *PrivateMessageHandler {
	return &PrivateMessageHandler{
		msgRepo:  msgRepo,
		userRepo: userRepo,
		logger:   utils.GetLogger(),
	}
}

// getUserInfo 获取用户信息（辅助方法）
func (h *PrivateMessageHandler) getUserInfo(ctx context.Context, userID uint) (models.ConversationUser, error) {
	user, err := h.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return models.ConversationUser{}, err
	}

	// 获取用户扩展资料
	profile, _ := h.userRepo.GetUserProfile(ctx, userID)

	nickname := user.Username
	avatar := ""

	if profile != nil {
		if profile.Nickname != "" {
			nickname = profile.Nickname
		}
		avatar = profile.AvatarURL
	}

	return models.ConversationUser{
		ID:       user.ID,
		Username: user.Username,
		Nickname: nickname,
		Avatar:   avatar,
	}, nil
}

// GetConversations 获取会话列表
func (h *PrivateMessageHandler) GetConversations(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	ctx := c.Request.Context()
	conversations, err := h.msgRepo.GetUserConversations(ctx, userID)
	if err != nil {
		h.logger.Error("获取会话列表失败", "userID", userID, "error", err.Error())
		utils.ErrorResponse(c, 500, "获取会话列表失败")
		return
	}

	// 构建响应，包含对方用户信息
	var response []models.ConversationResponse
	totalUnread := 0

	for _, conv := range conversations {
		// 确定对方用户ID
		otherUserID := conv.User2ID
		unreadCount := conv.User1Unread
		if userID == conv.User2ID {
			otherUserID = conv.User1ID
			unreadCount = conv.User2Unread
		}

		// 获取对方用户基本信息
		userInfo, err := h.getUserInfo(ctx, otherUserID)
		if err != nil {
			h.logger.Warn("获取用户信息失败", "userID", otherUserID, "error", err.Error())
			continue
		}

		response = append(response, models.ConversationResponse{
			ID:              conv.ID,
			OtherUser:       userInfo,
			LastMessage:     conv.LastMessageContent,
			LastMessageTime: conv.LastMessageTime,
			UnreadCount:     unreadCount,
			CreatedAt:       conv.CreatedAt,
		})

		totalUnread += unreadCount
	}

	h.logger.Info("获取会话列表成功", "userID", userID, "count", len(response))
	utils.SuccessResponse(c, 200, "获取成功", models.ConversationsListResponse{
		Conversations: response,
		TotalUnread:   totalUnread,
	})
}

// GetMessages 获取会话消息
func (h *PrivateMessageHandler) GetMessages(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	conversationIDStr := c.Param("id")
	conversationID, err := strconv.ParseUint(conversationIDStr, 10, 32)
	if err != nil {
		utils.BadRequestResponse(c, "无效的会话ID")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit > 100 {
		limit = 100
	}

	ctx := c.Request.Context()

	// 验证用户是否是该会话的参与者
	conv, err := h.msgRepo.GetConversationByID(ctx, uint(conversationID))
	if err != nil {
		utils.NotFoundResponse(c, "会话不存在")
		return
	}

	if conv.User1ID != userID && conv.User2ID != userID {
		utils.ForbiddenResponse(c, "无权访问该会话")
		return
	}

	// 获取消息列表
	messages, err := h.msgRepo.GetConversationMessages(ctx, uint(conversationID), limit)
	if err != nil {
		h.logger.Error("获取消息失败", "conversationID", conversationID, "error", err.Error())
		utils.ErrorResponse(c, 500, "获取消息失败")
		return
	}

	// 构建响应
	var response []models.MessageResponse
	for _, msg := range messages {
		// 获取发送者和接收者信息
		sender, _ := h.getUserInfo(ctx, msg.SenderID)
		receiver, _ := h.getUserInfo(ctx, msg.ReceiverID)

		response = append(response, models.MessageResponse{
			ID:             msg.ID,
			ConversationID: msg.ConversationID,
			Sender:         sender,
			Receiver:       receiver,
			Content:        msg.Content,
			IsRead:         msg.IsRead,
			IsSelf:         msg.SenderID == userID,
			CreatedAt:      msg.CreatedAt,
		})
	}

	// 标记消息为已读
	go func() {
		_ = h.msgRepo.MarkAsRead(context.Background(), uint(conversationID), userID)
	}()

	h.logger.Info("获取消息成功", "conversationID", conversationID, "count", len(response))
	utils.SuccessResponse(c, 200, "获取成功", models.MessagesListResponse{
		Messages: response,
		Total:    len(response),
	})
}

// SendMessage 发送消息
func (h *PrivateMessageHandler) SendMessage(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	var req models.SendPrivateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, "请求参数错误")
		return
	}

	// 不能给自己发消息
	if req.ReceiverID == userID {
		utils.ValidationErrorResponse(c, "不能给自己发送消息")
		return
	}

	ctx := c.Request.Context()

	// 验证接收者是否存在
	receiver, err := h.userRepo.GetUserByID(ctx, req.ReceiverID)
	if err != nil || receiver == nil {
		utils.NotFoundResponse(c, "接收者不存在")
		return
	}

	// 发送消息
	message, err := h.msgRepo.SendMessage(ctx, userID, req.ReceiverID, req.Content)
	if err != nil {
		h.logger.Error("发送消息失败", "senderID", userID, "receiverID", req.ReceiverID, "error", err.Error())
		utils.ErrorResponse(c, 500, "发送消息失败")
		return
	}

	h.logger.Info("发送消息成功", "messageID", message.ID, "senderID", userID, "receiverID", req.ReceiverID)
	utils.SuccessResponse(c, 201, "发送成功", models.SendPrivateMessageResponse{
		MessageID:      message.ID,
		ConversationID: message.ConversationID,
	})
}

// GetUnreadCount 获取未读消息数
func (h *PrivateMessageHandler) GetUnreadCount(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	ctx := c.Request.Context()
	count, err := h.msgRepo.GetUnreadCount(ctx, userID)
	if err != nil {
		h.logger.Error("获取未读数失败", "userID", userID, "error", err.Error())
		utils.ErrorResponse(c, 500, "获取未读数失败")
		return
	}

	utils.SuccessResponse(c, 200, "获取成功", gin.H{
		"unread_count": count,
	})
}

// StartConversation 开始与指定用户的会话
func (h *PrivateMessageHandler) StartConversation(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	otherUserIDStr := c.Param("userId")
	otherUserID, err := strconv.ParseUint(otherUserIDStr, 10, 32)
	if err != nil {
		utils.BadRequestResponse(c, "无效的用户ID")
		return
	}

	if uint(otherUserID) == userID {
		utils.ValidationErrorResponse(c, "不能和自己开始会话")
		return
	}

	ctx := c.Request.Context()

	// 验证对方用户是否存在
	otherUser, err := h.userRepo.GetUserByID(ctx, uint(otherUserID))
	if err != nil || otherUser == nil {
		utils.NotFoundResponse(c, "用户不存在")
		return
	}

	// 获取或创建会话
	conv, err := h.msgRepo.GetOrCreateConversation(ctx, userID, uint(otherUserID))
	if err != nil {
		h.logger.Error("创建会话失败", "user1", userID, "user2", otherUserID, "error", err.Error())
		utils.ErrorResponse(c, 500, "创建会话失败")
		return
	}

	h.logger.Info("开始会话", "conversationID", conv.ID, "userID", userID, "otherUserID", otherUserID)
	utils.SuccessResponse(c, 200, "成功", gin.H{
		"conversation_id": conv.ID,
	})
}
