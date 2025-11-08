package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gin/internal/models"
)

// BatchRepository 批量查询仓库（解决N+1查询问题）
type BatchRepository struct {
	db *Database
}

// NewBatchRepository 创建批量查询仓库
func NewBatchRepository(db *Database) *BatchRepository {
	return &BatchRepository{db: db}
}

// BatchGetUsers 批量获取用户信息
func (r *BatchRepository) BatchGetUsers(ctx context.Context, userIDs []uint) (map[uint]*models.User, error) {
	if len(userIDs) == 0 {
		return make(map[uint]*models.User), nil
	}

	// 去重（预分配容量）
	uniqueIDs := make(map[uint]bool, len(userIDs))
	for _, id := range userIDs {
		uniqueIDs[id] = true
	}

	// 构建IN查询
	ids := make([]uint, 0, len(uniqueIDs))
	for id := range uniqueIDs {
		ids = append(ids, id)
	}

	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]

	query := fmt.Sprintf(`
		SELECT id, username, email, role, created_at
		FROM users
		WHERE id IN (%s)
	`, placeholders)

	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	rows, err := r.db.QueryWithCache(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make(map[uint]*models.User, len(ids))
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Role,
			&user.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		users[user.ID] = &user
	}

	return users, rows.Err()
}

// BatchGetArticles 批量获取文章信息
func (r *BatchRepository) BatchGetArticles(ctx context.Context, articleIDs []uint) (map[uint]*models.Article, error) {
	if len(articleIDs) == 0 {
		return make(map[uint]*models.Article), nil
	}

	// 去重（预分配容量）
	uniqueIDs := make(map[uint]bool, len(articleIDs))
	for _, id := range articleIDs {
		uniqueIDs[id] = true
	}

	ids := make([]uint, 0, len(uniqueIDs))
	for id := range uniqueIDs {
		ids = append(ids, id)
	}

	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]

	query := fmt.Sprintf(`
		SELECT id, user_id, title, description, content, status, 
		       view_count, like_count, comment_count, created_at, updated_at
		FROM articles
		WHERE id IN (%s) AND status = 1
	`, placeholders)

	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	rows, err := r.db.QueryWithCache(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	articles := make(map[uint]*models.Article, len(ids))
	for rows.Next() {
		var article models.Article
		err := rows.Scan(
			&article.ID,
			&article.UserID,
			&article.Title,
			&article.Description,
			&article.Content,
			&article.Status,
			&article.ViewCount,
			&article.LikeCount,
			&article.CommentCount,
			&article.CreatedAt,
			&article.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		articles[article.ID] = &article
	}

	return articles, rows.Err()
}

// BatchGetUserProfiles 批量获取用户资料（包含profile字段）
func (r *BatchRepository) BatchGetUserProfiles(ctx context.Context, userIDs []uint) (map[uint]*BatchUserProfile, error) {
	if len(userIDs) == 0 {
		return make(map[uint]*BatchUserProfile), nil
	}

	// 去重（预分配容量）
	uniqueIDs := make(map[uint]bool, len(userIDs))
	for _, id := range userIDs {
		uniqueIDs[id] = true
	}

	ids := make([]uint, 0, len(uniqueIDs))
	for id := range uniqueIDs {
		ids = append(ids, id)
	}

	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]

	query := fmt.Sprintf(`
		SELECT u.id, u.username, u.email, u.avatar, u.role, u.created_at,
		       up.nickname, up.bio, up.location, up.website
		FROM users u
		LEFT JOIN user_profiles up ON u.id = up.user_id
		WHERE u.id IN (%s)
	`, placeholders)

	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	rows, err := r.db.QueryWithCache(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make(map[uint]*BatchUserProfile, len(ids))
	for rows.Next() {
		var user BatchUserProfile
		var nickname, bio, location, website *string

		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Avatar,
			&user.Role,
			&user.CreatedAt,
			&nickname,
			&bio,
			&location,
			&website,
		)
		if err != nil {
			return nil, err
		}

		// 设置profile字段
		if nickname != nil {
			user.Nickname = *nickname
		}
		if bio != nil {
			user.Bio = *bio
		}
		if location != nil {
			user.Location = *location
		}
		if website != nil {
			user.Website = *website
		}

		users[user.ID] = &user
	}

	return users, rows.Err()
}

// BatchUserProfile 批量查询用户资料的结构体（优化版）
type BatchUserProfile struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Avatar    string    `json:"avatar"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	Nickname  string    `json:"nickname,omitempty"`
	Bio       string    `json:"bio,omitempty"`
	Location  string    `json:"location,omitempty"`
	Website   string    `json:"website,omitempty"`
}

// BatchCheckArticleLikes 批量检查文章点赞状态
func (r *BatchRepository) BatchCheckArticleLikes(ctx context.Context, articleIDs []uint, userID uint) (map[uint]bool, error) {
	if len(articleIDs) == 0 {
		return make(map[uint]bool), nil
	}

	// 去重（预分配容量）
	uniqueIDs := make(map[uint]bool, len(articleIDs))
	for _, id := range articleIDs {
		uniqueIDs[id] = true
	}

	ids := make([]uint, 0, len(uniqueIDs))
	for id := range uniqueIDs {
		ids = append(ids, id)
	}

	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]

	query := fmt.Sprintf(`
		SELECT article_id
		FROM article_likes
		WHERE article_id IN (%s) AND user_id = ?
	`, placeholders)

	args := make([]interface{}, len(ids)+1)
	for i, id := range ids {
		args[i] = id
	}
	args[len(ids)] = userID

	rows, err := r.db.QueryWithCache(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	likes := make(map[uint]bool, len(ids))
	for rows.Next() {
		var articleID uint
		if err := rows.Scan(&articleID); err != nil {
			return nil, err
		}
		likes[articleID] = true
	}

	// 填充未点赞的文章
	for _, id := range ids {
		if _, exists := likes[id]; !exists {
			likes[id] = false
		}
	}

	return likes, rows.Err()
}

// BatchGetCommentCounts 批量获取评论数
func (r *BatchRepository) BatchGetCommentCounts(ctx context.Context, articleIDs []uint) (map[uint]int, error) {
	if len(articleIDs) == 0 {
		return make(map[uint]int), nil
	}

	// 去重（预分配容量）
	uniqueIDs := make(map[uint]bool, len(articleIDs))
	for _, id := range articleIDs {
		uniqueIDs[id] = true
	}

	ids := make([]uint, 0, len(uniqueIDs))
	for id := range uniqueIDs {
		ids = append(ids, id)
	}

	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]

	query := fmt.Sprintf(`
		SELECT article_id, COUNT(*) as count
		FROM article_comments
		WHERE article_id IN (%s) AND status = 1
		GROUP BY article_id
	`, placeholders)

	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	rows, err := r.db.QueryWithCache(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[uint]int, len(ids))
	for rows.Next() {
		var articleID uint
		var count int
		if err := rows.Scan(&articleID, &count); err != nil {
			return nil, err
		}
		counts[articleID] = count
	}

	// 填充0计数的文章
	for _, id := range ids {
		if _, exists := counts[id]; !exists {
			counts[id] = 0
		}
	}

	return counts, rows.Err()
}
