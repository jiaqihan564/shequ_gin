package services

import (
	"context"
	"io"
	"time"

	"gin/internal/models"
)

// AuthServiceInterface 认证服务接口
type AuthServiceInterface interface {
	Login(ctx context.Context, username, password, clientIP string) (*models.LoginResponse, error)
	Register(ctx context.Context, username, password, email string) (*models.LoginResponse, error)
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
