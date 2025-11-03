package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gin/internal/models"
)

// PrivateMessageRepository 私信仓库
type PrivateMessageRepository struct {
	db *Database
}

// NewPrivateMessageRepository 创建私信仓库
func NewPrivateMessageRepository(db *Database) *PrivateMessageRepository {
	return &PrivateMessageRepository{db: db}
}

// GetOrCreateConversation 获取或创建会话
func (r *PrivateMessageRepository) GetOrCreateConversation(ctx context.Context, user1ID, user2ID uint) (*models.PrivateConversation, error) {
	// 确保user1ID < user2ID（标准化存储）
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
	}

	// 先尝试获取现有会话
	var conv models.PrivateConversation
	query := `SELECT id, user1_id, user2_id, last_message_id, last_message_content, last_message_time, 
	          user1_unread, user2_unread, created_at, updated_at 
	          FROM private_conversations WHERE user1_id = ? AND user2_id = ?`

	var lastMsgID sql.NullInt64
	var lastMsgContent sql.NullString
	var lastMsgTime sql.NullTime

	err := r.db.DB.QueryRowContext(ctx, query, user1ID, user2ID).Scan(
		&conv.ID,
		&conv.User1ID,
		&conv.User2ID,
		&lastMsgID,
		&lastMsgContent,
		&lastMsgTime,
		&conv.User1Unread,
		&conv.User2Unread,
		&conv.CreatedAt,
		&conv.UpdatedAt,
	)

	if err == nil {
		if lastMsgID.Valid {
			id := uint(lastMsgID.Int64)
			conv.LastMessageID = &id
		}
		if lastMsgContent.Valid {
			conv.LastMessageContent = lastMsgContent.String
		}
		if lastMsgTime.Valid {
			conv.LastMessageTime = &lastMsgTime.Time
		}
		return &conv, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("查询会话失败: %w", err)
	}

	// 不存在则创建新会话
	now := time.Now().UTC()
	insertQuery := `
		INSERT INTO private_conversations (user1_id, user2_id, user1_unread, user2_unread, created_at, updated_at)
		VALUES (?, ?, 0, 0, ?, ?)
	`
	result, err := r.db.DB.ExecContext(ctx, insertQuery, user1ID, user2ID, now, now)
	if err != nil {
		return nil, fmt.Errorf("创建会话失败: %w", err)
	}

	id, _ := result.LastInsertId()
	conv.ID = uint(id)
	conv.User1ID = user1ID
	conv.User2ID = user2ID
	conv.User1Unread = 0
	conv.User2Unread = 0
	conv.CreatedAt = now
	conv.UpdatedAt = now

	return &conv, nil
}

// SendMessage 发送私信
func (r *PrivateMessageRepository) SendMessage(ctx context.Context, senderID, receiverID uint, content string) (*models.PrivateMessage, error) {
	// 获取或创建会话
	conv, err := r.GetOrCreateConversation(ctx, senderID, receiverID)
	if err != nil {
		return nil, err
	}

	// 插入消息
	now := time.Now().UTC()
	insertQuery := `
		INSERT INTO private_messages (conversation_id, sender_id, receiver_id, content, is_read, created_at)
		VALUES (?, ?, ?, ?, 0, ?)
	`
	result, err := r.db.DB.ExecContext(ctx, insertQuery, conv.ID, senderID, receiverID, content, now)
	if err != nil {
		return nil, fmt.Errorf("发送消息失败: %w", err)
	}

	messageID, _ := result.LastInsertId()

	// 更新会话信息
	isUser1 := senderID == conv.User1ID
	updateQuery := `
		UPDATE private_conversations
		SET last_message_id = ?,
		    last_message_content = ?,
		    last_message_time = ?,
		    user1_unread = user1_unread + ?,
		    user2_unread = user2_unread + ?,
		    updated_at = ?
		WHERE id = ?
	`
	var user1UnreadInc, user2UnreadInc int
	if isUser1 {
		user2UnreadInc = 1 // user2未读+1
	} else {
		user1UnreadInc = 1 // user1未读+1
	}

	_, err = r.db.DB.ExecContext(ctx, updateQuery, messageID, content, now, user1UnreadInc, user2UnreadInc, now, conv.ID)
	if err != nil {
		return nil, fmt.Errorf("更新会话失败: %w", err)
	}

	message := &models.PrivateMessage{
		ID:             uint(messageID),
		ConversationID: conv.ID,
		SenderID:       senderID,
		ReceiverID:     receiverID,
		Content:        content,
		IsRead:         false,
		CreatedAt:      now,
	}

	return message, nil
}

