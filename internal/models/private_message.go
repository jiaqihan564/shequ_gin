package models

import "time"

// PrivateConversation 私信会话
type PrivateConversation struct {
	ID                 uint       `json:"id" db:"id"`
	User1ID            uint       `json:"user1_id" db:"user1_id"`
	User2ID            uint       `json:"user2_id" db:"user2_id"`
	LastMessageID      *uint      `json:"last_message_id" db:"last_message_id"`
	LastMessageContent string     `json:"last_message_content" db:"last_message_content"`
	LastMessageTime    *time.Time `json:"last_message_time" db:"last_message_time"`
	User1Unread        int        `json:"user1_unread" db:"user1_unread"`
	User2Unread        int        `json:"user2_unread" db:"user2_unread"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
}

// PrivateMessage 私信消息
type PrivateMessage struct {
	ID             uint      `json:"id" db:"id"`
	ConversationID uint      `json:"conversation_id" db:"conversation_id"`
	SenderID       uint      `json:"sender_id" db:"sender_id"`
	ReceiverID     uint      `json:"receiver_id" db:"receiver_id"`
	Content        string    `json:"content" db:"content"`
	IsRead         bool      `json:"is_read" db:"is_read"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// ========== 响应 DTO ==========

// ConversationUser 会话中的用户信息
type ConversationUser struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// ConversationResponse 会话响应
type ConversationResponse struct {
	ID              uint             `json:"id"`
	OtherUser       ConversationUser `json:"other_user"` // 对方用户信息
	LastMessage     string           `json:"last_message"`
	LastMessageTime *time.Time       `json:"last_message_time"`
	UnreadCount     int              `json:"unread_count"`
	CreatedAt       time.Time        `json:"created_at"`
}

// MessageResponse 消息响应
type MessageResponse struct {
	ID             uint             `json:"id"`
	ConversationID uint             `json:"conversation_id"`
	Sender         ConversationUser `json:"sender"`
	Receiver       ConversationUser `json:"receiver"`
	Content        string           `json:"content"`
	IsRead         bool             `json:"is_read"`
	IsSelf         bool             `json:"is_self"` // 是否是自己发送的
	CreatedAt      time.Time        `json:"created_at"`
}

// ConversationsListResponse 会话列表响应
type ConversationsListResponse struct {
	Conversations []ConversationResponse `json:"conversations"`
	TotalUnread   int                    `json:"total_unread"`
}

// MessagesListResponse 消息列表响应
type MessagesListResponse struct {
	Messages []MessageResponse `json:"messages"`
	Total    int               `json:"total"`
}

// SendPrivateMessageRequest 发送私信请求
type SendPrivateMessageRequest struct {
	ReceiverID uint   `json:"receiver_id" binding:"required"`
	Content    string `json:"content" binding:"required,min=1,max=1000"`
}

// SendPrivateMessageResponse 发送私信响应
type SendPrivateMessageResponse struct {
	MessageID      uint `json:"message_id"`
	ConversationID uint `json:"conversation_id"`
}
