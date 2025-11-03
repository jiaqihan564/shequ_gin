package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"gin/internal/config"
	"gin/internal/utils"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ResourceStorageService 资源存储服务（独立MinIO桶）
type ResourceStorageService struct {
	client *minio.Client
	cfg    *config.Config
	logger utils.Logger
}

// NewResourceStorageService 初始化资源存储服务
func NewResourceStorageService(cfg *config.Config) (*ResourceStorageService, error) {
	logger := utils.GetLogger()

	// 初始化MinIO客户端
	cli, err := minio.New(cfg.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKeyID, cfg.MinIO.SecretAccessKey, ""),
		Secure: cfg.MinIO.UseSSL,
	})
	if err != nil {
		logger.Error("初始化资源存储MinIO客户端失败", "error", err.Error())
		return nil, err
	}

	// 确保资源桶存在
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.MinIO.OperationTimeout)*time.Second)
	defer cancel()

	bucketName := cfg.ResourcesStorage.Bucket
	exists, err := cli.BucketExists(ctx, bucketName)
	if err != nil {
		logger.Error("检查资源桶失败", "bucket", bucketName, "error", err.Error())
		return nil, err
	}

	if !exists {
		if err := cli.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
			logger.Error("创建资源桶失败", "bucket", bucketName, "error", err.Error())
			return nil, err
		}
		logger.Info("已创建资源桶", "bucket", bucketName)
	}

	// 设置桶策略为公开只读（从配置读取策略参数）
	policy := fmt.Sprintf(`{
		"Version": "%s",
		"Statement": [
			{
				"Sid": "PublicReadGetObject",
				"Effect": "%s",
				"Principal": "*",
				"Action": "%s",
				"Resource": "arn:aws:s3:::%s/*"
			}
		]
	}`, cfg.MinioAdvanced.PolicyVersion, cfg.MinioAdvanced.PolicyEffect, cfg.MinioAdvanced.PolicyAction, bucketName)

	if err := cli.SetBucketPolicy(ctx, bucketName, policy); err != nil {
		logger.Warn("设置资源桶公开访问策略失败",
			"bucket", bucketName,
			"error", err.Error(),
			"suggestion", "请手动在MinIO控制台设置桶为Public访问")
	} else {
		logger.Info("资源桶策略已设置为公开只读", "bucket", bucketName)
	}

	return &ResourceStorageService{client: cli, cfg: cfg, logger: logger}, nil
}

// UploadResourceImage 上传资源预览图到临时目录
func (s *ResourceStorageService) UploadResourceImage(ctx context.Context, file io.Reader, filename string, size int64) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("资源存储客户端未初始化")
	}

	// 生成UUID作为临时目录名
	tempID := uuid.New().String()

	// 获取文件扩展名
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".jpg" // 默认扩展名
	}

	// 构建临时路径: images/temp/{uuid}/{filename}
	objectPath := fmt.Sprintf("images/temp/%s/%s", tempID, filename)

	return s.uploadFile(ctx, file, objectPath, size, "image/jpeg")
}

// UploadDocumentImage 上传文档图片（永久存储）
func (s *ResourceStorageService) UploadDocumentImage(ctx context.Context, file io.Reader, filename string, size int64) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("资源存储客户端未初始化")
	}

	// 构建路径: documents/{year}/{month}/{uuid}_{filename}
	now := time.Now().UTC()
	uniqueID := uuid.New().String()[:8]
	objectPath := fmt.Sprintf("documents/%d/%02d/%s_%s", now.Year(), now.Month(), uniqueID, filename)

	return s.uploadFile(ctx, file, objectPath, size, "image/jpeg")
}

// MoveResourceImages 将临时图片移动到正式目录
func (s *ResourceStorageService) MoveResourceImages(ctx context.Context, tempURLs []string, resourceID uint) ([]string, error) {
	if s.client == nil {
		return nil, fmt.Errorf("资源存储客户端未初始化")
	}

	var finalURLs []string
	bucket := s.cfg.ResourcesStorage.Bucket
	baseURL := s.cfg.ResourcesStorage.PublicBaseURL

	for i, tempURL := range tempURLs {
		// 提取临时路径（去掉base URL）
		tempPath := strings.TrimPrefix(tempURL, baseURL+"/")

		// 构建正式路径: images/{resourceID}/preview_{i}.{ext}
		ext := filepath.Ext(tempPath)
		finalPath := fmt.Sprintf("images/%d/preview_%d%s", resourceID, i, ext)

		// 复制对象
		src := minio.CopySrcOptions{Bucket: bucket, Object: tempPath}
		dst := minio.CopyDestOptions{Bucket: bucket, Object: finalPath}

		_, err := s.client.CopyObject(ctx, dst, src)
		if err != nil {
			s.logger.Error("移动资源图片失败", "src", tempPath, "dst", finalPath, "error", err.Error())
			continue
		}

		// 删除临时文件
		_ = s.client.RemoveObject(ctx, bucket, tempPath, minio.RemoveObjectOptions{})

		// 构建最终URL
		finalURL := fmt.Sprintf("%s/%s", baseURL, finalPath)
		finalURLs = append(finalURLs, finalURL)

		s.logger.Info("成功移动资源图片", "from", tempPath, "to", finalPath)
	}

	return finalURLs, nil
}

