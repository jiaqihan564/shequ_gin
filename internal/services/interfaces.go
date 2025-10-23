package services

import (
	"context"
	"io"
	"time"

	"gin/internal/models"
)

// AuthServiceInterface 认证服务接口
type AuthServiceInterface interface {
	Login(ctx context.Context, username, password, clientIP, province, city string) (*models.LoginResponse, error)
	Register(ctx context.Context, username, password, email, clientIP, userAgent, province, city string) (*models.LoginResponse, error)
	ChangePassword(ctx context.Context, userID uint, currentPassword, newPassword string) error
	ForgotPassword(ctx context.Context, email string) (string, error)
	ResetPassword(ctx context.Context, token, newPassword string) error
}

// UserServiceInterface 用户服务接口
type UserServiceInterface interface {
	GetUserByID(ctx context.Context, id uint) (*models.User, error)
	GetUserProfile(ctx context.Context, userID uint) (*models.UserExtraProfile, error)
	UpsertUserProfile(ctx context.Context, profile *models.UserExtraProfile) error
	UpdateUserAvatar(ctx context.Context, profile *models.UserExtraProfile) error
}

// StorageClient 存储客户端接口（用于头像上传等场景）
type StorageClient interface {
	PutObject(ctx context.Context, objectPath string, contentType string, reader io.Reader, size int64) (string, error)
	ObjectExists(ctx context.Context, objectPath string) (bool, error)
	CopyObject(ctx context.Context, srcPath, dstPath string) error
	RemoveObject(ctx context.Context, objectPath string) error
	ListObjects(ctx context.Context, prefix string) ([]ObjectInfo, error)
	GetPublicBaseURL() string
}

// ObjectInfo 对象元信息（用于列举）
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
}

// ArticleRepositoryInterface 文章管理操作接口
type ArticleRepositoryInterface interface {
	// 文章CRUD
	CreateArticle(ctx context.Context, article *models.Article, codeBlocks []models.ArticleCodeBlock, categoryIDs, tagIDs []uint) error
	GetArticleByID(ctx context.Context, articleID uint, userID uint) (*models.ArticleDetailResponse, error)
	ListArticles(ctx context.Context, query models.ArticleListQuery) (*models.ArticleListResponse, error)
	UpdateArticle(ctx context.Context, articleID uint, userID uint, req models.UpdateArticleRequest) error
	DeleteArticle(ctx context.Context, articleID uint, userID uint) error

	// 文章交互
	ToggleArticleLike(ctx context.Context, articleID uint, userID uint) (bool, error)
	IncrementViewCount(ctx context.Context, articleID uint) error

	// 评论
	CreateComment(ctx context.Context, comment *models.ArticleComment) error
	GetComments(ctx context.Context, articleID uint, page, pageSize int, userID uint) (*models.CommentsResponse, error)
	ToggleCommentLike(ctx context.Context, commentID uint, userID uint) (bool, error)
	DeleteComment(ctx context.Context, commentID uint, userID uint) error

	// 分类和标签
	GetAllCategories(ctx context.Context) ([]models.ArticleCategory, error)
	GetAllTags(ctx context.Context) ([]models.ArticleTag, error)
	CreateOrGetTag(ctx context.Context, tagName string) (uint, error)

	// 举报
	CreateReport(ctx context.Context, report *models.ArticleReport) error
}

// UserRepositoryInterface 用户管理操作接口
type UserRepositoryInterface interface {
	// 用户CRUD
	GetUserByID(ctx context.Context, id uint) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	CreateUser(ctx context.Context, user *models.User) error
	UpdateUser(ctx context.Context, user *models.User) error

	// 资料管理
	GetUserProfile(ctx context.Context, userID uint) (*models.UserExtraProfile, error)
	UpdateUserProfile(ctx context.Context, profile *models.UserExtraProfile) error

	// 认证
	UpdateLastLogin(ctx context.Context, userID uint, ip string) error
	IncrementFailedLoginCount(ctx context.Context, userID uint) error
	ResetFailedLoginCount(ctx context.Context, userID uint) error
}

// CacheServiceInterface 缓存操作接口
type CacheServiceInterface interface {
	// 文章缓存
	GetArticleCategories(ctx context.Context) ([]models.ArticleCategory, error)
	GetArticleTags(ctx context.Context) ([]models.ArticleTag, error)
	GetArticleDetail(ctx context.Context, articleID uint, userID uint) (*models.ArticleDetailResponse, error)

	// 缓存失效
	InvalidateArticleCategories()
	InvalidateArticleTags()
	InvalidateArticleDetail(articleID uint)

	// 在线人数
	SetOnlineCount(count int)
	GetOnlineCount() (int, bool)

	// 统计
	GetCacheStats() map[string]interface{}
	GetAllCacheStats() map[string]interface{}
	ClearAllCache()
}

// BatchRepositoryInterface 批量查询操作接口（解决N+1问题）
type BatchRepositoryInterface interface {
	BatchGetUsers(ctx context.Context, userIDs []uint) (map[uint]*models.User, error)
	BatchGetArticles(ctx context.Context, articleIDs []uint) (map[uint]*models.Article, error)
	BatchGetUserProfiles(ctx context.Context, userIDs []uint) (map[uint]*BatchUserProfile, error)
	BatchCheckArticleLikes(ctx context.Context, articleIDs []uint, userID uint) (map[uint]bool, error)
	BatchGetCommentCounts(ctx context.Context, articleIDs []uint) (map[uint]int, error)
}
