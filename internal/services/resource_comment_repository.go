package services

import (
	"context"
	"database/sql"
	"time"

	"gin/internal/models"
	"gin/internal/utils"
)

// ResourceCommentRepository 资源评论仓库
type ResourceCommentRepository struct {
	db     *Database
	logger utils.Logger
}

// NewResourceCommentRepository 创建资源评论仓库
func NewResourceCommentRepository(db *Database) *ResourceCommentRepository {
	return &ResourceCommentRepository{
		db:     db,
		logger: utils.GetLogger(),
	}
}

// CreateComment 创建评论
func (r *ResourceCommentRepository) CreateComment(ctx context.Context, comment *models.ResourceComment) error {
	tx, err := r.db.DB.BeginTx(ctx, nil)
	if err != nil {
		return utils.ErrDatabaseQuery
	}
	defer tx.Rollback()

	// 插入评论
	query := `INSERT INTO resource_comments (resource_id, user_id, parent_id, root_id, reply_to_user_id, 
	          content, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := tx.ExecContext(ctx, query, comment.ResourceID, comment.UserID, comment.ParentID,
		comment.RootID, comment.ReplyToUserID, comment.Content, comment.CreatedAt, comment.UpdatedAt)

	if err != nil {
		r.logger.Error("插入评论失败", "error", err.Error())
		return utils.ErrDatabaseInsert
	}

	commentID, _ := result.LastInsertId()
	comment.ID = uint(commentID)

	// 更新资源评论数
	_, _ = tx.ExecContext(ctx, `UPDATE resources SET comment_count = comment_count + 1 WHERE id = ?`, comment.ResourceID)

	// 如果是回复，更新父评论的回复数
	if comment.ParentID > 0 {
		_, _ = tx.ExecContext(ctx, `UPDATE resource_comments SET reply_count = reply_count + 1 WHERE id = ?`, comment.ParentID)
	}

	if err := tx.Commit(); err != nil {
		return utils.ErrDatabaseInsert
	}

	r.logger.Info("创建评论成功", "commentID", comment.ID, "resourceID", comment.ResourceID)
	return nil
}

// GetCommentsByResourceID 获取资源评论列表
func (r *ResourceCommentRepository) GetCommentsByResourceID(ctx context.Context, resourceID, userID uint, page, pageSize int) (*models.ResourceCommentsResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 查询总数
	var total int
	countQuery := `SELECT COUNT(*) FROM resource_comments WHERE resource_id = ? AND status = 1`
	err := r.db.DB.QueryRowContext(ctx, countQuery, resourceID).Scan(&total)
	if err != nil {
		return nil, utils.ErrDatabaseQuery
	}

	// 查询一级评论
	offset := (page - 1) * pageSize
	query := `SELECT id, resource_id, user_id, parent_id, root_id, reply_to_user_id, content,
	          like_count, reply_count, created_at, updated_at
	          FROM resource_comments 
	          WHERE resource_id = ? AND parent_id = 0 AND status = 1
	          ORDER BY created_at DESC
	          LIMIT ? OFFSET ?`

	rows, err := r.db.DB.QueryContext(ctx, query, resourceID, pageSize, offset)
	if err != nil {
		return nil, utils.ErrDatabaseQuery
	}
	defer rows.Close()

	var comments []models.ResourceCommentResponse
	for rows.Next() {
		comment, err := r.scanComment(rows, userID)
		if err != nil {
			continue
		}

		// 加载用户信息
		r.loadCommentUser(ctx, &comment)

		// 加载回复
		r.loadReplies(ctx, &comment, userID)

		comments = append(comments, comment)
	}

	totalPages := (total + pageSize - 1) / pageSize

	return &models.ResourceCommentsResponse{
		Comments:   comments,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// scanComment 扫描评论数据
func (r *ResourceCommentRepository) scanComment(rows *sql.Rows, userID uint) (models.ResourceCommentResponse, error) {
	var comment models.ResourceCommentResponse
	var replyToUserID sql.NullInt64

	err := rows.Scan(
		&comment.ID, &comment.ResourceID, &comment.UserID, &comment.ParentID,
		&comment.RootID, &replyToUserID, &comment.Content, &comment.LikeCount,
		&comment.ReplyCount, &comment.CreatedAt, &sql.NullTime{},
	)

	if err != nil {
		return comment, err
	}

	if replyToUserID.Valid {
		id := uint(replyToUserID.Int64)
		comment.ReplyToUser = &models.CommentUser{ID: id}
	}

	// 检查当前用户是否点赞
	if userID > 0 {
		comment.IsLiked = r.checkUserLiked(context.Background(), comment.ID, userID)
	}

	return comment, nil
}

// loadCommentUser 加载评论用户信息
func (r *ResourceCommentRepository) loadCommentUser(ctx context.Context, comment *models.ResourceCommentResponse) {
	query := `SELECT ua.id, ua.username, COALESCE(up.nickname, ua.username) as nickname, 
	          COALESCE(up.avatar_url, '') as avatar
	          FROM user_auth ua LEFT JOIN user_profile up ON ua.id = up.user_id 
	          WHERE ua.id = ?`

	user := &models.CommentUser{}
	err := r.db.DB.QueryRowContext(ctx, query, comment.UserID).Scan(
		&user.ID, &user.Username, &user.Nickname, &user.Avatar,
	)

	if err == nil {
		comment.User = user
	}

	// 加载回复对象的用户信息
	if comment.ReplyToUser != nil && comment.ReplyToUser.ID > 0 {
		replyUser := &models.CommentUser{}
		err := r.db.DB.QueryRowContext(ctx, query, comment.ReplyToUser.ID).Scan(
			&replyUser.ID, &replyUser.Username, &replyUser.Nickname, &replyUser.Avatar,
		)
		if err == nil {
			comment.ReplyToUser = replyUser
		}
	}
}

// loadReplies 加载子评论
func (r *ResourceCommentRepository) loadReplies(ctx context.Context, comment *models.ResourceCommentResponse, userID uint) {
	query := `SELECT id, resource_id, user_id, parent_id, root_id, reply_to_user_id, content,
	          like_count, reply_count, created_at, updated_at
	          FROM resource_comments 
	          WHERE parent_id = ? AND status = 1
	          ORDER BY created_at ASC`

	rows, err := r.db.DB.QueryContext(ctx, query, comment.ID)
	if err != nil {
		return
	}
	defer rows.Close()

	var replies []models.ResourceCommentResponse
	for rows.Next() {
		reply, err := r.scanComment(rows, userID)
		if err != nil {
			continue
		}

		r.loadCommentUser(ctx, &reply)
		replies = append(replies, reply)
	}

	comment.Replies = replies
}

// checkUserLiked 检查用户是否点赞
func (r *ResourceCommentRepository) checkUserLiked(ctx context.Context, commentID, userID uint) bool {
	var id uint
	query := `SELECT id FROM resource_comment_likes WHERE comment_id = ? AND user_id = ?`
	err := r.db.DB.QueryRowContext(ctx, query, commentID, userID).Scan(&id)
	return err == nil
}

// ToggleCommentLike 切换评论点赞
func (r *ResourceCommentRepository) ToggleCommentLike(ctx context.Context, commentID, userID uint) (bool, error) {
	// 检查是否已点赞
	checkQuery := `SELECT id FROM resource_comment_likes WHERE comment_id = ? AND user_id = ?`
	var likeID uint
	err := r.db.DB.QueryRowContext(ctx, checkQuery, commentID, userID).Scan(&likeID)

	isLiked := false
	switch err {
	case sql.ErrNoRows:
		// 未点赞，执行点赞
		_, err := r.db.DB.ExecContext(ctx, `INSERT INTO resource_comment_likes (comment_id, user_id, created_at) VALUES (?, ?, ?)`,
			commentID, userID, time.Now())
		if err != nil {
			return false, utils.ErrDatabaseInsert
		}
		_, _ = r.db.DB.ExecContext(ctx, `UPDATE resource_comments SET like_count = like_count + 1 WHERE id = ?`, commentID)
		isLiked = true
	case nil:
		// 已点赞，取消点赞
		_, err := r.db.DB.ExecContext(ctx, `DELETE FROM resource_comment_likes WHERE comment_id = ? AND user_id = ?`, commentID, userID)
		if err != nil {
			return false, utils.ErrDatabaseUpdate
		}
		_, _ = r.db.DB.ExecContext(ctx, `UPDATE resource_comments SET like_count = GREATEST(like_count - 1, 0) WHERE id = ?`, commentID)
		isLiked = false
	default:
		return false, utils.ErrDatabaseQuery
	}

	return isLiked, nil
}

// GetParentRootID 获取父评论的root_id
func (r *ResourceCommentRepository) GetParentRootID(ctx context.Context, parentID uint) uint {
	var rootID uint
	query := `SELECT CASE WHEN root_id = 0 THEN id ELSE root_id END as root_id FROM resource_comments WHERE id = ?`
	err := r.db.DB.QueryRowContext(ctx, query, parentID).Scan(&rootID)
	if err != nil {
		return parentID // 如果查询失败，返回父ID作为root_id
	}
	return rootID
}

// DeleteComment 删除评论
func (r *ResourceCommentRepository) DeleteComment(ctx context.Context, commentID, userID uint) error {
	// 检查所有权
	var ownerID uint
	var resourceID uint
	err := r.db.DB.QueryRowContext(ctx, `SELECT user_id, resource_id FROM resource_comments WHERE id = ? AND status != 0`, commentID).Scan(&ownerID, &resourceID)
	if err != nil {
		if err == sql.ErrNoRows {
			return utils.ErrUserNotFound
		}
		return utils.ErrDatabaseQuery
	}

	if ownerID != userID {
		return utils.ErrUnauthorized
	}

	// 软删除
	_, err = r.db.DB.ExecContext(ctx, `UPDATE resource_comments SET status = 0, updated_at = ? WHERE id = ?`, time.Now(), commentID)
	if err != nil {
		return utils.ErrDatabaseUpdate
	}

	// 更新资源评论数
	_, _ = r.db.DB.ExecContext(ctx, `UPDATE resources SET comment_count = GREATEST(comment_count - 1, 0) WHERE id = ?`, resourceID)

	return nil
}
