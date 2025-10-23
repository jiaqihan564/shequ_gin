package services

import (
	"context"
	"database/sql"
	"strings"
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

	r.logger.Info("准备插入评论", "resourceID", comment.ResourceID, "userID", comment.UserID,
		"parentID", comment.ParentID, "rootID", comment.RootID, "replyToUserID", comment.ReplyToUserID)

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

	// 初始化为空数组，避免返回null
	comments := make([]models.ResourceCommentResponse, 0)
	commentIDs := make([]uint, 0)
	userIDs := make([]uint, 0)

	// 第一步：收集所有评论和用户ID
	for rows.Next() {
		comment, err := r.scanComment(rows, userID)
		if err != nil {
			continue
		}
		commentIDs = append(commentIDs, comment.ID)
		userIDs = append(userIDs, comment.UserID)
		comments = append(comments, comment)
	}

	// 如果没有评论，直接返回
	if len(comments) == 0 {
		return &models.ResourceCommentsResponse{
			Comments:   comments,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: (total + pageSize - 1) / pageSize,
		}, nil
	}

	// 第二步：批量查询所有用户信息（优化N+1）
	userMap := r.batchGetUserInfo(ctx, userIDs)

	// 第三步：批量查询所有评论的回复（优化N+1）
	repliesMap := r.batchGetReplies(ctx, commentIDs, userID)
	r.logger.Info("批量查询评论回复", "commentCount", len(commentIDs), "repliesMapSize", len(repliesMap))

	// 第四步：组装数据
	for i := range comments {
		// 设置用户信息
		if user, exists := userMap[comments[i].UserID]; exists {
			comments[i].User = user
		}

		// 设置回复，确保始终有 Replies 字段（即使为空数组）
		if replies, exists := repliesMap[comments[i].ID]; exists && len(replies) > 0 {
			comments[i].Replies = replies
		} else {
			comments[i].Replies = make([]models.ResourceCommentResponse, 0)
		}
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
	// 初始化Replies为空数组
	comment.Replies = make([]models.ResourceCommentResponse, 0)
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

// batchGetUserInfo 批量获取用户信息（优化N+1）
func (r *ResourceCommentRepository) batchGetUserInfo(ctx context.Context, userIDs []uint) map[uint]*models.CommentUser {
	userMap := make(map[uint]*models.CommentUser)
	if len(userIDs) == 0 {
		return userMap
	}

	// 去重
	uniqueIDs := make(map[uint]bool)
	for _, id := range userIDs {
		uniqueIDs[id] = true
	}

	// 构建批量查询
	ids := make([]uint, 0, len(uniqueIDs))
	for id := range uniqueIDs {
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return userMap
	}

	query := `SELECT ua.id, ua.username, COALESCE(up.nickname, ua.username) as nickname, 
	          COALESCE(up.avatar_url, '') as avatar
	          FROM user_auth ua LEFT JOIN user_profile up ON ua.id = up.user_id 
	          WHERE ua.id IN (?` + strings.Repeat(",?", len(ids)-1) + `)`

	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	rows, err := r.db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return userMap
	}
	defer rows.Close()

	for rows.Next() {
		user := &models.CommentUser{}
		if err := rows.Scan(&user.ID, &user.Username, &user.Nickname, &user.Avatar); err == nil {
			userMap[user.ID] = user
		}
	}

	return userMap
}

// batchGetReplies 批量获取评论回复（优化N+1）
func (r *ResourceCommentRepository) batchGetReplies(ctx context.Context, commentIDs []uint, userID uint) map[uint][]models.ResourceCommentResponse {
	repliesMap := make(map[uint][]models.ResourceCommentResponse)

	if len(commentIDs) == 0 {
		return repliesMap
	}

	// 批量查询所有回复
	query := `SELECT id, resource_id, user_id, parent_id, root_id, reply_to_user_id, content,
	          like_count, reply_count, created_at, updated_at
	          FROM resource_comments 
	          WHERE parent_id IN (?` + strings.Repeat(",?", len(commentIDs)-1) + `) AND status = 1
	          ORDER BY created_at ASC`

	args := make([]interface{}, len(commentIDs))
	for i, id := range commentIDs {
		args[i] = id
	}

	rows, err := r.db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("批量查询回复失败", "error", err.Error())
		// 返回空map，每个评论都有空数组
		for _, id := range commentIDs {
			repliesMap[id] = make([]models.ResourceCommentResponse, 0)
		}
		return repliesMap
	}
	defer rows.Close()

	// 初始化所有评论的回复数组
	for _, id := range commentIDs {
		repliesMap[id] = make([]models.ResourceCommentResponse, 0)
	}

	// 收集所有回复和用户ID
	allReplies := make([]models.ResourceCommentResponse, 0)
	replyUserIDs := make([]uint, 0)
	childCommentIDs := make([]uint, 0)

	for rows.Next() {
		reply, err := r.scanComment(rows, userID)
		if err != nil {
			r.logger.Warn("扫描回复失败", "error", err.Error())
			continue
		}
		allReplies = append(allReplies, reply)
		replyUserIDs = append(replyUserIDs, reply.UserID)
		childCommentIDs = append(childCommentIDs, reply.ID)
		if reply.ReplyToUser != nil && reply.ReplyToUser.ID > 0 {
			replyUserIDs = append(replyUserIDs, reply.ReplyToUser.ID)
		}
	}

	r.logger.Info("收集回复数据", "allRepliesCount", len(allReplies), "childCommentIDs", len(childCommentIDs))

	// 批量查询回复的用户信息
	if len(replyUserIDs) > 0 {
		replyUserMap := r.batchGetUserInfo(ctx, replyUserIDs)

		// 组装回复数据
		for i := range allReplies {
			// 设置回复者信息
			if user, exists := replyUserMap[allReplies[i].UserID]; exists {
				allReplies[i].User = user
			}

			// 设置被回复者信息
			if allReplies[i].ReplyToUser != nil && allReplies[i].ReplyToUser.ID > 0 {
				if replyToUser, exists := replyUserMap[allReplies[i].ReplyToUser.ID]; exists {
					allReplies[i].ReplyToUser = replyToUser
				}
			}
		}
	}

	// 递归获取子评论的子评论（支持多层嵌套）
	if len(childCommentIDs) > 0 {
		childRepliesMap := r.batchGetReplies(ctx, childCommentIDs, userID)

		// 为每个回复设置其子回复，确保总是有 Replies 字段
		for i := range allReplies {
			if childReplies, exists := childRepliesMap[allReplies[i].ID]; exists && len(childReplies) > 0 {
				allReplies[i].Replies = childReplies
			} else {
				allReplies[i].Replies = make([]models.ResourceCommentResponse, 0)
			}
		}
	}

	// 添加到对应的父评论
	for i := range allReplies {
		repliesMap[allReplies[i].ParentID] = append(repliesMap[allReplies[i].ParentID], allReplies[i])
	}

	return repliesMap
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