// GetUserConversations 获取用户的所有会话
func (r *PrivateMessageRepository) GetUserConversations(ctx context.Context, userID uint) ([]models.PrivateConversation, error) {
	query := `
		SELECT id, user1_id, user2_id, last_message_id, last_message_content, last_message_time,
		       user1_unread, user2_unread, created_at, updated_at
		FROM private_conversations
		WHERE user1_id = ? OR user2_id = ?
		ORDER BY last_message_time DESC
	`
	rows, err := r.db.DB.QueryContext(ctx, query, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("获取会话列表失败: %w", err)
	}
	defer rows.Close()

	// 初始化为空数组，避免返回null
	conversations := make([]models.PrivateConversation, 0)
	for rows.Next() {
		var conv models.PrivateConversation
		var lastMsgID sql.NullInt64
		var lastMsgContent sql.NullString
		var lastMsgTime sql.NullTime

		err := rows.Scan(
			&conv.ID,
			&conv.User1ID,
			&conv.User2ID,
			&lastMsgID,
			&lastMsgContent,
			&lastMsgTime,
			&conv.User1Unread,
			&conv.User2Unread,
			&conv.CreatedAt,
			&conv.UpdatedAt,
		)
		if err != nil {
			continue
		}

		if lastMsgID.Valid {
			id := uint(lastMsgID.Int64)
			conv.LastMessageID = &id
		}
		if lastMsgContent.Valid {
			conv.LastMessageContent = lastMsgContent.String
		}
		if lastMsgTime.Valid {
			conv.LastMessageTime = &lastMsgTime.Time
		}

		conversations = append(conversations, conv)
	}

	return conversations, nil
}

