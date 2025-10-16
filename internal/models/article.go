package models

import "time"

// Article 文章结构体
type Article struct {
	ID           uint      `json:"id" db:"id"`
	UserID       uint      `json:"user_id" db:"user_id"`
	Title        string    `json:"title" db:"title"`
	Description  string    `json:"description" db:"description"`
	Content      string    `json:"content" db:"content"`
	Status       int       `json:"status" db:"status"` // 0-草稿，1-已发布，2-已删除
	ViewCount    int       `json:"view_count" db:"view_count"`
	LikeCount    int       `json:"like_count" db:"like_count"`
	CommentCount int       `json:"comment_count" db:"comment_count"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// ArticleCodeBlock 代码块结构体
type ArticleCodeBlock struct {
	ID          uint      `json:"id" db:"id"`
	ArticleID   uint      `json:"article_id" db:"article_id"`
	Language    string    `json:"language" db:"language"`
	CodeContent string    `json:"code_content" db:"code_content"`
	Description string    `json:"description" db:"description"`
	OrderIndex  int       `json:"order_index" db:"order_index"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// ArticleCategory 分类结构体
type ArticleCategory struct {
	ID           uint      `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	Slug         string    `json:"slug" db:"slug"`
	Description  string    `json:"description" db:"description"`
	ParentID     uint      `json:"parent_id" db:"parent_id"`
	ArticleCount int       `json:"article_count" db:"article_count"`
	SortOrder    int       `json:"sort_order" db:"sort_order"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// ArticleTag 标签结构体
type ArticleTag struct {
	ID           uint      `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	Slug         string    `json:"slug" db:"slug"`
	ArticleCount int       `json:"article_count" db:"article_count"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// ArticleComment 评论结构体
type ArticleComment struct {
	ID            uint      `json:"id" db:"id"`
	ArticleID     uint      `json:"article_id" db:"article_id"`
	UserID        uint      `json:"user_id" db:"user_id"`
	ParentID      uint      `json:"parent_id" db:"parent_id"`
	RootID        uint      `json:"root_id" db:"root_id"`
	ReplyToUserID *uint     `json:"reply_to_user_id" db:"reply_to_user_id"`
	Content       string    `json:"content" db:"content"`
	LikeCount     int       `json:"like_count" db:"like_count"`
	ReplyCount    int       `json:"reply_count" db:"reply_count"`
	Status        int       `json:"status" db:"status"` // 0-已删除，1-正常，2-已折叠
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// ArticleLike 文章点赞结构体
type ArticleLike struct {
	ID        uint      `json:"id" db:"id"`
	ArticleID uint      `json:"article_id" db:"article_id"`
	UserID    uint      `json:"user_id" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ArticleCommentLike 评论点赞结构体
type ArticleCommentLike struct {
	ID        uint      `json:"id" db:"id"`
	CommentID uint      `json:"comment_id" db:"comment_id"`
	UserID    uint      `json:"user_id" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ArticleReport 举报结构体
type ArticleReport struct {
	ID          uint       `json:"id" db:"id"`
	ArticleID   *uint      `json:"article_id" db:"article_id"`
	CommentID   *uint      `json:"comment_id" db:"comment_id"`
	UserID      uint       `json:"user_id" db:"user_id"`
	Reason      string     `json:"reason" db:"reason"`
	Status      int        `json:"status" db:"status"` // 0-待处理，1-已处理，2-已驳回
	HandlerID   *uint      `json:"handler_id" db:"handler_id"`
	HandlerNote string     `json:"handler_note" db:"handler_note"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	HandledAt   *time.Time `json:"handled_at" db:"handled_at"`
}

// ========== 请求/响应 DTO ==========

// ArticleAuthor 文章作者信息
type ArticleAuthor struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// CommentAuthor 评论作者信息
type CommentAuthor struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// ArticleDetailResponse 文章详情响应
type ArticleDetailResponse struct {
	Article
	Author     ArticleAuthor      `json:"author"`
	CodeBlocks []ArticleCodeBlock `json:"code_blocks"`
	Categories []ArticleCategory  `json:"categories"`
	Tags       []ArticleTag       `json:"tags"`
	IsLiked    bool               `json:"is_liked"`
}

// ArticleListItem 文章列表项
type ArticleListItem struct {
	ID           uint              `json:"id"`
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	Author       ArticleAuthor     `json:"author"`
	Categories   []ArticleCategory `json:"categories"`
	Tags         []ArticleTag      `json:"tags"`
	ViewCount    int               `json:"view_count"`
	LikeCount    int               `json:"like_count"`
	CommentCount int               `json:"comment_count"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// ArticleListResponse 文章列表响应
type ArticleListResponse struct {
	Articles   []ArticleListItem `json:"articles"`
	Total      int               `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}

// CreateArticleRequest 创建文章请求
type CreateArticleRequest struct {
	Title       string                   `json:"title" binding:"required,min=1,max=200"`
	Description string                   `json:"description" binding:"max=500"`
	Content     string                   `json:"content" binding:"required"`
	Status      int                      `json:"status"`                                // 0-草稿，1-发布
	CodeBlocks  []CreateArticleCodeBlock `json:"code_blocks"`                           // 代码块列表
	CategoryIDs []uint                   `json:"category_ids" binding:"required,min=1"` // 分类ID列表
	TagIDs      []uint                   `json:"tag_ids"`                               // 标签ID列表（可选）
	TagNames    []string                 `json:"tag_names"`                             // 新标签名称列表（自动创建）
}

// CreateArticleCodeBlock 创建文章代码块
type CreateArticleCodeBlock struct {
	Language    string `json:"language" binding:"required"`
	CodeContent string `json:"code_content" binding:"required"`
	Description string `json:"description"`
	OrderIndex  int    `json:"order_index"`
}

// UpdateArticleRequest 更新文章请求
type UpdateArticleRequest struct {
	Title       *string                  `json:"title" binding:"omitempty,min=1,max=200"`
	Description *string                  `json:"description" binding:"omitempty,max=500"`
	Content     *string                  `json:"content" binding:"omitempty,min=1"`
	Status      *int                     `json:"status"`
	CodeBlocks  []CreateArticleCodeBlock `json:"code_blocks"`
	CategoryIDs []uint                   `json:"category_ids"`
	TagIDs      []uint                   `json:"tag_ids"`
	TagNames    []string                 `json:"tag_names"`
}

// CreateCommentRequest 创建评论请求
type CreateCommentRequest struct {
	Content       string `json:"content" binding:"required,min=1,max=1000"`
	ParentID      uint   `json:"parent_id"`        // 父评论ID（0表示一级评论）
	ReplyToUserID *uint  `json:"reply_to_user_id"` // 回复的用户ID
}

// CommentDetailResponse 评论详情响应
type CommentDetailResponse struct {
	ArticleComment
	Author      CommentAuthor           `json:"author"`
	ReplyToUser *CommentAuthor          `json:"reply_to_user,omitempty"` // 回复的用户信息
	Replies     []CommentDetailResponse `json:"replies"`                 // 子评论列表
	IsLiked     bool                    `json:"is_liked"`                // 当前用户是否点赞
}

// CommentsResponse 评论列表响应
type CommentsResponse struct {
	Comments   []CommentDetailResponse `json:"comments"`
	Total      int                     `json:"total"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"page_size"`
	TotalPages int                     `json:"total_pages"`
}

// CreateReportRequest 创建举报请求
type CreateReportRequest struct {
	ArticleID *uint  `json:"article_id"`
	CommentID *uint  `json:"comment_id"`
	Reason    string `json:"reason" binding:"required,min=10,max=500"`
}

// ArticleListQuery 文章列表查询参数
type ArticleListQuery struct {
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
	CategoryID uint   `form:"category_id"`
	TagID      uint   `form:"tag_id"`
	UserID     uint   `form:"user_id"`
	Status     int    `form:"status"`
	Keyword    string `form:"keyword"`
	SortBy     string `form:"sort_by"` // latest, hot, popular
}