// uploadFile 通用上传文件方法
func (s *ResourceStorageService) uploadFile(ctx context.Context, reader io.Reader, objectPath string, size int64, contentType string) (string, error) {
	bucket := s.cfg.ResourcesStorage.Bucket

	// 如果size未知，读入内存（使用对象池优化）
	if size < 0 {
		buf := utils.GetBuffer()
		defer utils.PutBuffer(buf)

		if _, err := io.Copy(buf, reader); err != nil {
			return "", err
		}
		size = int64(buf.Len())
		reader = bytes.NewReader(buf.Bytes())
	}

	opts := minio.PutObjectOptions{
		ContentType:  contentType,
		CacheControl: "no-cache, no-store, must-revalidate",
	}

	_, err := s.client.PutObject(ctx, bucket, objectPath, reader, size, opts)
	if err != nil {
		s.logger.Error("上传文件到资源桶失败", "bucket", bucket, "object", objectPath, "error", err.Error())
		return "", err
	}

	// 返回公共URL
	publicURL := fmt.Sprintf("%s/%s", s.cfg.ResourcesStorage.PublicBaseURL, objectPath)
	s.logger.Info("文件上传成功", "path", objectPath, "url", publicURL)

	return publicURL, nil
}

// GetObject 获取对象（支持Range请求）
// 返回io.ReadCloser以支持流式读取
func (s *ResourceStorageService) GetObject(ctx context.Context, objectPath string, opts minio.GetObjectOptions) (io.ReadCloser, error) {
	if s.client == nil {
		return nil, fmt.Errorf("资源存储客户端未初始化")
	}
	bucket := s.cfg.ResourcesStorage.Bucket
	obj, err := s.client.GetObject(ctx, bucket, objectPath, opts)
	if err != nil {
		s.logger.Error("获取资源对象失败", "bucket", bucket, "object", objectPath, "error", err.Error())
		return nil, err
	}
	return obj, nil
}

// StatObject 获取对象信息
func (s *ResourceStorageService) StatObject(ctx context.Context, objectPath string) (minio.ObjectInfo, error) {
	if s.client == nil {
		return minio.ObjectInfo{}, fmt.Errorf("资源存储客户端未初始化")
	}
	bucket := s.cfg.ResourcesStorage.Bucket
	info, err := s.client.StatObject(ctx, bucket, objectPath, minio.StatObjectOptions{})
	if err != nil {
		s.logger.Error("获取资源对象信息失败", "bucket", bucket, "object", objectPath, "error", err.Error())
		return minio.ObjectInfo{}, err
	}
	return info, nil
}

// StatObjectFromBucket 从指定桶获取对象信息
// 性能优化：使用空的StatObjectOptions避免每次分配
func (s *ResourceStorageService) StatObjectFromBucket(ctx context.Context, bucketName string, objectPath string) (minio.ObjectInfo, error) {
	if s.client == nil {
		return minio.ObjectInfo{}, fmt.Errorf("资源存储客户端未初始化")
	}
	info, err := s.client.StatObject(ctx, bucketName, objectPath, minio.StatObjectOptions{})
	if err != nil {
		// 性能优化：仅记录错误，避免记录对象路径（可能很长）
		s.logger.Debug("获取对象信息失败", "bucket", bucketName, "error", err.Error())
		return minio.ObjectInfo{}, err
	}
	return info, nil
}

// GetObjectFromBucket 从指定桶获取对象
// 性能优化：直接返回对象，避免不必要的日志记录
func (s *ResourceStorageService) GetObjectFromBucket(ctx context.Context, bucketName string, objectPath string, opts minio.GetObjectOptions) (io.ReadCloser, error) {
	if s.client == nil {
		return nil, fmt.Errorf("资源存储客户端未初始化")
	}
	// 直接返回，错误由调用方处理（避免双重日志）
	return s.client.GetObject(ctx, bucketName, objectPath, opts)
}

// ListObjectsRecursive 递归列举指定前缀下的所有对象
func (s *ResourceStorageService) ListObjectsRecursive(ctx context.Context, prefix string) ([]string, error) {
	if s.client == nil {
		return nil, fmt.Errorf("资源存储客户端未初始化")
	}
	bucket := s.cfg.ResourcesStorage.Bucket
	opts := minio.ListObjectsOptions{Prefix: prefix, Recursive: true}
	ch := s.client.ListObjects(ctx, bucket, opts)

	var objects []string
	for obj := range ch {
		if obj.Err != nil {
			s.logger.Error("列举对象失败", "bucket", bucket, "prefix", prefix, "error", obj.Err.Error())
			return nil, obj.Err
		}
		objects = append(objects, obj.Key)
	}
	return objects, nil
}

// RemoveObject 删除对象
func (s *ResourceStorageService) RemoveObject(ctx context.Context, objectPath string) error {
	if s.client == nil {
		return fmt.Errorf("资源存储客户端未初始化")
	}
	bucket := s.cfg.ResourcesStorage.Bucket
	err := s.client.RemoveObject(ctx, bucket, objectPath, minio.RemoveObjectOptions{})
	if err != nil {
		s.logger.Error("删除资源对象失败", "bucket", bucket, "object", objectPath, "error", err.Error())
		return err
	}
	return nil
}

// GetPublicBaseURL 返回资源桶的公共访问基地址
func (s *ResourceStorageService) GetPublicBaseURL() string {
	return s.cfg.ResourcesStorage.PublicBaseURL
}
