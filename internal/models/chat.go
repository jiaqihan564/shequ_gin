package models

import "time"

// ChatMessage 聊天消息
type ChatMessage struct {
	ID          uint      `json:"id" db:"id"`
	UserID      uint      `json:"user_id" db:"user_id"`
	Username    string    `json:"username" db:"username"`
	Nickname    string    `json:"nickname" db:"nickname"`
	Avatar      string    `json:"avatar" db:"avatar"`
	Content     string    `json:"content" db:"content"`
	MessageType int       `json:"message_type" db:"message_type"` // 1-普通消息，2-系统消息
	SendTime    time.Time `json:"send_time" db:"send_time"`
	IPAddress   string    `json:"ip_address,omitempty" db:"ip_address"`
	Status      int       `json:"status" db:"status"` // 0-已删除，1-正常
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// SendMessageRequest 发送消息请求
type SendMessageRequest struct {
	Content string `json:"content" binding:"required,min=1,max=500"`
}

// SendMessageResponse 发送消息响应
type SendMessageResponse struct {
	MessageID uint      `json:"message_id"`
	SendTime  time.Time `json:"send_time"`
}

// GetMessagesResponse 获取消息列表响应
type GetMessagesResponse struct {
	Messages []ChatMessage `json:"messages"`
	HasMore  bool          `json:"has_more"`
}

// OnlineUser 在线用户
type OnlineUser struct {
	ID            uint      `json:"id" db:"id"`
	UserID        uint      `json:"user_id" db:"user_id"`
	Username      string    `json:"username" db:"username"`
	LastHeartbeat time.Time `json:"last_heartbeat" db:"last_heartbeat"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// OnlineCountResponse 在线用户数响应
type OnlineCountResponse struct {
	Count int `json:"count"`
}
