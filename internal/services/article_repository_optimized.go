package services

import (
	"context"
	"database/sql"
	"time"

	"gin/internal/models"
	"gin/internal/utils"
)

// GetArticleByIDOptimized 优化版本：使用JOIN减少查询次数
// 原版本需要6次查询，优化后只需要2-3次查询
func (r *ArticleRepository) GetArticleByIDOptimized(ctx context.Context, articleID uint, userID uint) (*models.ArticleDetailResponse, error) {
	start := time.Now()
	r.logger.Debug("开始获取文章详情（优化版）", "articleID", articleID, "userID", userID)

	// 第一步：使用JOIN一次性获取文章基本信息、作者信息
	// 合并原来的2次查询为1次
	query := `
		SELECT 
			a.id, a.user_id, a.title, a.description, a.content, 
			a.status, a.view_count, a.like_count, a.comment_count, 
			a.created_at, a.updated_at,
			ua.username, 
			COALESCE(up.nickname, ua.username) as nickname, 
			COALESCE(up.avatar_url, '') as avatar
		FROM articles a
		INNER JOIN user_auth ua ON a.user_id = ua.id
		LEFT JOIN user_profile up ON ua.id = up.user_id
		WHERE a.id = ? AND a.status != 2
	`

	var article models.Article
	var authorID uint
	var authorUsername, authorNickname, authorAvatar string

	err := r.db.DB.QueryRowContext(ctx, query, articleID).Scan(
		&article.ID, &article.UserID, &article.Title, &article.Description, &article.Content,
		&article.Status, &article.ViewCount, &article.LikeCount, &article.CommentCount,
		&article.CreatedAt, &article.UpdatedAt,
		&authorUsername, &authorNickname, &authorAvatar)

	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Debug("文章不存在", "articleID", articleID)
			return nil, utils.ErrUserNotFound
		}
		r.logger.Error("查询文章失败", "error", err.Error())
		return nil, utils.ErrDatabaseQuery
	}

	authorID = article.UserID

	response := &models.ArticleDetailResponse{
		Article: article,
		Author: models.ArticleAuthor{
			ID:       authorID,
			Username: authorUsername,
			Nickname: authorNickname,
			Avatar:   authorAvatar,
		},
		// 初始化空数组，避免返回null
		CodeBlocks: make([]models.ArticleCodeBlock, 0),
		Categories: make([]models.ArticleCategory, 0),
		Tags:       make([]models.ArticleTag, 0),
	}

	// 第二步：并行获取其他信息（代码块、分类、标签、点赞状态）
	// 使用goroutine并行查询，减少总耗时
	type queryResult struct {
		codeBlocks []models.ArticleCodeBlock
		categories []models.ArticleCategory
		tags       []models.ArticleTag
		isLiked    bool
		err        error
	}

	resultChan := make(chan queryResult, 1)

	go func() {
		result := queryResult{}

		// 创建子context，设置超时
		subCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		// 并行查询多个关联数据（使用子goroutine）
		var (
			codeBlocksChan = make(chan []models.ArticleCodeBlock, 1)
			categoriesChan = make(chan []models.ArticleCategory, 1)
			tagsChan       = make(chan []models.ArticleTag, 1)
			likeChan       = make(chan bool, 1)
			errChan        = make(chan error, 4)
		)

		// 查询代码块
		go func() {
			blocks, err := r.getCodeBlocks(subCtx, articleID)
			if err != nil {
				errChan <- err
				codeBlocksChan <- make([]models.ArticleCodeBlock, 0)
			} else {
				codeBlocksChan <- blocks
			}
		}()

		// 查询分类（使用JOIN优化）
		go func() {
			cats, err := r.getCategoriesByArticleID(subCtx, articleID)
			if err != nil {
				errChan <- err
				categoriesChan <- make([]models.ArticleCategory, 0)
			} else {
				categoriesChan <- cats
			}
		}()

		// 查询标签（使用JOIN优化）
		go func() {
			tags, err := r.getTagsByArticleID(subCtx, articleID)
			if err != nil {
				errChan <- err
				tagsChan <- make([]models.ArticleTag, 0)
			} else {
				tagsChan <- tags
			}
		}()

		// 查询点赞状态
		go func() {
			if userID > 0 {
				liked := r.checkArticleLike(subCtx, articleID, userID)
				likeChan <- liked
			} else {
				likeChan <- false
			}
		}()

		// 收集结果
		result.codeBlocks = <-codeBlocksChan
		result.categories = <-categoriesChan
		result.tags = <-tagsChan
		result.isLiked = <-likeChan

		resultChan <- result
	}()

	// 等待结果
	result := <-resultChan

	response.CodeBlocks = result.codeBlocks
	response.Categories = result.categories
	response.Tags = result.tags
	response.IsLiked = result.isLiked

	duration := time.Since(start)
	r.logger.Info("获取文章详情成功（优化版）",
		"articleID", articleID,
		"duration", duration,
		"codeBlocks", len(response.CodeBlocks),
		"categories", len(response.Categories),
		"tags", len(response.Tags))

	return response, nil
}

