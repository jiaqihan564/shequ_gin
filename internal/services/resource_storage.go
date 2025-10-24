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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	// 设置桶策略为公开只读
	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "PublicReadGetObject",
				"Effect": "Allow",
				"Principal": "*",
				"Action": "s3:GetObject",
				"Resource": "arn:aws:s3:::%s/*"
			}
		]
	}`, bucketName)

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
	now := time.Now()
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

// GetPublicBaseURL 返回资源桶的公共访问基地址
func (s *ResourceStorageService) GetPublicBaseURL() string {
	return s.cfg.ResourcesStorage.PublicBaseURL
}
