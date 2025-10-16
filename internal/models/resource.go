package models

import "time"

// Resource 资源主表
type Resource struct {
	ID            uint      `json:"id" db:"id"`
	UserID        uint      `json:"user_id" db:"user_id"`
	Title         string    `json:"title" db:"title"`
	Description   string    `json:"description" db:"description"`
	Document      string    `json:"document" db:"document"`
	CategoryID    *uint     `json:"category_id" db:"category_id"`
	FileName      string    `json:"file_name" db:"file_name"`
	FileSize      int64     `json:"file_size" db:"file_size"`
	FileType      string    `json:"file_type" db:"file_type"`
	FileExtension string    `json:"file_extension" db:"file_extension"`
	StoragePath   string    `json:"storage_path" db:"storage_path"`
	DownloadCount int       `json:"download_count" db:"download_count"`
	ViewCount     int       `json:"view_count" db:"view_count"`
	LikeCount     int       `json:"like_count" db:"like_count"`
	Status        int       `json:"status" db:"status"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// ResourceImage 资源预览图
type ResourceImage struct {
	ID         uint      `json:"id" db:"id"`
	ResourceID uint      `json:"resource_id" db:"resource_id"`
	ImageURL   string    `json:"image_url" db:"image_url"`
	ImageOrder int       `json:"image_order" db:"image_order"`
	IsCover    bool      `json:"is_cover" db:"is_cover"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// ResourceCategory 资源分类
type ResourceCategory struct {
	ID            uint      `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	Slug          string    `json:"slug" db:"slug"`
	Description   string    `json:"description" db:"description"`
	ResourceCount int       `json:"resource_count" db:"resource_count"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// ResourceTag 资源标签
type ResourceTag struct {
	ID         uint      `json:"id" db:"id"`
	ResourceID uint      `json:"resource_id" db:"resource_id"`
	TagName    string    `json:"tag_name" db:"tag_name"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// UploadChunk 断点续传记录
type UploadChunk struct {
	ID             uint      `json:"id" db:"id"`
	UploadID       string    `json:"upload_id" db:"upload_id"`
	UserID         uint      `json:"user_id" db:"user_id"`
	FileName       string    `json:"file_name" db:"file_name"`
	FileSize       int64     `json:"file_size" db:"file_size"`
	ChunkSize      int       `json:"chunk_size" db:"chunk_size"`
	TotalChunks    int       `json:"total_chunks" db:"total_chunks"`
	UploadedChunks string    `json:"uploaded_chunks" db:"uploaded_chunks"` // JSON数组
	StoragePath    string    `json:"storage_path" db:"storage_path"`
	Status         int       `json:"status" db:"status"`
	ExpiresAt      time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// ========== 请求/响应 DTO ==========

// ResourceAuthor 资源上传者信息
type ResourceAuthor struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// CreateResourceRequest 创建资源请求
type CreateResourceRequest struct {
	Title       string   `json:"title" binding:"required,min=1,max=200"`
	Description string   `json:"description" binding:"max=1000"`
	Document    string   `json:"document"`
	CategoryID  *uint    `json:"category_id"`
	FileName    string   `json:"file_name" binding:"required"`
	FileSize    int64    `json:"file_size" binding:"required"`
	FileType    string   `json:"file_type"`
	StoragePath string   `json:"storage_path" binding:"required"`
	ImageURLs   []string `json:"image_urls"` // 预览图URL列表
	Tags        []string `json:"tags"`       // 标签列表
}

// UpdateResourceRequest 更新资源请求
type UpdateResourceRequest struct {
	Title       *string  `json:"title" binding:"omitempty,min=1,max=200"`
	Description *string  `json:"description" binding:"omitempty,max=1000"`
	Document    *string  `json:"document"`
	CategoryID  *uint    `json:"category_id"`
	ImageURLs   []string `json:"image_urls"`
	Tags        []string `json:"tags"`
}

// ResourceDetailResponse 资源详情响应
type ResourceDetailResponse struct {
	Resource
	Author   ResourceAuthor    `json:"author"`
	Images   []ResourceImage   `json:"images"`
	Category *ResourceCategory `json:"category"`
	Tags     []string          `json:"tags"`
	IsLiked  bool              `json:"is_liked"`
}

// ResourceListItem 资源列表项
type ResourceListItem struct {
	ID            uint              `json:"id"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	Author        ResourceAuthor    `json:"author"`
	Category      *ResourceCategory `json:"category"`
	CoverImage    string            `json:"cover_image"`
	FileName      string            `json:"file_name"`
	FileSize      int64             `json:"file_size"`
	FileExtension string            `json:"file_extension"`
	DownloadCount int               `json:"download_count"`
	ViewCount     int               `json:"view_count"`
	LikeCount     int               `json:"like_count"`
	CreatedAt     time.Time         `json:"created_at"`
}

// ResourceListResponse 资源列表响应
type ResourceListResponse struct {
	Resources  []ResourceListItem `json:"resources"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
}

// InitUploadRequest 初始化上传请求
type InitUploadRequest struct {
	FileName    string `json:"file_name" binding:"required"`
	FileSize    int64  `json:"file_size" binding:"required"`
	TotalChunks int    `json:"total_chunks" binding:"required"`
	UploadID    string `json:"upload_id" binding:"required"` // 文件MD5
}

// InitUploadResponse 初始化上传响应
type InitUploadResponse struct {
	UploadID       string `json:"upload_id"`
	UploadedChunks []int  `json:"uploaded_chunks"` // 已上传的分片索引
	ChunkSize      int    `json:"chunk_size"`
}

// UploadChunkRequest 上传分片请求（multipart）
// 通过 form-data 传递

// MergeChunksRequest 合并分片请求
type MergeChunksRequest struct {
	UploadID string `json:"upload_id" binding:"required"`
}

// MergeChunksResponse 合并分片响应
type MergeChunksResponse struct {
	StoragePath string `json:"storage_path"`
	FileURL     string `json:"file_url"`
}

// ResourceListQuery 资源列表查询参数
type ResourceListQuery struct {
	Page       int    `form:"page,default=1"`
	PageSize   int    `form:"page_size,default=20"`
	CategoryID *uint  `form:"category_id"`
	Keyword    string `form:"keyword"`
	SortBy     string `form:"sort_by,default=latest"` // latest, popular, downloads
	UserID     *uint  `form:"user_id"`                // 查询指定用户的资源
}

// ========== 资源评论相关模型 ==========

// ResourceComment 资源评论
type ResourceComment struct {
	ID            uint      `json:"id" db:"id"`
	ResourceID    uint      `json:"resource_id" db:"resource_id"`
	UserID        uint      `json:"user_id" db:"user_id"`
	ParentID      uint      `json:"parent_id" db:"parent_id"`
	RootID        uint      `json:"root_id" db:"root_id"`
	ReplyToUserID *uint     `json:"reply_to_user_id" db:"reply_to_user_id"`
	Content       string    `json:"content" db:"content"`
	LikeCount     int       `json:"like_count" db:"like_count"`
	ReplyCount    int       `json:"reply_count" db:"reply_count"`
	Status        int       `json:"status" db:"status"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// CommentUser 评论用户信息
type CommentUser struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// ResourceCommentResponse 资源评论响应
type ResourceCommentResponse struct {
	ID          uint                      `json:"id"`
	ResourceID  uint                      `json:"resource_id"`
	UserID      uint                      `json:"user_id"`
	ParentID    uint                      `json:"parent_id"`
	RootID      uint                      `json:"root_id"`
	Content     string                    `json:"content"`
	LikeCount   int                       `json:"like_count"`
	ReplyCount  int                       `json:"reply_count"`
	IsLiked     bool                      `json:"is_liked"`
	User        *CommentUser              `json:"user"`
	ReplyToUser *CommentUser              `json:"reply_to_user,omitempty"`
	Replies     []ResourceCommentResponse `json:"replies,omitempty"`
	CreatedAt   time.Time                 `json:"created_at"`
}

// CreateResourceCommentRequest 创建资源评论请求
type CreateResourceCommentRequest struct {
	Content       string `json:"content" binding:"required,min=1,max=500"`
	ParentID      *uint  `json:"parent_id"`
	ReplyToUserID *uint  `json:"reply_to_user_id"`
}

// ResourceCommentsResponse 资源评论列表响应
type ResourceCommentsResponse struct {
	Comments   []ResourceCommentResponse `json:"comments"`
	Total      int                       `json:"total"`
	Page       int                       `json:"page"`
	PageSize   int                       `json:"page_size"`
	TotalPages int                       `json:"total_pages"`
}