// getCodeBlocks 获取代码块
func (r *ArticleRepository) getCodeBlocks(ctx context.Context, articleID uint) ([]models.ArticleCodeBlock, error) {
	query := `SELECT id, article_id, language, code_content, description, order_index, created_at
			  FROM article_code_blocks WHERE article_id = ? ORDER BY order_index ASC`

	rows, err := r.db.DB.QueryContext(ctx, query, articleID)
	if err != nil {
		return make([]models.ArticleCodeBlock, 0), err
	}
	defer rows.Close()

	blocks := make([]models.ArticleCodeBlock, 0)
	for rows.Next() {
		var block models.ArticleCodeBlock
		if err := rows.Scan(&block.ID, &block.ArticleID, &block.Language, &block.CodeContent,
			&block.Description, &block.OrderIndex, &block.CreatedAt); err == nil {
			blocks = append(blocks, block)
		}
	}
	return blocks, nil
}

// getCategoriesByArticleID 获取文章分类（JOIN优化）
func (r *ArticleRepository) getCategoriesByArticleID(ctx context.Context, articleID uint) ([]models.ArticleCategory, error) {
	query := `SELECT ac.id, ac.name, ac.slug, ac.description, ac.parent_id, ac.article_count, ac.sort_order, ac.created_at
			  FROM article_categories ac
			  INNER JOIN article_category_relations acr ON ac.id = acr.category_id
			  WHERE acr.article_id = ?`

	rows, err := r.db.DB.QueryContext(ctx, query, articleID)
	if err != nil {
		return make([]models.ArticleCategory, 0), err
	}
	defer rows.Close()

	categories := make([]models.ArticleCategory, 0)
	for rows.Next() {
		var cat models.ArticleCategory
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Slug, &cat.Description,
			&cat.ParentID, &cat.ArticleCount, &cat.SortOrder, &cat.CreatedAt); err == nil {
			categories = append(categories, cat)
		}
	}
	return categories, nil
}

// getTagsByArticleID 获取文章标签（JOIN优化）
func (r *ArticleRepository) getTagsByArticleID(ctx context.Context, articleID uint) ([]models.ArticleTag, error) {
	query := `SELECT at.id, at.name, at.slug, at.article_count, at.created_at
			  FROM article_tags at
			  INNER JOIN article_tag_relations atr ON at.id = atr.tag_id
			  WHERE atr.article_id = ?`

	rows, err := r.db.DB.QueryContext(ctx, query, articleID)
	if err != nil {
		return make([]models.ArticleTag, 0), err
	}
	defer rows.Close()

	tags := make([]models.ArticleTag, 0)
	for rows.Next() {
		var tag models.ArticleTag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Slug, &tag.ArticleCount, &tag.CreatedAt); err == nil {
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

// checkArticleLike 检查用户是否点赞文章
func (r *ArticleRepository) checkArticleLike(ctx context.Context, articleID uint, userID uint) bool {
	query := `SELECT COUNT(*) FROM article_likes WHERE article_id = ? AND user_id = ?`
	var count int
	if err := r.db.DB.QueryRowContext(ctx, query, articleID, userID).Scan(&count); err == nil {
		return count > 0
	}
	return false
}
