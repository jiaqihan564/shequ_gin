package services

import (
	"context"
	"database/sql"
	"time"

	"gin/internal/models"
	"gin/internal/utils"
)

// ResourceRepository 资源仓库
type ResourceRepository struct {
	db     *Database
	logger utils.Logger
}

// NewResourceRepository 创建资源仓库
func NewResourceRepository(db *Database) *ResourceRepository {
	return &ResourceRepository{
		db:     db,
		logger: utils.GetLogger(),
	}
}

// CreateResource 创建资源
func (r *ResourceRepository) CreateResource(ctx context.Context, resource *models.Resource, imageURLs []string, tags []string) error {
	// 开启事务
	tx, err := r.db.DB.BeginTx(ctx, nil)
	if err != nil {
		return utils.ErrDatabaseQuery
	}
	defer tx.Rollback()

	// 插入资源主记录
	query := `INSERT INTO resources (user_id, title, description, document, category_id, file_name, 
	          file_size, file_type, file_extension, storage_path, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := tx.ExecContext(ctx, query,
		resource.UserID, resource.Title, resource.Description, resource.Document,
		resource.CategoryID, resource.FileName, resource.FileSize, resource.FileType,
		resource.FileExtension, resource.StoragePath, resource.CreatedAt, resource.UpdatedAt)

	if err != nil {
		r.logger.Error("插入资源失败", "error", err.Error())
		return utils.ErrDatabaseInsert
	}

	resourceID, _ := result.LastInsertId()
	resource.ID = uint(resourceID)

	// 插入预览图
	r.logger.Info("开始插入预览图", "resourceID", resource.ID, "imageCount", len(imageURLs))
	if len(imageURLs) > 0 {
		imgQuery := `INSERT INTO resource_images (resource_id, image_url, image_order, is_cover, created_at) VALUES (?, ?, ?, ?, ?)`
		for i, url := range imageURLs {
			isCover := 0
			if i == 0 {
				isCover = 1 // 第一张设为封面
			}
			r.logger.Info("插入预览图", "index", i, "url", url, "isCover", isCover)
			_, err := tx.ExecContext(ctx, imgQuery, resource.ID, url, i, isCover, time.Now())
			if err != nil {
				r.logger.Error("插入预览图失败", "index", i, "url", url, "error", err.Error())
			} else {
				r.logger.Info("预览图插入成功", "index", i)
			}
		}
	} else {
		r.logger.Warn("没有预览图需要插入")
	}

	// 插入标签
	if len(tags) > 0 {
		tagQuery := `INSERT INTO resource_tags (resource_id, tag_name, created_at) VALUES (?, ?, ?)`
		for _, tag := range tags {
			if tag != "" {
				_, _ = tx.ExecContext(ctx, tagQuery, resource.ID, tag, time.Now())
			}
		}
	}

	// 更新分类资源数
	if resource.CategoryID != nil {
		_, _ = tx.ExecContext(ctx, `UPDATE resource_categories SET resource_count = resource_count + 1 WHERE id = ?`, *resource.CategoryID)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return utils.ErrDatabaseInsert
	}

	r.logger.Info("创建资源成功", "resourceID", resource.ID, "title", resource.Title)
	return nil
}

// GetResourceByID 获取资源详情
func (r *ResourceRepository) GetResourceByID(ctx context.Context, resourceID, userID uint) (*models.ResourceDetailResponse, error) {
	// 查询资源基本信息
	query := `SELECT id, user_id, title, description, document, category_id, file_name, file_size,
	          file_type, file_extension, storage_path, download_count, view_count, like_count,
	          status, created_at, updated_at FROM resources WHERE id = ? AND status != 0`

	var resource models.Resource
	var categoryID sql.NullInt64

	err := r.db.DB.QueryRowContext(ctx, query, resourceID).Scan(
		&resource.ID, &resource.UserID, &resource.Title, &resource.Description,
		&resource.Document, &categoryID, &resource.FileName, &resource.FileSize,
		&resource.FileType, &resource.FileExtension, &resource.StoragePath,
		&resource.DownloadCount, &resource.ViewCount, &resource.LikeCount,
		&resource.Status, &resource.CreatedAt, &resource.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, utils.ErrUserNotFound
		}
		return nil, utils.ErrDatabaseQuery
	}

	if categoryID.Valid {
		id := uint(categoryID.Int64)
		resource.CategoryID = &id
	}

	response := &models.ResourceDetailResponse{
		Resource: resource,
	}

	// 获取作者信息
	authorQuery := `SELECT ua.id, ua.username, COALESCE(up.nickname, ua.username) as nickname, 
	                COALESCE(up.avatar_url, '') as avatar
	                FROM user_auth ua LEFT JOIN user_profile up ON ua.id = up.user_id 
	                WHERE ua.id = ?`
	_ = r.db.DB.QueryRowContext(ctx, authorQuery, resource.UserID).Scan(
		&response.Author.ID, &response.Author.Username,
		&response.Author.Nickname, &response.Author.Avatar,
	)

	// 获取预览图
	imgQuery := `SELECT id, resource_id, image_url, image_order, is_cover, created_at 
	             FROM resource_images WHERE resource_id = ? ORDER BY image_order ASC`
	rows, err := r.db.DB.QueryContext(ctx, imgQuery, resourceID)
	if err != nil {
		r.logger.Warn("查询预览图失败", "resourceID", resourceID, "error", err.Error())
	} else {
		defer rows.Close()
		imageCount := 0
		for rows.Next() {
			var img models.ResourceImage
			var isCover int
			if err := rows.Scan(&img.ID, &img.ResourceID, &img.ImageURL, &img.ImageOrder, &isCover, &img.CreatedAt); err == nil {
				img.IsCover = isCover == 1
				response.Images = append(response.Images, img)
				imageCount++
				r.logger.Info("读取预览图", "resourceID", resourceID, "imageID", img.ID, "url", img.ImageURL, "order", img.ImageOrder)
			} else {
				r.logger.Error("扫描预览图数据失败", "error", err.Error())
			}
		}
		r.logger.Info("预览图加载完成", "resourceID", resourceID, "count", imageCount)
	}

	// 获取分类信息
	if resource.CategoryID != nil {
		catQuery := `SELECT id, name, slug, description, resource_count, created_at FROM resource_categories WHERE id = ?`
		var cat models.ResourceCategory
		if err := r.db.DB.QueryRowContext(ctx, catQuery, *resource.CategoryID).Scan(
			&cat.ID, &cat.Name, &cat.Slug, &cat.Description, &cat.ResourceCount, &cat.CreatedAt,
		); err == nil {
			response.Category = &cat
		}
	}

	// 获取标签
	tagQuery := `SELECT tag_name FROM resource_tags WHERE resource_id = ?`
	tagRows, err := r.db.DB.QueryContext(ctx, tagQuery, resourceID)
	if err == nil {
		defer tagRows.Close()
		for tagRows.Next() {
			var tag string
			if err := tagRows.Scan(&tag); err == nil {
				response.Tags = append(response.Tags, tag)
			}
		}
	}

	// 检查当前用户是否点赞
	if userID > 0 {
		likeQuery := `SELECT id FROM resource_likes WHERE resource_id = ? AND user_id = ?`
		var likeID uint
		err := r.db.DB.QueryRowContext(ctx, likeQuery, resourceID, userID).Scan(&likeID)
		response.IsLiked = (err == nil)
	}

	return response, nil
}

// ListResources 获取资源列表
func (r *ResourceRepository) ListResources(ctx context.Context, query models.ResourceListQuery) (*models.ResourceListResponse, error) {
	// 构建查询条件
	whereClause := "WHERE r.status = 1"
	var args []interface{}

	if query.CategoryID != nil {
		whereClause += " AND r.category_id = ?"
		args = append(args, *query.CategoryID)
	}

	if query.UserID != nil {
		whereClause += " AND r.user_id = ?"
		args = append(args, *query.UserID)
	}

	if query.Keyword != "" {
		whereClause += " AND (r.title LIKE ? OR r.description LIKE ?)"
		keyword := "%" + query.Keyword + "%"
		args = append(args, keyword, keyword)
	}

	// 排序
	orderBy := "ORDER BY r.created_at DESC"
	switch query.SortBy {
	case "popular":
		orderBy = "ORDER BY r.view_count DESC, r.like_count DESC"
	case "downloads":
		orderBy = "ORDER BY r.download_count DESC"
	}

	// 查询总数
	countQuery := "SELECT COUNT(*) FROM resources r " + whereClause
	var total int
	err := r.db.DB.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, utils.ErrDatabaseQuery
	}

	// 分页
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 20
	}
	offset := (query.Page - 1) * query.PageSize

	// 查询列表
	listQuery := `SELECT r.id, r.user_id, r.title, r.description, r.category_id, r.file_name,
	              r.file_size, r.file_extension, r.download_count, r.view_count, r.like_count, r.created_at
	              FROM resources r ` + whereClause + ` ` + orderBy + ` LIMIT ? OFFSET ?`
	args = append(args, query.PageSize, offset)

	rows, err := r.db.DB.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, utils.ErrDatabaseQuery
	}
	defer rows.Close()

	var resources []models.ResourceListItem
	for rows.Next() {
		var item models.ResourceListItem
		var categoryID sql.NullInt64

		err := rows.Scan(
			&item.ID, &item.Author.ID, &item.Title, &item.Description, &categoryID,
			&item.FileName, &item.FileSize, &item.FileExtension,
			&item.DownloadCount, &item.ViewCount, &item.LikeCount, &item.CreatedAt,
		)
		if err != nil {
			continue
		}

		// 获取作者信息
		authorQuery := `SELECT ua.username, COALESCE(up.nickname, ua.username) as nickname, 
		                COALESCE(up.avatar_url, '') as avatar
		                FROM user_auth ua LEFT JOIN user_profile up ON ua.id = up.user_id 
		                WHERE ua.id = ?`
		_ = r.db.DB.QueryRowContext(ctx, authorQuery, item.Author.ID).Scan(
			&item.Author.Username, &item.Author.Nickname, &item.Author.Avatar,
		)

		// 获取封面图
		imgQuery := `SELECT image_url FROM resource_images WHERE resource_id = ? AND is_cover = 1 LIMIT 1`
		_ = r.db.DB.QueryRowContext(ctx, imgQuery, item.ID).Scan(&item.CoverImage)

		// 获取分类
		if categoryID.Valid {
			catQuery := `SELECT id, name, slug FROM resource_categories WHERE id = ?`
			var cat models.ResourceCategory
			if err := r.db.DB.QueryRowContext(ctx, catQuery, uint(categoryID.Int64)).Scan(&cat.ID, &cat.Name, &cat.Slug); err == nil {
				item.Category = &cat
			}
		}

		resources = append(resources, item)
	}

	totalPages := (total + query.PageSize - 1) / query.PageSize

	return &models.ResourceListResponse{
		Resources:  resources,
		Total:      total,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: totalPages,
	}, nil
}

// ToggleResourceLike 切换资源点赞
func (r *ResourceRepository) ToggleResourceLike(ctx context.Context, resourceID, userID uint) (bool, error) {
	// 检查是否已点赞
	checkQuery := `SELECT id FROM resource_likes WHERE resource_id = ? AND user_id = ?`
	var likeID uint
	err := r.db.DB.QueryRowContext(ctx, checkQuery, resourceID, userID).Scan(&likeID)

	isLiked := false
	switch err {
	case sql.ErrNoRows:
		// 未点赞，执行点赞
		_, err := r.db.DB.ExecContext(ctx, `INSERT INTO resource_likes (resource_id, user_id, created_at) VALUES (?, ?, ?)`,
			resourceID, userID, time.Now())
		if err != nil {
			return false, utils.ErrDatabaseInsert
		}
		_, _ = r.db.DB.ExecContext(ctx, `UPDATE resources SET like_count = like_count + 1 WHERE id = ?`, resourceID)
		isLiked = true
	case nil:
		// 已点赞，取消点赞
		_, err := r.db.DB.ExecContext(ctx, `DELETE FROM resource_likes WHERE resource_id = ? AND user_id = ?`, resourceID, userID)
		if err != nil {
			return false, utils.ErrDatabaseUpdate
		}
		_, _ = r.db.DB.ExecContext(ctx, `UPDATE resources SET like_count = GREATEST(like_count - 1, 0) WHERE id = ?`, resourceID)
		isLiked = false
	default:
		return false, utils.ErrDatabaseQuery
	}

	return isLiked, nil
}

// IncrementDownloadCount 增加下载次数
func (r *ResourceRepository) IncrementDownloadCount(ctx context.Context, resourceID uint) error {
	_, err := r.db.DB.ExecContext(ctx, `UPDATE resources SET download_count = download_count + 1 WHERE id = ?`, resourceID)
	return err
}

// IncrementViewCount 增加浏览次数
func (r *ResourceRepository) IncrementViewCount(ctx context.Context, resourceID uint) error {
	_, err := r.db.DB.ExecContext(ctx, `UPDATE resources SET view_count = view_count + 1 WHERE id = ?`, resourceID)
	return err
}

// DeleteResource 删除资源
func (r *ResourceRepository) DeleteResource(ctx context.Context, resourceID, userID uint) error {
	// 检查所有权
	var ownerID uint
	err := r.db.DB.QueryRowContext(ctx, `SELECT user_id FROM resources WHERE id = ? AND status != 0`, resourceID).Scan(&ownerID)
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
	_, err = r.db.DB.ExecContext(ctx, `UPDATE resources SET status = 0, updated_at = ? WHERE id = ?`, time.Now(), resourceID)
	return err
}

// GetAllCategories 获取所有资源分类
func (r *ResourceRepository) GetAllCategories(ctx context.Context) ([]models.ResourceCategory, error) {
	query := `SELECT id, name, slug, description, resource_count, created_at FROM resource_categories ORDER BY id ASC`
	rows, err := r.db.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.ResourceCategory
	for rows.Next() {
		var cat models.ResourceCategory
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Slug, &cat.Description, &cat.ResourceCount, &cat.CreatedAt); err == nil {
			categories = append(categories, cat)
		}
	}

	return categories, nil
}
