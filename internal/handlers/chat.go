package handlers

import (
	"strconv"

	"gin/internal/config"
	"gin/internal/models"
	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// ChatHandler 聊天处理器
type ChatHandler struct {
	chatRepo *services.ChatRepository
	userRepo *services.UserRepository
	config   *config.Config
	logger   utils.Logger
}

// NewChatHandler 创建聊天处理器
func NewChatHandler(chatRepo *services.ChatRepository, userRepo *services.UserRepository, cfg *config.Config) *ChatHandler {
	return &ChatHandler{
		chatRepo: chatRepo,
		userRepo: userRepo,
		config:   cfg,
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

	// 使用辅助函数获取用户信息和profile
	userInfo, err := GetUserWithProfile(ctx, h.userRepo, userID)
	if err != nil {
		handleInternalError(c, ErrGetUserFailed, err, h.logger, "userID", userID)
		return
	}

	// 获取IP地址
	ipAddress := c.ClientIP()

	// 发送消息
	message, err := h.chatRepo.SendMessage(userID, userInfo.User.Username, userInfo.Nickname, userInfo.Avatar, req.Content, ipAddress)
	if err != nil {
		handleInternalError(c, ErrSendMessageFailed, err, h.logger,
			"userID", userID,
			"contentLength", len(req.Content))
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
	limitStr := c.DefaultQuery("limit", strconv.Itoa(h.config.Pagination.DefaultLimit))
	beforeIDStr := c.DefaultQuery("before_id", "0")

	limit, _ := strconv.Atoi(limitStr)
	beforeID, _ := strconv.ParseUint(beforeIDStr, 10, 32)

	// 限制单次查询数量
	if limit <= 0 || limit > h.config.Pagination.MaxLimit {
		limit = h.config.Pagination.DefaultLimit
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

