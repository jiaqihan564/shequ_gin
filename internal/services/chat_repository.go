package services

import (
	"context"
	"database/sql"
	"gin/internal/models"
	"gin/internal/utils"
	"time"
)

// ChatRepository 聊天消息仓库
type ChatRepository struct {
	db     *Database
	logger utils.Logger
}

// NewChatRepository 创建聊天消息仓库
func NewChatRepository(db *Database) *ChatRepository {
	return &ChatRepository{
		db:     db,
		logger: utils.GetLogger(),
	}
}

// SendMessage 发送消息
func (r *ChatRepository) SendMessage(userID uint, username, nickname, avatar, content, ipAddress string) (*models.ChatMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	query := `INSERT INTO chat_messages (user_id, username, nickname, avatar, content, message_type, send_time, ip_address, status, created_at)
			  VALUES (?, ?, ?, ?, ?, 1, ?, ?, 1, ?)`

	result, err := r.db.DB.ExecContext(ctx, query, userID, username, nickname, avatar, content, now, ipAddress, now)
	if err != nil {
		r.logger.Error("发送消息失败", "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}

	messageID, err := result.LastInsertId()
	if err != nil {
		r.logger.Error("获取消息ID失败", "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}

	return &models.ChatMessage{
		ID:          uint(messageID),
		UserID:      userID,
		Username:    username,
		Nickname:    nickname,
		Avatar:      avatar,
		Content:     content,
		MessageType: 1,
		SendTime:    now,
		IPAddress:   ipAddress,
		Status:      1,
		CreatedAt:   now,
	}, nil
}

// GetMessages 获取消息列表（分页）
func (r *ChatRepository) GetMessages(limit int, beforeID uint) ([]models.ChatMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var query string
	var rows *sql.Rows
	var err error

	if beforeID > 0 {
		// 获取指定ID之前的消息
		query = `SELECT id, user_id, username, nickname, avatar, content, message_type, send_time, status, created_at
				 FROM chat_messages
				 WHERE status = 1 AND id < ?
				 ORDER BY id DESC
				 LIMIT ?`
		rows, err = r.db.DB.QueryContext(ctx, query, beforeID, limit)
	} else {
		// 获取最新消息
		query = `SELECT id, user_id, username, nickname, avatar, content, message_type, send_time, status, created_at
				 FROM chat_messages
				 WHERE status = 1
				 ORDER BY id DESC
				 LIMIT ?`
		rows, err = r.db.DB.QueryContext(ctx, query, limit)
	}

	if err != nil {
		r.logger.Error("获取消息列表失败", "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}
	defer rows.Close()

	// 预分配slice容量（性能优化）
	messages := make([]models.ChatMessage, 0, limit)
	for rows.Next() {
		var msg models.ChatMessage
		if err := rows.Scan(&msg.ID, &msg.UserID, &msg.Username, &msg.Nickname, &msg.Avatar,
			&msg.Content, &msg.MessageType, &msg.SendTime, &msg.Status, &msg.CreatedAt); err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	// 反转顺序，让最旧的消息在前面
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// GetNewMessages 获取新消息（指定ID之后的）
func (r *ChatRepository) GetNewMessages(afterID uint) ([]models.ChatMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `SELECT id, user_id, username, nickname, avatar, content, message_type, send_time, status, created_at
			  FROM chat_messages
			  WHERE status = 1 AND id > ?
			  ORDER BY id ASC
			  LIMIT 100`

	rows, err := r.db.DB.QueryContext(ctx, query, afterID)
	if err != nil {
		r.logger.Error("获取新消息失败", "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}
	defer rows.Close()

	// 预分配slice容量（性能优化，最多100条）
	messages := make([]models.ChatMessage, 0, 100)
	for rows.Next() {
		var msg models.ChatMessage
		if err := rows.Scan(&msg.ID, &msg.UserID, &msg.Username, &msg.Nickname, &msg.Avatar,
			&msg.Content, &msg.MessageType, &msg.SendTime, &msg.Status, &msg.CreatedAt); err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// DeleteMessage 删除消息（软删除）
func (r *ChatRepository) DeleteMessage(messageID, userID uint) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `UPDATE chat_messages SET status = 0 WHERE id = ? AND user_id = ?`
	result, err := r.db.DB.ExecContext(ctx, query, messageID, userID)
	if err != nil {
		r.logger.Error("删除消息失败", "error", err.Error())
		return utils.ErrDatabaseQuery
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return utils.ErrDatabaseUpdate
	}

	return nil
}

// Legacy online user management methods removed - now handled by WebSocket ConnectionHub
// The following methods are no longer needed:
// - UpdateOnlineUser: Online status managed in memory by WebSocket
// - GetOnlineCount: Replaced by ConnectionHub.GetOnlineCount()
// - CleanOldOnlineUsers: WebSocket automatically handles disconnections
// - RemoveOnlineUser: WebSocket handles connection cleanup
