package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"gin/internal/models"
	"gin/internal/utils"
)

// ArticleRepository 文章仓库
type ArticleRepository struct {
	db     *Database
	logger utils.Logger
}

// NewArticleRepository 创建文章仓库
func NewArticleRepository(db *Database) *ArticleRepository {
	return &ArticleRepository{
		db:     db,
		logger: utils.GetLogger(),
	}
}

// CreateArticle 创建文章
func (r *ArticleRepository) CreateArticle(ctx context.Context, article *models.Article, codeBlocks []models.CreateArticleCodeBlock, categoryIDs, tagIDs []uint) error {
	start := time.Now()
	r.logger.Debug("开始创建文章",
		"userID", article.UserID,
		"title", article.Title,
		"codeBlockCount", len(codeBlocks),
		"categoryCount", len(categoryIDs),
		"tagCount", len(tagIDs))

	// 开启事务
	tx, err := r.db.DB.BeginTx(ctx, nil)
	if err != nil {
		r.logger.Error("开启事务失败", "error", err.Error())
		return utils.ErrDatabaseQuery
	}
	defer tx.Rollback()

	// 1. 插入文章
	query := `INSERT INTO articles (user_id, title, description, content, status, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	result, err := tx.ExecContext(ctx, query,
		article.UserID, article.Title, article.Description, article.Content,
		article.Status, article.CreatedAt, article.UpdatedAt)
	if err != nil {
		r.logger.Error("插入文章失败", "error", err.Error())
		return utils.ErrDatabaseInsert
	}

	articleID, err := result.LastInsertId()
	if err != nil {
		r.logger.Error("获取文章ID失败", "error", err.Error())
		return utils.ErrDatabaseInsert
	}
	article.ID = uint(articleID)

	// 2. 插入代码块
	if len(codeBlocks) > 0 {
		for _, block := range codeBlocks {
			blockQuery := `INSERT INTO article_code_blocks (article_id, language, code_content, description, order_index, created_at)
						   VALUES (?, ?, ?, ?, ?, ?)`
			_, err := tx.ExecContext(ctx, blockQuery,
				article.ID, block.Language, block.CodeContent, block.Description, block.OrderIndex, time.Now())
			if err != nil {
				r.logger.Error("插入代码块失败", "error", err.Error())
				return utils.ErrDatabaseInsert
			}
		}
	}

	// 3. 关联分类
	if len(categoryIDs) > 0 {
		for _, catID := range categoryIDs {
			relQuery := `INSERT INTO article_category_relations (article_id, category_id, created_at) VALUES (?, ?, ?)`
			_, err := tx.ExecContext(ctx, relQuery, article.ID, catID, time.Now())
			if err != nil {
				r.logger.Error("关联分类失败", "categoryID", catID, "error", err.Error())
				return utils.ErrDatabaseInsert
			}
			// 更新分类文章计数
			_, _ = tx.ExecContext(ctx, `UPDATE article_categories SET article_count = article_count + 1 WHERE id = ?`, catID)
		}
	}

	// 4. 关联标签
	if len(tagIDs) > 0 {
		for _, tagID := range tagIDs {
			relQuery := `INSERT INTO article_tag_relations (article_id, tag_id, created_at) VALUES (?, ?, ?)`
			_, err := tx.ExecContext(ctx, relQuery, article.ID, tagID, time.Now())
			if err != nil {
				r.logger.Error("关联标签失败", "tagID", tagID, "error", err.Error())
				return utils.ErrDatabaseInsert
			}
			// 更新标签文章计数
			_, _ = tx.ExecContext(ctx, `UPDATE article_tags SET article_count = article_count + 1 WHERE id = ?`, tagID)
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		r.logger.Error("提交事务失败", "error", err.Error())
		return utils.ErrDatabaseQuery
	}

	r.logger.Info("创建文章成功",
		"articleID", article.ID,
		"userID", article.UserID,
		"title", article.Title,
		"duration", time.Since(start))
	return nil
}

// GetArticleByID 根据ID获取文章详情
func (r *ArticleRepository) GetArticleByID(ctx context.Context, articleID uint, userID uint) (*models.ArticleDetailResponse, error) {
	start := time.Now()
	r.logger.Debug("开始获取文章详情", "articleID", articleID, "userID", userID)

	// 1. 获取文章基本信息
	query := `SELECT id, user_id, title, description, content, status, view_count, like_count, comment_count, created_at, updated_at
			  FROM articles WHERE id = ? AND status != 2`

	var article models.Article
	err := r.db.DB.QueryRowContext(ctx, query, articleID).Scan(
		&article.ID, &article.UserID, &article.Title, &article.Description, &article.Content,
		&article.Status, &article.ViewCount, &article.LikeCount, &article.CommentCount,
		&article.CreatedAt, &article.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Debug("文章不存在", "articleID", articleID)
			return nil, utils.ErrUserNotFound
		}
		r.logger.Error("查询文章失败", "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}

	response := &models.ArticleDetailResponse{
		Article: article,
	}

	// 2. 获取作者信息
	authorQuery := `SELECT ua.id, ua.username, COALESCE(up.nickname, ua.username) as nickname, COALESCE(up.avatar_url, '') as avatar
					FROM user_auth ua
					LEFT JOIN user_profile up ON ua.id = up.user_id
					WHERE ua.id = ?`
	err = r.db.DB.QueryRowContext(ctx, authorQuery, article.UserID).Scan(
		&response.Author.ID, &response.Author.Username, &response.Author.Nickname, &response.Author.Avatar)
	if err != nil && err != sql.ErrNoRows {
		r.logger.Warn("获取作者信息失败", "userID", article.UserID, "error", err.Error())
	}

	// 3. 获取代码块
	codeQuery := `SELECT id, article_id, language, code_content, description, order_index, created_at
				  FROM article_code_blocks WHERE article_id = ? ORDER BY order_index ASC`
	rows, err := r.db.DB.QueryContext(ctx, codeQuery, articleID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var block models.ArticleCodeBlock
			if err := rows.Scan(&block.ID, &block.ArticleID, &block.Language, &block.CodeContent,
				&block.Description, &block.OrderIndex, &block.CreatedAt); err == nil {
				response.CodeBlocks = append(response.CodeBlocks, block)
			}
		}
	}

	// 4. 获取分类
	catQuery := `SELECT ac.id, ac.name, ac.slug, ac.description, ac.parent_id, ac.article_count, ac.sort_order, ac.created_at
				 FROM article_categories ac
				 INNER JOIN article_category_relations acr ON ac.id = acr.category_id
				 WHERE acr.article_id = ?`
	rows, err = r.db.DB.QueryContext(ctx, catQuery, articleID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cat models.ArticleCategory
			if err := rows.Scan(&cat.ID, &cat.Name, &cat.Slug, &cat.Description,
				&cat.ParentID, &cat.ArticleCount, &cat.SortOrder, &cat.CreatedAt); err == nil {
				response.Categories = append(response.Categories, cat)
			}
		}
	}

	// 5. 获取标签
	tagQuery := `SELECT at.id, at.name, at.slug, at.article_count, at.created_at
				 FROM article_tags at
				 INNER JOIN article_tag_relations atr ON at.id = atr.tag_id
				 WHERE atr.article_id = ?`
	rows, err = r.db.DB.QueryContext(ctx, tagQuery, articleID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tag models.ArticleTag
			if err := rows.Scan(&tag.ID, &tag.Name, &tag.Slug, &tag.ArticleCount, &tag.CreatedAt); err == nil {
				response.Tags = append(response.Tags, tag)
			}
		}
	}

	// 6. 检查当前用户是否点赞
	if userID > 0 {
		likeQuery := `SELECT COUNT(*) FROM article_likes WHERE article_id = ? AND user_id = ?`
		var count int
		if err := r.db.DB.QueryRowContext(ctx, likeQuery, articleID, userID).Scan(&count); err == nil {
			response.IsLiked = count > 0
		}
	}

	r.logger.Info("获取文章详情成功", "articleID", articleID, "duration", time.Since(start))
	return response, nil
}

// ListArticles 获取文章列表
func (r *ArticleRepository) ListArticles(ctx context.Context, query models.ArticleListQuery) (*models.ArticleListResponse, error) {
	start := time.Now()

	// 设置默认值
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 || query.PageSize > 100 {
		query.PageSize = 20
	}
	offset := (query.Page - 1) * query.PageSize

	// 构建查询条件
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "a.status = 1") // 只查询已发布的

	if query.CategoryID > 0 {
		conditions = append(conditions, "EXISTS (SELECT 1 FROM article_category_relations acr WHERE acr.article_id = a.id AND acr.category_id = ?)")
		args = append(args, query.CategoryID)
	}

	if query.TagID > 0 {
		conditions = append(conditions, "EXISTS (SELECT 1 FROM article_tag_relations atr WHERE atr.article_id = a.id AND atr.tag_id = ?)")
		args = append(args, query.TagID)
	}

	if query.UserID > 0 {
		conditions = append(conditions, "a.user_id = ?")
		args = append(args, query.UserID)
	}

	if query.Keyword != "" {
		conditions = append(conditions, "(a.title LIKE ? OR a.description LIKE ? OR a.content LIKE ?)")
		keyword := "%" + query.Keyword + "%"
		args = append(args, keyword, keyword, keyword)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// 排序
	orderBy := "a.created_at DESC"
	switch query.SortBy {
	case "hot":
		orderBy = "a.like_count DESC, a.view_count DESC, a.created_at DESC"
	case "popular":
		orderBy = "a.view_count DESC, a.like_count DESC, a.created_at DESC"
	}

	// 查询总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM articles a %s", whereClause)
	var total int
	err := r.db.DB.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		r.logger.Error("查询文章总数失败", "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}

	// 查询列表
	listQuery := fmt.Sprintf(`
		SELECT a.id, a.user_id, a.title, a.description, a.view_count, a.like_count, a.comment_count, a.created_at, a.updated_at,
			   ua.username, COALESCE(up.nickname, ua.username) as nickname, COALESCE(up.avatar_url, '') as avatar
		FROM articles a
		INNER JOIN user_auth ua ON a.user_id = ua.id
		LEFT JOIN user_profile up ON ua.id = up.user_id
		%s
		ORDER BY %s
		LIMIT ? OFFSET ?`, whereClause, orderBy)

	args = append(args, query.PageSize, offset)
	rows, err := r.db.DB.QueryContext(ctx, listQuery, args...)
	if err != nil {
		r.logger.Error("查询文章列表失败", "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}
	defer rows.Close()

	var articles []models.ArticleListItem
	for rows.Next() {
		var item models.ArticleListItem
		err := rows.Scan(
			&item.ID, &item.Author.ID, &item.Title, &item.Description,
			&item.ViewCount, &item.LikeCount, &item.CommentCount,
			&item.CreatedAt, &item.UpdatedAt,
			&item.Author.Username, &item.Author.Nickname, &item.Author.Avatar)
		if err != nil {
			continue
		}

		// 获取分类
		catQuery := `SELECT ac.id, ac.name, ac.slug
					 FROM article_categories ac
					 INNER JOIN article_category_relations acr ON ac.id = acr.category_id
					 WHERE acr.article_id = ?`
		catRows, err := r.db.DB.QueryContext(ctx, catQuery, item.ID)
		if err == nil {
			for catRows.Next() {
				var cat models.ArticleCategory
				if err := catRows.Scan(&cat.ID, &cat.Name, &cat.Slug); err == nil {
					item.Categories = append(item.Categories, cat)
				}
			}
			catRows.Close()
		}

		// 获取标签
		tagQuery := `SELECT at.id, at.name, at.slug
					 FROM article_tags at
					 INNER JOIN article_tag_relations atr ON at.id = atr.tag_id
					 WHERE atr.article_id = ?`
		tagRows, err := r.db.DB.QueryContext(ctx, tagQuery, item.ID)
		if err == nil {
			for tagRows.Next() {
				var tag models.ArticleTag
				if err := tagRows.Scan(&tag.ID, &tag.Name, &tag.Slug); err == nil {
					item.Tags = append(item.Tags, tag)
				}
			}
			tagRows.Close()
		}

		articles = append(articles, item)
	}

	totalPages := (total + query.PageSize - 1) / query.PageSize
	response := &models.ArticleListResponse{
		Articles:   articles,
		Total:      total,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: totalPages,
	}

	r.logger.Info("获取文章列表成功", "total", total, "page", query.Page, "duration", time.Since(start))
	return response, nil
}

// UpdateArticle 更新文章
func (r *ArticleRepository) UpdateArticle(ctx context.Context, articleID, userID uint, req models.UpdateArticleRequest) error {
	start := time.Now()
	r.logger.Debug("开始更新文章", "articleID", articleID, "userID", userID)

	// 检查文章是否存在且属于当前用户
	checkQuery := `SELECT user_id FROM articles WHERE id = ? AND status != 2`
	var ownerID uint
	err := r.db.DB.QueryRowContext(ctx, checkQuery, articleID).Scan(&ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return utils.ErrUserNotFound
		}
		return utils.ErrDatabaseQuery
	}
	if ownerID != userID {
		return utils.ErrUnauthorized
	}

	// 开启事务
	tx, err := r.db.DB.BeginTx(ctx, nil)
	if err != nil {
		return utils.ErrDatabaseQuery
	}
	defer tx.Rollback()

	// 构建更新语句
	var updates []string
	var args []interface{}

	if req.Title != nil {
		updates = append(updates, "title = ?")
		args = append(args, *req.Title)
	}
	if req.Description != nil {
		updates = append(updates, "description = ?")
		args = append(args, *req.Description)
	}
	if req.Content != nil {
		updates = append(updates, "content = ?")
		args = append(args, *req.Content)
	}
	if req.Status != nil {
		updates = append(updates, "status = ?")
		args = append(args, *req.Status)
	}

	if len(updates) > 0 {
		updates = append(updates, "updated_at = ?")
		args = append(args, time.Now())
		args = append(args, articleID)

		updateQuery := fmt.Sprintf("UPDATE articles SET %s WHERE id = ?", strings.Join(updates, ", "))
		_, err := tx.ExecContext(ctx, updateQuery, args...)
		if err != nil {
			r.logger.Error("更新文章失败", "error", err.Error())
			return utils.ErrDatabaseUpdate
		}
	}

	// 更新代码块（先删除再插入）
	if req.CodeBlocks != nil {
		_, _ = tx.ExecContext(ctx, "DELETE FROM article_code_blocks WHERE article_id = ?", articleID)
		for _, block := range req.CodeBlocks {
			blockQuery := `INSERT INTO article_code_blocks (article_id, language, code_content, description, order_index, created_at)
						   VALUES (?, ?, ?, ?, ?, ?)`
			_, err := tx.ExecContext(ctx, blockQuery,
				articleID, block.Language, block.CodeContent, block.Description, block.OrderIndex, time.Now())
			if err != nil {
				r.logger.Error("插入代码块失败", "error", err.Error())
				return utils.ErrDatabaseInsert
			}
		}
	}

	// 更新分类关联
	if req.CategoryIDs != nil {
		// 先删除旧关联
		_, _ = tx.ExecContext(ctx, "DELETE FROM article_category_relations WHERE article_id = ?", articleID)
		// 插入新关联
		for _, catID := range req.CategoryIDs {
			_, _ = tx.ExecContext(ctx, "INSERT INTO article_category_relations (article_id, category_id, created_at) VALUES (?, ?, ?)",
				articleID, catID, time.Now())
		}
	}

	// 更新标签关联
	if req.TagIDs != nil {
		// 先删除旧关联
		_, _ = tx.ExecContext(ctx, "DELETE FROM article_tag_relations WHERE article_id = ?", articleID)
		// 插入新关联
		for _, tagID := range req.TagIDs {
			_, _ = tx.ExecContext(ctx, "INSERT INTO article_tag_relations (article_id, tag_id, created_at) VALUES (?, ?, ?)",
				articleID, tagID, time.Now())
		}
	}

	if err := tx.Commit(); err != nil {
		r.logger.Error("提交事务失败", "error", err.Error())
		return utils.ErrDatabaseQuery
	}

	r.logger.Info("更新文章成功", "articleID", articleID, "duration", time.Since(start))
	return nil
}

// DeleteArticle 删除文章（软删除）
func (r *ArticleRepository) DeleteArticle(ctx context.Context, articleID, userID uint) error {
	start := time.Now()

	// 检查文章所有权
	checkQuery := `SELECT user_id FROM articles WHERE id = ? AND status != 2`
	var ownerID uint
	err := r.db.DB.QueryRowContext(ctx, checkQuery, articleID).Scan(&ownerID)
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
	query := `UPDATE articles SET status = 2, updated_at = ? WHERE id = ?`
	_, err = r.db.DB.ExecContext(ctx, query, time.Now(), articleID)
	if err != nil {
		r.logger.Error("删除文章失败", "error", err.Error())
		return utils.ErrDatabaseUpdate
	}

	r.logger.Info("删除文章成功", "articleID", articleID, "duration", time.Since(start))
	return nil
}

// ToggleArticleLike 切换文章点赞
func (r *ArticleRepository) ToggleArticleLike(ctx context.Context, articleID, userID uint) (bool, error) {
	start := time.Now()

	// 检查是否已点赞
	checkQuery := `SELECT id FROM article_likes WHERE article_id = ? AND user_id = ?`
	var likeID uint
	err := r.db.DB.QueryRowContext(ctx, checkQuery, articleID, userID).Scan(&likeID)

	isLiked := false
	if err == sql.ErrNoRows {
		// 未点赞，执行点赞
		insertQuery := `INSERT INTO article_likes (article_id, user_id, created_at) VALUES (?, ?, ?)`
		_, err := r.db.DB.ExecContext(ctx, insertQuery, articleID, userID, time.Now())
		if err != nil {
			r.logger.Error("点赞失败", "error", err.Error())
			return false, utils.ErrDatabaseInsert
		}
		// 更新文章点赞数
		_, _ = r.db.DB.ExecContext(ctx, `UPDATE articles SET like_count = like_count + 1 WHERE id = ?`, articleID)
		isLiked = true
	} else if err == nil {
		// 已点赞，取消点赞
		deleteQuery := `DELETE FROM article_likes WHERE article_id = ? AND user_id = ?`
		_, err := r.db.DB.ExecContext(ctx, deleteQuery, articleID, userID)
		if err != nil {
			r.logger.Error("取消点赞失败", "error", err.Error())
			return false, utils.ErrDatabaseUpdate
		}
		// 更新文章点赞数
		_, _ = r.db.DB.ExecContext(ctx, `UPDATE articles SET like_count = GREATEST(like_count - 1, 0) WHERE id = ?`, articleID)
		isLiked = false
	} else {
		return false, utils.ErrDatabaseQuery
	}

	r.logger.Info("切换文章点赞成功", "articleID", articleID, "userID", userID, "isLiked", isLiked, "duration", time.Since(start))
	return isLiked, nil
}

// IncrementViewCount 增加浏览次数
func (r *ArticleRepository) IncrementViewCount(ctx context.Context, articleID uint) error {
	query := `UPDATE articles SET view_count = view_count + 1 WHERE id = ?`
	_, err := r.db.DB.ExecContext(ctx, query, articleID)
	if err != nil {
		r.logger.Error("增加浏览次数失败", "articleID", articleID, "error", err.Error())
		return utils.ErrDatabaseUpdate
	}
	return nil
}

// CreateComment 创建评论
func (r *ArticleRepository) CreateComment(ctx context.Context, comment *models.ArticleComment) error {
	start := time.Now()
	r.logger.Debug("开始创建评论", "articleID", comment.ArticleID, "userID", comment.UserID, "parentID", comment.ParentID)

	// 如果是回复评论，需要确定 root_id
	rootID := comment.RootID
	if comment.ParentID > 0 {
		if rootID == 0 {
			// 查询父评论的 root_id
			query := `SELECT COALESCE(root_id, id) FROM article_comments WHERE id = ?`
			err := r.db.DB.QueryRowContext(ctx, query, comment.ParentID).Scan(&rootID)
			if err != nil {
				if err == sql.ErrNoRows {
					return utils.ErrUserNotFound
				}
				return utils.ErrDatabaseQuery
			}
			comment.RootID = rootID
		}
	}

	// 插入评论
	query := `INSERT INTO article_comments (article_id, user_id, parent_id, root_id, reply_to_user_id, content, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.DB.ExecContext(ctx, query,
		comment.ArticleID, comment.UserID, comment.ParentID, comment.RootID,
		comment.ReplyToUserID, comment.Content, comment.CreatedAt, comment.UpdatedAt)
	if err != nil {
		r.logger.Error("插入评论失败", "error", err.Error())
		return utils.ErrDatabaseInsert
	}

	commentID, _ := result.LastInsertId()
	comment.ID = uint(commentID)

	// 更新文章评论数
	_, _ = r.db.DB.ExecContext(ctx, `UPDATE articles SET comment_count = comment_count + 1 WHERE id = ?`, comment.ArticleID)

	// 如果是回复评论，更新父评论的回复数
	if comment.ParentID > 0 {
		_, _ = r.db.DB.ExecContext(ctx, `UPDATE article_comments SET reply_count = reply_count + 1 WHERE id = ?`, comment.ParentID)
	}

	r.logger.Info("创建评论成功", "commentID", comment.ID, "articleID", comment.ArticleID, "duration", time.Since(start))
	return nil
}

// GetComments 获取评论列表
func (r *ArticleRepository) GetComments(ctx context.Context, articleID uint, page, pageSize int, userID uint) (*models.CommentsResponse, error) {
	start := time.Now()

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 查询总数
	var total int
	countQuery := `SELECT COUNT(*) FROM article_comments WHERE article_id = ? AND parent_id = 0 AND status = 1`
	err := r.db.DB.QueryRowContext(ctx, countQuery, articleID).Scan(&total)
	if err != nil {
		return nil, utils.ErrDatabaseQuery
	}

	// 查询一级评论
	query := `SELECT ac.id, ac.article_id, ac.user_id, ac.parent_id, ac.root_id, ac.reply_to_user_id, ac.content, 
					 ac.like_count, ac.reply_count, ac.status, ac.created_at, ac.updated_at,
					 ua.username, COALESCE(up.nickname, ua.username) as nickname, COALESCE(up.avatar_url, '') as avatar
			  FROM article_comments ac
			  INNER JOIN user_auth ua ON ac.user_id = ua.id
			  LEFT JOIN user_profile up ON ua.id = up.user_id
			  WHERE ac.article_id = ? AND ac.parent_id = 0 AND ac.status = 1
			  ORDER BY ac.created_at DESC
			  LIMIT ? OFFSET ?`

	rows, err := r.db.DB.QueryContext(ctx, query, articleID, pageSize, offset)
	if err != nil {
		r.logger.Error("查询评论失败", "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}
	defer rows.Close()

	var comments []models.CommentDetailResponse
	for rows.Next() {
		var comment models.CommentDetailResponse
		err := rows.Scan(
			&comment.ID, &comment.ArticleID, &comment.UserID, &comment.ParentID, &comment.RootID,
			&comment.ReplyToUserID, &comment.Content, &comment.LikeCount, &comment.ReplyCount,
			&comment.Status, &comment.CreatedAt, &comment.UpdatedAt,
			&comment.Author.Username, &comment.Author.Nickname, &comment.Author.Avatar)
		if err != nil {
			continue
		}
		comment.Author.ID = comment.UserID

		// 检查当前用户是否点赞
		if userID > 0 {
			likeQuery := `SELECT COUNT(*) FROM article_comment_likes WHERE comment_id = ? AND user_id = ?`
			var count int
			if err := r.db.DB.QueryRowContext(ctx, likeQuery, comment.ID, userID).Scan(&count); err == nil {
				comment.IsLiked = count > 0
			}
		}

		// 获取子评论
		comment.Replies = r.getChildComments(ctx, comment.ID, userID)

		comments = append(comments, comment)
	}

	totalPages := (total + pageSize - 1) / pageSize
	response := &models.CommentsResponse{
		Comments:   comments,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	r.logger.Info("获取评论列表成功", "articleID", articleID, "total", total, "duration", time.Since(start))
	return response, nil
}

// getChildComments 递归获取子评论
func (r *ArticleRepository) getChildComments(ctx context.Context, parentID uint, userID uint) []models.CommentDetailResponse {
	query := `SELECT ac.id, ac.article_id, ac.user_id, ac.parent_id, ac.root_id, ac.reply_to_user_id, ac.content,
					 ac.like_count, ac.reply_count, ac.status, ac.created_at, ac.updated_at,
					 ua.username, COALESCE(up.nickname, ua.username) as nickname, COALESCE(up.avatar_url, '') as avatar
			  FROM article_comments ac
			  INNER JOIN user_auth ua ON ac.user_id = ua.id
			  LEFT JOIN user_profile up ON ua.id = up.user_id
			  WHERE ac.parent_id = ? AND ac.status = 1
			  ORDER BY ac.created_at ASC
			  LIMIT 50`

	rows, err := r.db.DB.QueryContext(ctx, query, parentID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var replies []models.CommentDetailResponse
	for rows.Next() {
		var reply models.CommentDetailResponse
		err := rows.Scan(
			&reply.ID, &reply.ArticleID, &reply.UserID, &reply.ParentID, &reply.RootID,
			&reply.ReplyToUserID, &reply.Content, &reply.LikeCount, &reply.ReplyCount,
			&reply.Status, &reply.CreatedAt, &reply.UpdatedAt,
			&reply.Author.Username, &reply.Author.Nickname, &reply.Author.Avatar)
		if err != nil {
			continue
		}
		reply.Author.ID = reply.UserID

		// 检查当前用户是否点赞
		if userID > 0 {
			likeQuery := `SELECT COUNT(*) FROM article_comment_likes WHERE comment_id = ? AND user_id = ?`
			var count int
			if err := r.db.DB.QueryRowContext(ctx, likeQuery, reply.ID, userID).Scan(&count); err == nil {
				reply.IsLiked = count > 0
			}
		}

		// 获取回复对象的用户信息
		if reply.ReplyToUserID != nil && *reply.ReplyToUserID > 0 {
			userQuery := `SELECT ua.id, ua.username, COALESCE(up.nickname, ua.username) as nickname, COALESCE(up.avatar_url, '') as avatar
						  FROM user_auth ua
						  LEFT JOIN user_profile up ON ua.id = up.user_id
						  WHERE ua.id = ?`
			var replyToUser models.CommentAuthor
			if err := r.db.DB.QueryRowContext(ctx, userQuery, *reply.ReplyToUserID).Scan(
				&replyToUser.ID, &replyToUser.Username, &replyToUser.Nickname, &replyToUser.Avatar); err == nil {
				reply.ReplyToUser = &replyToUser
			}
		}

		// 递归获取子评论
		reply.Replies = r.getChildComments(ctx, reply.ID, userID)

		replies = append(replies, reply)
	}

	return replies
}

// ToggleCommentLike 切换评论点赞
func (r *ArticleRepository) ToggleCommentLike(ctx context.Context, commentID, userID uint) (bool, error) {
	start := time.Now()

	// 检查是否已点赞
	checkQuery := `SELECT id FROM article_comment_likes WHERE comment_id = ? AND user_id = ?`
	var likeID uint
	err := r.db.DB.QueryRowContext(ctx, checkQuery, commentID, userID).Scan(&likeID)

	isLiked := false
	if err == sql.ErrNoRows {
		// 未点赞，执行点赞
		insertQuery := `INSERT INTO article_comment_likes (comment_id, user_id, created_at) VALUES (?, ?, ?)`
		_, err := r.db.DB.ExecContext(ctx, insertQuery, commentID, userID, time.Now())
		if err != nil {
			r.logger.Error("点赞评论失败", "error", err.Error())
			return false, utils.ErrDatabaseInsert
		}
		// 更新评论点赞数
		_, _ = r.db.DB.ExecContext(ctx, `UPDATE article_comments SET like_count = like_count + 1 WHERE id = ?`, commentID)
		isLiked = true
	} else if err == nil {
		// 已点赞，取消点赞
		deleteQuery := `DELETE FROM article_comment_likes WHERE comment_id = ? AND user_id = ?`
		_, err := r.db.DB.ExecContext(ctx, deleteQuery, commentID, userID)
		if err != nil {
			r.logger.Error("取消点赞评论失败", "error", err.Error())
			return false, utils.ErrDatabaseUpdate
		}
		// 更新评论点赞数
		_, _ = r.db.DB.ExecContext(ctx, `UPDATE article_comments SET like_count = GREATEST(like_count - 1, 0) WHERE id = ?`, commentID)
		isLiked = false
	} else {
		return false, utils.ErrDatabaseQuery
	}

	r.logger.Info("切换评论点赞成功", "commentID", commentID, "userID", userID, "isLiked", isLiked, "duration", time.Since(start))
	return isLiked, nil
}

// DeleteComment 删除评论（软删除）
func (r *ArticleRepository) DeleteComment(ctx context.Context, commentID, userID uint) error {
	start := time.Now()

	// 检查评论所有权
	checkQuery := `SELECT user_id, article_id FROM article_comments WHERE id = ? AND status != 0`
	var ownerID, articleID uint
	err := r.db.DB.QueryRowContext(ctx, checkQuery, commentID).Scan(&ownerID, &articleID)
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
	query := `UPDATE article_comments SET status = 0, updated_at = ? WHERE id = ?`
	_, err = r.db.DB.ExecContext(ctx, query, time.Now(), commentID)
	if err != nil {
		r.logger.Error("删除评论失败", "error", err.Error())
		return utils.ErrDatabaseUpdate
	}

	// 更新文章评论数
	_, _ = r.db.DB.ExecContext(ctx, `UPDATE articles SET comment_count = GREATEST(comment_count - 1, 0) WHERE id = ?`, articleID)

	r.logger.Info("删除评论成功", "commentID", commentID, "duration", time.Since(start))
	return nil
}

// CreateReport 创建举报
func (r *ArticleRepository) CreateReport(ctx context.Context, report *models.ArticleReport) error {
	start := time.Now()
	r.logger.Debug("开始创建举报", "userID", report.UserID, "articleID", report.ArticleID, "commentID", report.CommentID)

	query := `INSERT INTO article_reports (article_id, comment_id, user_id, reason, status, created_at)
			  VALUES (?, ?, ?, ?, 0, ?)`

	result, err := r.db.DB.ExecContext(ctx, query,
		report.ArticleID, report.CommentID, report.UserID, report.Reason, report.CreatedAt)
	if err != nil {
		r.logger.Error("插入举报失败", "error", err.Error())
		return utils.ErrDatabaseInsert
	}

	reportID, _ := result.LastInsertId()
	report.ID = uint(reportID)

	r.logger.Info("创建举报成功", "reportID", report.ID, "duration", time.Since(start))
	return nil
}

// GetAllCategories 获取所有分类
func (r *ArticleRepository) GetAllCategories(ctx context.Context) ([]models.ArticleCategory, error) {
	query := `SELECT id, name, slug, description, parent_id, article_count, sort_order, created_at
			  FROM article_categories ORDER BY sort_order ASC, id ASC`

	rows, err := r.db.DB.QueryContext(ctx, query)
	if err != nil {
		r.logger.Error("查询分类失败", "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}
	defer rows.Close()

	var categories []models.ArticleCategory
	for rows.Next() {
		var cat models.ArticleCategory
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Slug, &cat.Description,
			&cat.ParentID, &cat.ArticleCount, &cat.SortOrder, &cat.CreatedAt); err == nil {
			categories = append(categories, cat)
		}
	}

	return categories, nil
}

// GetAllTags 获取所有标签
func (r *ArticleRepository) GetAllTags(ctx context.Context) ([]models.ArticleTag, error) {
	query := `SELECT id, name, slug, article_count, created_at
			  FROM article_tags ORDER BY article_count DESC, id ASC LIMIT 100`

	rows, err := r.db.DB.QueryContext(ctx, query)
	if err != nil {
		r.logger.Error("查询标签失败", "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}
	defer rows.Close()

	var tags []models.ArticleTag
	for rows.Next() {
		var tag models.ArticleTag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Slug, &tag.ArticleCount, &tag.CreatedAt); err == nil {
			tags = append(tags, tag)
		}
	}

	return tags, nil
}

// CreateOrGetTag 创建或获取标签
func (r *ArticleRepository) CreateOrGetTag(ctx context.Context, tagName string) (uint, error) {
	// 先查询是否存在
	var tagID uint
	query := `SELECT id FROM article_tags WHERE name = ?`
	err := r.db.DB.QueryRowContext(ctx, query, tagName).Scan(&tagID)
	if err == nil {
		return tagID, nil
	}

	// 不存在则创建
	slug := strings.ToLower(strings.ReplaceAll(tagName, " ", "-"))
	insertQuery := `INSERT INTO article_tags (name, slug, created_at) VALUES (?, ?, ?)`
	result, err := r.db.DB.ExecContext(ctx, insertQuery, tagName, slug, time.Now())
	if err != nil {
		r.logger.Error("创建标签失败", "tagName", tagName, "error", err.Error())
		return 0, utils.ErrDatabaseInsert
	}

	id, _ := result.LastInsertId()
	return uint(id), nil
}