// GetConversationMessages 获取会话消息列表
func (r *PrivateMessageRepository) GetConversationMessages(ctx context.Context, conversationID uint, limit int) ([]models.PrivateMessage, error) {
	query := `
		SELECT id, conversation_id, sender_id, receiver_id, content, is_read, created_at
		FROM private_messages
		WHERE conversation_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`
	rows, err := r.db.DB.QueryContext(ctx, query, conversationID, limit)
	if err != nil {
		return nil, fmt.Errorf("获取消息列表失败: %w", err)
	}
	defer rows.Close()

	// 预分配slice容量（性能优化）
	messages := make([]models.PrivateMessage, 0, limit)
	for rows.Next() {
		var msg models.PrivateMessage
		var isRead int

		err := rows.Scan(
			&msg.ID,
			&msg.ConversationID,
			&msg.SenderID,
			&msg.ReceiverID,
			&msg.Content,
			&isRead,
			&msg.CreatedAt,
		)
		if err != nil {
			continue
		}

		msg.IsRead = isRead == 1
		messages = append(messages, msg)
	}

	// 反转顺序（最旧的在前）
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// MarkAsRead 标记消息为已读
func (r *PrivateMessageRepository) MarkAsRead(ctx context.Context, conversationID, userID uint) error {
	// 标记该会话中接收给当前用户的所有未读消息为已读
	updateQuery := `
		UPDATE private_messages
		SET is_read = 1
		WHERE conversation_id = ? AND receiver_id = ? AND is_read = 0
	`
	_, err := r.db.DB.ExecContext(ctx, updateQuery, conversationID, userID)
	if err != nil {
		return fmt.Errorf("标记已读失败: %w", err)
	}

	// 获取会话信息
	conv, err := r.GetConversationByID(ctx, conversationID)
	if err != nil {
		return err
	}

	// 清零对应用户的未读数
	if userID == conv.User1ID {
		_, err = r.db.DB.ExecContext(ctx, `UPDATE private_conversations SET user1_unread = 0 WHERE id = ?`, conversationID)
	} else {
		_, err = r.db.DB.ExecContext(ctx, `UPDATE private_conversations SET user2_unread = 0 WHERE id = ?`, conversationID)
	}

	return err
}

// GetUnreadCount 获取用户未读消息总数
func (r *PrivateMessageRepository) GetUnreadCount(ctx context.Context, userID uint) (int, error) {
	query := `
		SELECT COALESCE(SUM(
			CASE
				WHEN user1_id = ? THEN user1_unread
				WHEN user2_id = ? THEN user2_unread
				ELSE 0
			END
		), 0) as total_unread
		FROM private_conversations
		WHERE user1_id = ? OR user2_id = ?
	`
	var count int
	err := r.db.DB.QueryRowContext(ctx, query, userID, userID, userID, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("获取未读数失败: %w", err)
	}
	return count, nil
}

// GetConversationByID 根据ID获取会话
func (r *PrivateMessageRepository) GetConversationByID(ctx context.Context, conversationID uint) (*models.PrivateConversation, error) {
	var conv models.PrivateConversation
	var lastMsgID sql.NullInt64
	var lastMsgContent sql.NullString
	var lastMsgTime sql.NullTime

	query := `SELECT id, user1_id, user2_id, last_message_id, last_message_content, last_message_time,
	          user1_unread, user2_unread, created_at, updated_at FROM private_conversations WHERE id = ?`

	err := r.db.DB.QueryRowContext(ctx, query, conversationID).Scan(
		&conv.ID,
		&conv.User1ID,
		&conv.User2ID,
		&lastMsgID,
		&lastMsgContent,
		&lastMsgTime,
		&conv.User1Unread,
		&conv.User2Unread,
		&conv.CreatedAt,
		&conv.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("获取会话失败: %w", err)
	}

	if lastMsgID.Valid {
		id := uint(lastMsgID.Int64)
		conv.LastMessageID = &id
	}
	if lastMsgContent.Valid {
		conv.LastMessageContent = lastMsgContent.String
	}
	if lastMsgTime.Valid {
		conv.LastMessageTime = &lastMsgTime.Time
	}

	return &conv, nil
}

// GetConversationByUsers 根据两个用户ID获取会话
func (r *PrivateMessageRepository) GetConversationByUsers(ctx context.Context, user1ID, user2ID uint) (*models.PrivateConversation, error) {
	// 确保user1ID < user2ID
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
	}

	var conv models.PrivateConversation
	var lastMsgID sql.NullInt64
	var lastMsgContent sql.NullString
	var lastMsgTime sql.NullTime

	query := `SELECT id, user1_id, user2_id, last_message_id, last_message_content, last_message_time,
	          user1_unread, user2_unread, created_at, updated_at FROM private_conversations WHERE user1_id = ? AND user2_id = ?`

	err := r.db.DB.QueryRowContext(ctx, query, user1ID, user2ID).Scan(
		&conv.ID,
		&conv.User1ID,
		&conv.User2ID,
		&lastMsgID,
		&lastMsgContent,
		&lastMsgTime,
		&conv.User1Unread,
		&conv.User2Unread,
		&conv.CreatedAt,
		&conv.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if lastMsgID.Valid {
		id := uint(lastMsgID.Int64)
		conv.LastMessageID = &id
	}
	if lastMsgContent.Valid {
		conv.LastMessageContent = lastMsgContent.String
	}
	if lastMsgTime.Valid {
		conv.LastMessageTime = &lastMsgTime.Time
	}

	return &conv, nil
}
