package handlers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gin/internal/config"
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
	config   *config.Config
}

// NewPrivateMessageHandler 创建私信处理器
func NewPrivateMessageHandler(msgRepo *services.PrivateMessageRepository, userRepo *services.UserRepository, cfg *config.Config) *PrivateMessageHandler {
	return &PrivateMessageHandler{
		msgRepo:  msgRepo,
		userRepo: userRepo,
		logger:   utils.GetLogger(),
		config:   cfg,
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

// batchGetUserInfo 批量获取用户信息（解决N+1问题）
func (h *PrivateMessageHandler) batchGetUserInfo(ctx context.Context, userIDs []uint) (map[uint]models.ConversationUser, error) {
	if len(userIDs) == 0 {
		return make(map[uint]models.ConversationUser), nil
	}

	// 去重（预分配容量）
	uniqueIDs := make(map[uint]bool, len(userIDs))
	for _, id := range userIDs {
		uniqueIDs[id] = true
	}

	result := make(map[uint]models.ConversationUser, len(userIDs)) // 预分配容量

	// 批量查询用户基本信息和资料（使用JOIN）
	ids := make([]uint, 0, len(uniqueIDs))
	for id := range uniqueIDs {
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return result, nil
	}

	// 批量查询用户信息
	// 使用缓存提升性能
	for _, id := range ids {
		info, err := h.getUserInfo(ctx, id)
		if err != nil {
			h.logger.Warn("获取用户信息失败", "userID", id, "error", err.Error())
			continue
		}
		result[id] = info
	}

	return result, nil
}

// GetConversations 获取会话列表
func (h *PrivateMessageHandler) GetConversations(c *gin.Context) {
	userID, isOK := getUserIDOrFail(c)
	if !isOK {
		return
	}

	ctx := c.Request.Context()
	conversations, err := h.msgRepo.GetUserConversations(ctx, userID)
	if err != nil {
		h.logger.Error("获取会话列表失败", "userID", userID, "error", err.Error())
		utils.ErrorResponse(c, 500, "获取会话列表失败")
		return
	}

	// 批量获取所有对方用户信息（解决N+1问题）
	otherUserIDs := make([]uint, 0, len(conversations))
	for _, conv := range conversations {
		otherUserID := conv.User2ID
		if userID == conv.User2ID {
			otherUserID = conv.User1ID
		}
		otherUserIDs = append(otherUserIDs, otherUserID)
	}

	// 批量查询用户信息
	userInfoMap := make(map[uint]models.ConversationUser)
	if len(otherUserIDs) > 0 {
		users, err := h.batchGetUserInfo(ctx, otherUserIDs)
		if err != nil {
			h.logger.Warn("批量获取用户信息失败", "error", err.Error())
		} else {
			userInfoMap = users
		}
	}

	// 构建响应
	response := make([]models.ConversationResponse, 0, len(conversations))
	totalUnread := 0

	for _, conv := range conversations {
		// 确定对方用户ID和未读数
		otherUserID := conv.User2ID
		unreadCount := conv.User1Unread
		if userID == conv.User2ID {
			otherUserID = conv.User1ID
			unreadCount = conv.User2Unread
		}

		// 从map获取用户信息
		userInfo, ok := userInfoMap[otherUserID]
		if !ok {
			h.logger.Warn("找不到用户信息", "userID", otherUserID)
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
	userID, isOK := getUserIDOrFail(c)
	if !isOK {
		return
	}

	conversationIDStr := c.Param("id")
	conversationID, err := strconv.ParseUint(conversationIDStr, 10, 32)
	if err != nil {
		utils.BadRequestResponse(c, "无效的会话ID")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(h.config.Pagination.DefaultLimit)))
	if limit > h.config.Pagination.MaxLimit {
		limit = h.config.Pagination.MaxLimit
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

	// 使用Worker Pool标记消息为已读（避免goroutine泄漏）
	taskID := fmt.Sprintf("mark_read_%d_%d", conversationID, userID)
	_ = utils.SubmitTask(taskID, func(taskCtx context.Context) error {
		h.logger.Info("Marking messages as read (async task)",
			"conversationID", conversationID,
			"userID", userID)

		hasUpdates, err := h.msgRepo.MarkAsRead(taskCtx, uint(conversationID), userID)
		if err != nil {
			h.logger.Error("Failed to mark messages as read",
				"conversationID", conversationID,
				"userID", userID,
				"error", err.Error())
			return err
		}

		// 只有实际标记了消息时才通知发送者（避免重复通知）
		if hasUpdates {
			// 通知对方用户消息已读
			otherUserID := conv.User1ID
			if conv.User1ID == userID {
				otherUserID = conv.User2ID
			}

			h.logger.Info("Notifying sender about message read",
				"senderID", otherUserID,
				"readerID", userID,
				"conversationID", conversationID,
				"updatedMessages", hasUpdates)

			NotifyMessageRead(otherUserID, uint(conversationID), userID)
		} else {
			h.logger.Debug("No unread messages to mark, skipping notification",
				"conversationID", conversationID,
				"userID", userID)
		}
		return nil
	}, time.Duration(h.config.AsyncTasks.MessageMarkReadTimeout)*time.Second)

	h.logger.Info("获取消息成功", "conversationID", conversationID, "count", len(response))
	utils.SuccessResponse(c, 200, "获取成功", models.MessagesListResponse{
		Messages: response,
		Total:    len(response),
	})
}

// SendMessage 发送消息
func (h *PrivateMessageHandler) SendMessage(c *gin.Context) {
	userID, isOK := getUserIDOrFail(c)
	if !isOK {
		return
	}

	var req models.SendPrivateMessageRequest
	if !bindJSONOrFail(c, &req, nil, "") {
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

	// 通过WebSocket实时推送消息给接收者
	sender, _ := h.getUserInfo(ctx, userID)
	receiverInfo, _ := h.getUserInfo(ctx, req.ReceiverID)

	messageResponse := models.MessageResponse{
		ID:             message.ID,
		ConversationID: message.ConversationID,
		Sender:         sender,
		Receiver:       receiverInfo,
		Content:        message.Content,
		IsRead:         false,
		IsSelf:         false, // 对接收者来说不是自己发的
		CreatedAt:      message.CreatedAt,
	}

	// 发送WebSocket通知给接收者
	NotifyPrivateMessage(req.ReceiverID, &messageResponse)

	utils.SuccessResponse(c, 201, "发送成功", models.SendPrivateMessageResponse{
		MessageID:      message.ID,
		ConversationID: message.ConversationID,
	})
}

// MarkConversationAsRead 标记会话消息为已读
func (h *PrivateMessageHandler) MarkConversationAsRead(c *gin.Context) {
	userID, isOK := getUserIDOrFail(c)
	if !isOK {
		return
	}

	conversationIDStr := c.Param("id")
	conversationID, err := strconv.ParseUint(conversationIDStr, 10, 32)
	if err != nil {
		utils.BadRequestResponse(c, "无效的会话ID")
		return
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

	// 标记消息为已读
	hasUpdates, err := h.msgRepo.MarkAsRead(ctx, uint(conversationID), userID)
	if err != nil {
		h.logger.Error("标记已读失败", "conversationID", conversationID, "userID", userID, "error", err.Error())
		utils.ErrorResponse(c, 500, "标记已读失败")
		return
	}

	// 只有实际标记了消息时才通知发送者
	if hasUpdates {
		// 通知对方用户消息已读
		otherUserID := conv.User1ID
		if conv.User1ID == userID {
			otherUserID = conv.User2ID
		}

		h.logger.Info("Messages marked as read",
			"conversationID", conversationID,
			"readerID", userID,
			"senderID", otherUserID,
			"updatedMessages", hasUpdates)

		NotifyMessageRead(otherUserID, uint(conversationID), userID)
	} else {
		h.logger.Debug("No unread messages, skipping notification",
			"conversationID", conversationID,
			"userID", userID)
	}

	utils.SuccessResponse(c, 200, "标记成功", nil)
}

// GetUnreadCount 获取未读消息数
func (h *PrivateMessageHandler) GetUnreadCount(c *gin.Context) {
	userID, isOK := getUserIDOrFail(c)
	if !isOK {
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
	userID, isOK := getUserIDOrFail(c)
	if !isOK {
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
