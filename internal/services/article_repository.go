package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"gin/internal/config"
	"gin/internal/models"
	"gin/internal/utils"
)

// ArticleRepository 文章仓库
type ArticleRepository struct {
	db     *Database
	logger utils.Logger
	config *config.Config
}

// NewArticleRepository 创建文章仓库
func NewArticleRepository(db *Database, cfg *config.Config) *ArticleRepository {
	return &ArticleRepository{
		db:     db,
		logger: utils.GetLogger(),
		config: cfg,
	}
}

// CreateArticle 创建文章
func (r *ArticleRepository) CreateArticle(ctx context.Context, article *models.Article, codeBlocks []models.CreateArticleCodeBlock, categoryIDs, tagIDs []uint) error {
	start := time.Now().UTC()
	r.logger.Info("开始创建文章",
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

	// 2. 批量插入代码块（优化性能）
	if len(codeBlocks) > 0 {
		blockQuery := `INSERT INTO article_code_blocks (article_id, language, code_content, description, order_index, created_at) VALUES `
		blockValues := []string{}
		blockArgs := []interface{}{}
		now := time.Now().UTC()

		for _, block := range codeBlocks {
			blockValues = append(blockValues, "(?, ?, ?, ?, ?, ?)")
			blockArgs = append(blockArgs, article.ID, block.Language, block.CodeContent, block.Description, block.OrderIndex, now)
		}

		blockQuery += strings.Join(blockValues, ", ")
		_, err := tx.ExecContext(ctx, blockQuery, blockArgs...)
		if err != nil {
			r.logger.Error("批量插入代码块失败", "error", err.Error())
			return utils.ErrDatabaseInsert
		}
	}

	// 3. 批量关联分类（优化性能）
	if len(categoryIDs) > 0 {
		catQuery := `INSERT INTO article_category_relations (article_id, category_id, created_at) VALUES `
		catValues := []string{}
		catArgs := []interface{}{}
		now := time.Now().UTC()

		for _, catID := range categoryIDs {
			catValues = append(catValues, "(?, ?, ?)")
			catArgs = append(catArgs, article.ID, catID, now)
		}

		catQuery += strings.Join(catValues, ", ")
		_, err := tx.ExecContext(ctx, catQuery, catArgs...)
		if err != nil {
			r.logger.Error("批量关联分类失败", "error", err.Error())
			return utils.ErrDatabaseInsert
		}

		// 批量更新分类文章计数
		updateCatQuery := `UPDATE article_categories SET article_count = article_count + 1 WHERE id IN (?` + strings.Repeat(",?", len(categoryIDs)-1) + `)`
		updateCatArgs := make([]interface{}, len(categoryIDs))
		for i, catID := range categoryIDs {
			updateCatArgs[i] = catID
		}
		_, _ = tx.ExecContext(ctx, updateCatQuery, updateCatArgs...)
	}

	// 4. 批量关联标签（优化性能）
	if len(tagIDs) > 0 {
		tagQuery := `INSERT INTO article_tag_relations (article_id, tag_id, created_at) VALUES `
		tagValues := []string{}
		tagArgs := []interface{}{}
		now := time.Now().UTC()

		for _, tagID := range tagIDs {
			tagValues = append(tagValues, "(?, ?, ?)")
			tagArgs = append(tagArgs, article.ID, tagID, now)
		}

		tagQuery += strings.Join(tagValues, ", ")
		_, err := tx.ExecContext(ctx, tagQuery, tagArgs...)
		if err != nil {
			r.logger.Error("批量关联标签失败", "error", err.Error())
			return utils.ErrDatabaseInsert
		}

		// 批量更新标签文章计数
		updateTagQuery := `UPDATE article_tags SET article_count = article_count + 1 WHERE id IN (?` + strings.Repeat(",?", len(tagIDs)-1) + `)`
		updateTagArgs := make([]interface{}, len(tagIDs))
		for i, tagID := range tagIDs {
			updateTagArgs[i] = tagID
		}
		_, _ = tx.ExecContext(ctx, updateTagQuery, updateTagArgs...)
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

// GetArticleByID 根据ID获取文章详情（优化版本：使用JOIN减少查询次数）
// 原版本需要6次查询，优化后只需要2-3次查询
func (r *ArticleRepository) GetArticleByID(ctx context.Context, articleID uint, userID uint) (*models.ArticleDetailResponse, error) {
	start := time.Now().UTC()

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
	}

	resultChan := make(chan queryResult, 1)

	go func() {
		result := queryResult{}

		// 创建子context，设置超时
		subCtx, cancel := context.WithTimeout(ctx, r.db.GetAsyncTaskTimeout())
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

// ListArticles 获取文章列表
func (r *ArticleRepository) ListArticles(ctx context.Context, query models.ArticleListQuery) (*models.ArticleListResponse, error) {
	start := time.Now().UTC()

	// 设置默认值（从配置读取）
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 || query.PageSize > r.config.Pagination.MaxPageSize {
		query.PageSize = r.config.Pagination.DefaultPageSize
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

	// 并行执行COUNT和列表查询（优化性能）
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM articles a %s", whereClause)
	listQuery := fmt.Sprintf(`
		SELECT a.id, a.user_id, a.title, a.description, a.view_count, a.like_count, a.comment_count, a.created_at, a.updated_at,
			   ua.username, COALESCE(up.nickname, ua.username) as nickname, COALESCE(up.avatar_url, '') as avatar
		FROM articles a
		INNER JOIN user_auth ua ON a.user_id = ua.id
		LEFT JOIN user_profile up ON ua.id = up.user_id
		%s
		ORDER BY %s
		LIMIT ? OFFSET ?`, whereClause, orderBy)

	// 准备参数
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	listArgs := append(args, query.PageSize, offset)

	// 并行查询
	type countResult struct {
		total int
		err   error
	}
	type listResult struct {
		rows *sql.Rows
		err  error
	}

	countChan := make(chan countResult, 1)
	listChan := make(chan listResult, 1)

	// 并行执行COUNT
	go func() {
		var total int
		err := r.db.DB.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
		countChan <- countResult{total: total, err: err}
	}()

	// 并行执行列表查询
	go func() {
		rows, err := r.db.DB.QueryContext(ctx, listQuery, listArgs...)
		listChan <- listResult{rows: rows, err: err}
	}()

	// 收集结果
	countRes := <-countChan
	listRes := <-listChan

	if countRes.err != nil {
		if listRes.rows != nil {
			listRes.rows.Close()
		}
		r.logger.Error("查询文章总数失败", "error", countRes.err.Error())
		return nil, utils.ErrDatabaseQuery
	}

	if listRes.err != nil {
		r.logger.Error("查询文章列表失败", "error", listRes.err.Error())
		return nil, utils.ErrDatabaseQuery
	}

	total := countRes.total
	rows := listRes.rows
	defer rows.Close()

	// 预分配容量（性能优化）
	articles := make([]models.ArticleListItem, 0, query.PageSize)
	articleIDs := make([]uint, 0, query.PageSize)
	articleMap := make(map[uint]*models.ArticleListItem, query.PageSize)

	// 第一步：收集所有文章信息
	for rows.Next() {
		var item models.ArticleListItem
		// 初始化空数组，避免返回null
		item.Categories = make([]models.ArticleCategory, 0)
		item.Tags = make([]models.ArticleTag, 0)

		err := rows.Scan(
			&item.ID, &item.Author.ID, &item.Title, &item.Description,
			&item.ViewCount, &item.LikeCount, &item.CommentCount,
			&item.CreatedAt, &item.UpdatedAt,
			&item.Author.Username, &item.Author.Nickname, &item.Author.Avatar)
		if err != nil {
			continue
		}

		articleIDs = append(articleIDs, item.ID)
		articles = append(articles, item)
		articleMap[item.ID] = &articles[len(articles)-1]
	}

	// 如果有文章，批量查询分类和标签（解决N+1问题）
	if len(articleIDs) > 0 {
		// 第二步：批量查询所有文章的分类
		catQuery := `SELECT acr.article_id, ac.id, ac.name, ac.slug
					 FROM article_categories ac
					 INNER JOIN article_category_relations acr ON ac.id = acr.category_id
					 WHERE acr.article_id IN (?` + strings.Repeat(",?", len(articleIDs)-1) + `)`

		catArgs := make([]interface{}, len(articleIDs))
		for i, id := range articleIDs {
			catArgs[i] = id
		}

		catRows, err := r.db.DB.QueryContext(ctx, catQuery, catArgs...)
		if err == nil {
			defer catRows.Close()
			for catRows.Next() {
				var articleID uint
				var cat models.ArticleCategory
				if err := catRows.Scan(&articleID, &cat.ID, &cat.Name, &cat.Slug); err == nil {
					if item, exists := articleMap[articleID]; exists {
						item.Categories = append(item.Categories, cat)
					}
				}
			}
		}

		// 第三步：批量查询所有文章的标签
		tagQuery := `SELECT atr.article_id, at.id, at.name, at.slug
					 FROM article_tags at
					 INNER JOIN article_tag_relations atr ON at.id = atr.tag_id
					 WHERE atr.article_id IN (?` + strings.Repeat(",?", len(articleIDs)-1) + `)`

		tagArgs := make([]interface{}, len(articleIDs))
		for i, id := range articleIDs {
			tagArgs[i] = id
		}

		tagRows, err := r.db.DB.QueryContext(ctx, tagQuery, tagArgs...)
		if err == nil {
			defer tagRows.Close()
			for tagRows.Next() {
				var articleID uint
				var tag models.ArticleTag
				if err := tagRows.Scan(&articleID, &tag.ID, &tag.Name, &tag.Slug); err == nil {
					if item, exists := articleMap[articleID]; exists {
						item.Tags = append(item.Tags, tag)
					}
				}
			}
		}
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
	start := time.Now().UTC()

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
		args = append(args, time.Now().UTC())
		args = append(args, articleID)

		updateQuery := fmt.Sprintf("UPDATE articles SET %s WHERE id = ?", strings.Join(updates, ", "))
		_, err := tx.ExecContext(ctx, updateQuery, args...)
		if err != nil {
			r.logger.Error("更新文章失败", "error", err.Error())
			return utils.ErrDatabaseUpdate
		}
	}

	// 更新代码块（先删除再批量插入）
	if req.CodeBlocks != nil {
		_, _ = tx.ExecContext(ctx, "DELETE FROM article_code_blocks WHERE article_id = ?", articleID)
		if len(req.CodeBlocks) > 0 {
			blockQuery := `INSERT INTO article_code_blocks (article_id, language, code_content, description, order_index, created_at) VALUES `
			blockValues := []string{}
			blockArgs := []interface{}{}
			now := time.Now().UTC()

			for _, block := range req.CodeBlocks {
				blockValues = append(blockValues, "(?, ?, ?, ?, ?, ?)")
				blockArgs = append(blockArgs, articleID, block.Language, block.CodeContent, block.Description, block.OrderIndex, now)
			}

			blockQuery += strings.Join(blockValues, ", ")
			_, err := tx.ExecContext(ctx, blockQuery, blockArgs...)
			if err != nil {
				r.logger.Error("批量插入代码块失败", "error", err.Error())
				return utils.ErrDatabaseInsert
			}
		}
	}

	// 更新分类关联（批量插入优化）
	if req.CategoryIDs != nil {
		// 先删除旧关联
		_, _ = tx.ExecContext(ctx, "DELETE FROM article_category_relations WHERE article_id = ?", articleID)
		// 批量插入新关联
		if len(req.CategoryIDs) > 0 {
			catQuery := `INSERT INTO article_category_relations (article_id, category_id, created_at) VALUES `
			catValues := []string{}
			catArgs := []interface{}{}
			now := time.Now().UTC()

			for _, catID := range req.CategoryIDs {
				catValues = append(catValues, "(?, ?, ?)")
				catArgs = append(catArgs, articleID, catID, now)
			}

			catQuery += strings.Join(catValues, ", ")
			_, _ = tx.ExecContext(ctx, catQuery, catArgs...)
		}
	}

	// 更新标签关联（批量插入优化）
	if req.TagIDs != nil {
		// 先删除旧关联
		_, _ = tx.ExecContext(ctx, "DELETE FROM article_tag_relations WHERE article_id = ?", articleID)
		// 批量插入新关联
		if len(req.TagIDs) > 0 {
			tagQuery := `INSERT INTO article_tag_relations (article_id, tag_id, created_at) VALUES `
			tagValues := []string{}
			tagArgs := []interface{}{}
			now := time.Now().UTC()

			for _, tagID := range req.TagIDs {
				tagValues = append(tagValues, "(?, ?, ?)")
				tagArgs = append(tagArgs, articleID, tagID, now)
			}

			tagQuery += strings.Join(tagValues, ", ")
			_, _ = tx.ExecContext(ctx, tagQuery, tagArgs...)
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
	start := time.Now().UTC()

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
	_, err = r.db.DB.ExecContext(ctx, query, time.Now().UTC(), articleID)
	if err != nil {
		r.logger.Error("删除文章失败", "error", err.Error())
		return utils.ErrDatabaseUpdate
	}

	r.logger.Info("删除文章成功", "articleID", articleID, "duration", time.Since(start))
	return nil
}

// ToggleArticleLike 切换文章点赞
func (r *ArticleRepository) ToggleArticleLike(ctx context.Context, articleID, userID uint) (bool, error) {
	start := time.Now().UTC()

	// 检查是否已点赞
	checkQuery := `SELECT id FROM article_likes WHERE article_id = ? AND user_id = ?`
	var likeID uint
	err := r.db.DB.QueryRowContext(ctx, checkQuery, articleID, userID).Scan(&likeID)

	isLiked := false
	switch err {
	case sql.ErrNoRows:
		// 未点赞，执行点赞
		insertQuery := `INSERT INTO article_likes (article_id, user_id, created_at) VALUES (?, ?, ?)`
		_, err := r.db.DB.ExecContext(ctx, insertQuery, articleID, userID, time.Now().UTC())
		if err != nil {
			r.logger.Error("点赞失败", "error", err.Error())
			return false, utils.ErrDatabaseInsert
		}
		// 更新文章点赞数
		_, _ = r.db.DB.ExecContext(ctx, `UPDATE articles SET like_count = like_count + 1 WHERE id = ?`, articleID)
		isLiked = true
	case nil:
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
	default:
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
	start := time.Now().UTC()

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
	start := time.Now().UTC()

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > r.config.Pagination.MaxPageSize {
		pageSize = r.config.Pagination.DefaultPageSize
	}
	offset := (page - 1) * pageSize

	// 并行执行COUNT和评论列表查询
	countQuery := `SELECT COUNT(*) FROM article_comments WHERE article_id = ? AND parent_id = 0 AND status = 1`
	listQuery := `SELECT ac.id, ac.article_id, ac.user_id, ac.parent_id, ac.root_id, ac.reply_to_user_id, ac.content, 
					 ac.like_count, ac.reply_count, ac.status, ac.created_at, ac.updated_at,
					 ua.username, COALESCE(up.nickname, ua.username) as nickname, COALESCE(up.avatar_url, '') as avatar
			  FROM article_comments ac
			  INNER JOIN user_auth ua ON ac.user_id = ua.id
			  LEFT JOIN user_profile up ON ua.id = up.user_id
			  WHERE ac.article_id = ? AND ac.parent_id = 0 AND ac.status = 1
			  ORDER BY ac.created_at DESC
			  LIMIT ? OFFSET ?`

	type countResult struct {
		total int
		err   error
	}
	type listResult struct {
		rows *sql.Rows
		err  error
	}

	countChan := make(chan countResult, 1)
	listChan := make(chan listResult, 1)

	// 并行执行
	go func() {
		var total int
		err := r.db.DB.QueryRowContext(ctx, countQuery, articleID).Scan(&total)
		countChan <- countResult{total: total, err: err}
	}()

	go func() {
		rows, err := r.db.DB.QueryContext(ctx, listQuery, articleID, pageSize, offset)
		listChan <- listResult{rows: rows, err: err}
	}()

	// 收集结果
	countRes := <-countChan
	listRes := <-listChan

	if countRes.err != nil {
		if listRes.rows != nil {
			listRes.rows.Close()
		}
		return nil, utils.ErrDatabaseQuery
	}

	if listRes.err != nil {
		r.logger.Error("查询评论失败", "error", listRes.err.Error())
		return nil, utils.ErrDatabaseQuery
	}

	total := countRes.total
	rows := listRes.rows
	defer rows.Close()

	// 初始化为空数组，避免返回null
	comments := make([]models.CommentDetailResponse, 0)
	commentIDs := make([]uint, 0)

	// 第一步：收集所有一级评论
	for rows.Next() {
		var comment models.CommentDetailResponse
		// 初始化Replies为空数组
		comment.Replies = make([]models.CommentDetailResponse, 0)
		err := rows.Scan(
			&comment.ID, &comment.ArticleID, &comment.UserID, &comment.ParentID, &comment.RootID,
			&comment.ReplyToUserID, &comment.Content, &comment.LikeCount, &comment.ReplyCount,
			&comment.Status, &comment.CreatedAt, &comment.UpdatedAt,
			&comment.Author.Username, &comment.Author.Nickname, &comment.Author.Avatar)
		if err != nil {
			continue
		}
		comment.Author.ID = comment.UserID
		commentIDs = append(commentIDs, comment.ID)
		comments = append(comments, comment)
	}

	// 如果没有评论，直接返回
	if len(comments) == 0 {
		return &models.CommentsResponse{
			Comments:   comments,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: (total + pageSize - 1) / pageSize,
		}, nil
	}

	// 第二步：批量检查用户点赞状态（优化N+1）
	if userID > 0 {
		likedMap := r.batchCheckCommentLikes(ctx, commentIDs, userID)
		for i := range comments {
			comments[i].IsLiked = likedMap[comments[i].ID]
		}
	}

	// 第三步：批量获取所有子评论（优化递归N+1）
	childCommentsMap := r.batchGetChildComments(ctx, articleID, commentIDs, userID)
	r.logger.Info("批量获取文章子评论", "commentCount", len(commentIDs), "childMapSize", len(childCommentsMap))
	for i := range comments {
		// 确保所有评论都有 Replies 字段（即使为空数组）
		if children, exists := childCommentsMap[comments[i].ID]; exists && len(children) > 0 {
			comments[i].Replies = children
			r.logger.Info("设置评论的子回复", "commentID", comments[i].ID, "repliesCount", len(children))
		} else {
			comments[i].Replies = make([]models.CommentDetailResponse, 0)
		}
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

// batchCheckCommentLikes 批量检查评论点赞状态（优化N+1）
func (r *ArticleRepository) batchCheckCommentLikes(ctx context.Context, commentIDs []uint, userID uint) map[uint]bool {
	likedMap := make(map[uint]bool, len(commentIDs)) // 预分配容量

	if len(commentIDs) == 0 || userID == 0 {
		return likedMap
	}

	// 初始化所有为false
	for _, id := range commentIDs {
		likedMap[id] = false
	}

	// 批量查询点赞记录
	query := `SELECT comment_id FROM article_comment_likes 
	          WHERE comment_id IN (?` + strings.Repeat(",?", len(commentIDs)-1) + `) AND user_id = ?`

	args := make([]interface{}, len(commentIDs)+1)
	for i, id := range commentIDs {
		args[i] = id
	}
	args[len(commentIDs)] = userID

	rows, err := r.db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return likedMap
	}
	defer rows.Close()

	for rows.Next() {
		var commentID uint
		if err := rows.Scan(&commentID); err == nil {
			likedMap[commentID] = true
		}
	}

	return likedMap
}

// batchGetChildComments 批量获取子评论（优化递归N+1）
func (r *ArticleRepository) batchGetChildComments(ctx context.Context, articleID uint, parentIDs []uint, userID uint) map[uint][]models.CommentDetailResponse {
	childMap := make(map[uint][]models.CommentDetailResponse, len(parentIDs)) // 预分配容量

	if len(parentIDs) == 0 {
		return childMap
	}

	// 初始化所有父评论的子评论列表为空数组
	for _, id := range parentIDs {
		childMap[id] = make([]models.CommentDetailResponse, 0)
	}

	// 一次性查询文章的所有子评论（包括所有层级）
	query := `SELECT ac.id, ac.article_id, ac.user_id, ac.parent_id, ac.root_id, ac.reply_to_user_id, ac.content,
					 ac.like_count, ac.reply_count, ac.status, ac.created_at, ac.updated_at,
					 ua.username, COALESCE(up.nickname, ua.username) as nickname, COALESCE(up.avatar_url, '') as avatar
			  FROM article_comments ac
			  INNER JOIN user_auth ua ON ac.user_id = ua.id
			  LEFT JOIN user_profile up ON ua.id = up.user_id
			  WHERE ac.article_id = ? AND ac.parent_id > 0 AND ac.status = 1
			  ORDER BY ac.created_at ASC`

	rows, err := r.db.DB.QueryContext(ctx, query, articleID)
	if err != nil {
		return childMap
	}
	defer rows.Close()

	// 收集所有子评论
	allChildren := make([]models.CommentDetailResponse, 0)
	childIDs := make([]uint, 0)
	replyToUserIDs := make([]uint, 0)

	for rows.Next() {
		var child models.CommentDetailResponse
		child.Replies = make([]models.CommentDetailResponse, 0)

		err := rows.Scan(
			&child.ID, &child.ArticleID, &child.UserID, &child.ParentID, &child.RootID,
			&child.ReplyToUserID, &child.Content, &child.LikeCount, &child.ReplyCount,
			&child.Status, &child.CreatedAt, &child.UpdatedAt,
			&child.Author.Username, &child.Author.Nickname, &child.Author.Avatar)
		if err != nil {
			continue
		}
		child.Author.ID = child.UserID

		allChildren = append(allChildren, child)
		childIDs = append(childIDs, child.ID)
		if child.ReplyToUserID != nil && *child.ReplyToUserID > 0 {
			replyToUserIDs = append(replyToUserIDs, *child.ReplyToUserID)
		}
	}

	if len(allChildren) == 0 {
		return childMap
	}

	// 批量检查子评论的点赞状态
	if userID > 0 && len(childIDs) > 0 {
		childLikedMap := r.batchCheckCommentLikes(ctx, childIDs, userID)
		for i := range allChildren {
			allChildren[i].IsLiked = childLikedMap[allChildren[i].ID]
		}
	}

	// 批量查询被回复用户的信息
	if len(replyToUserIDs) > 0 {
		replyToUserMap := r.batchGetCommentUsers(ctx, replyToUserIDs)
		for i := range allChildren {
			if allChildren[i].ReplyToUserID != nil && *allChildren[i].ReplyToUserID > 0 {
				if user, exists := replyToUserMap[*allChildren[i].ReplyToUserID]; exists {
					allChildren[i].ReplyToUser = user
				}
			}
		}
	}

	// 构建评论树（在内存中组装）
	// 按parent_id分组（预分配容量）
	commentsByParent := make(map[uint][]models.CommentDetailResponse, len(parentIDs))
	for i := range allChildren {
		// 确保每个评论都有Replies字段初始化
		if allChildren[i].Replies == nil {
			allChildren[i].Replies = make([]models.CommentDetailResponse, 0)
		}
		parentID := allChildren[i].ParentID
		commentsByParent[parentID] = append(commentsByParent[parentID], allChildren[i])
	}

	r.logger.Info("开始组装文章评论树", "totalChildren", len(allChildren), "topLevelParents", len(childMap))

	// 递归函数：为评论填充其子回复
	var fillReplies func(*models.CommentDetailResponse)
	fillReplies = func(comment *models.CommentDetailResponse) {
		if children, exists := commentsByParent[comment.ID]; exists {
			comment.Replies = make([]models.CommentDetailResponse, len(children))
			copy(comment.Replies, children)
			// 递归为每个子评论填充其子回复
			for i := range comment.Replies {
				fillReplies(&comment.Replies[i])
			}
			r.logger.Info("填充评论的子回复", "commentID", comment.ID, "repliesCount", len(comment.Replies))
		}
	}

	// 为所有一级评论填充子回复树
	for parentID, children := range commentsByParent {
		if _, isTopLevel := childMap[parentID]; isTopLevel {
			childMap[parentID] = make([]models.CommentDetailResponse, len(children))
			copy(childMap[parentID], children)
			// 为每个一级子评论递归填充其子回复
			for i := range childMap[parentID] {
				fillReplies(&childMap[parentID][i])
			}
			r.logger.Info("设置一级评论的子回复", "parentID", parentID, "childrenCount", len(children))
		}
	}

	r.logger.Info("完成组装文章评论树", "childMapSize", len(childMap))
	return childMap
}

// batchGetCommentUsers 批量获取评论用户信息
func (r *ArticleRepository) batchGetCommentUsers(ctx context.Context, userIDs []uint) map[uint]*models.CommentAuthor {
	userMap := make(map[uint]*models.CommentAuthor, len(userIDs)) // 预分配容量

	if len(userIDs) == 0 {
		return userMap
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
		user := &models.CommentAuthor{}
		if err := rows.Scan(&user.ID, &user.Username, &user.Nickname, &user.Avatar); err == nil {
			userMap[user.ID] = user
		}
	}

	return userMap
}

// ToggleCommentLike 切换评论点赞
func (r *ArticleRepository) ToggleCommentLike(ctx context.Context, commentID, userID uint) (bool, error) {
	start := time.Now().UTC()

	// 检查是否已点赞
	checkQuery := `SELECT id FROM article_comment_likes WHERE comment_id = ? AND user_id = ?`
	var likeID uint
	err := r.db.DB.QueryRowContext(ctx, checkQuery, commentID, userID).Scan(&likeID)

	isLiked := false
	switch err {
	case sql.ErrNoRows:
		// 未点赞，执行点赞
		insertQuery := `INSERT INTO article_comment_likes (comment_id, user_id, created_at) VALUES (?, ?, ?)`
		_, err := r.db.DB.ExecContext(ctx, insertQuery, commentID, userID, time.Now().UTC())
		if err != nil {
			r.logger.Error("点赞评论失败", "error", err.Error())
			return false, utils.ErrDatabaseInsert
		}
		// 更新评论点赞数
		_, _ = r.db.DB.ExecContext(ctx, `UPDATE article_comments SET like_count = like_count + 1 WHERE id = ?`, commentID)
		isLiked = true
	case nil:
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
	default:
		return false, utils.ErrDatabaseQuery
	}

	r.logger.Info("切换评论点赞成功", "commentID", commentID, "userID", userID, "isLiked", isLiked, "duration", time.Since(start))
	return isLiked, nil
}

// DeleteComment 删除评论（软删除）
func (r *ArticleRepository) DeleteComment(ctx context.Context, commentID, userID uint) error {
	start := time.Now().UTC()

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
	_, err = r.db.DB.ExecContext(ctx, query, time.Now().UTC(), commentID)
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
	start := time.Now().UTC()

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
	query := fmt.Sprintf(`SELECT id, name, slug, article_count, created_at
			  FROM article_tags ORDER BY article_count DESC, id ASC LIMIT %d`, r.config.StatisticsQuery.TagsListLimit)

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
	result, err := r.db.DB.ExecContext(ctx, insertQuery, tagName, slug, time.Now().UTC())
	if err != nil {
		r.logger.Error("创建标签失败", "tagName", tagName, "error", err.Error())
		return 0, utils.ErrDatabaseInsert
	}

	id, _ := result.LastInsertId()
	return uint(id), nil
}

// getCodeBlocks 获取代码块（辅助方法）
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

// getCategoriesByArticleID 获取文章分类（JOIN优化，辅助方法）
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

// getTagsByArticleID 获取文章标签（JOIN优化，辅助方法）
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

// checkArticleLike 检查用户是否点赞文章（辅助方法）
func (r *ArticleRepository) checkArticleLike(ctx context.Context, articleID uint, userID uint) bool {
	query := `SELECT COUNT(*) FROM article_likes WHERE article_id = ? AND user_id = ?`
	var count int
	if err := r.db.DB.QueryRowContext(ctx, query, articleID, userID).Scan(&count); err == nil {
		return count > 0
	}
	return false
}
