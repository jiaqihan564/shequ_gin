package services

import (
	"context"
	"fmt"
	"io"
	"time"

	"gin/internal/config"
	"gin/internal/utils"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// BucketType 桶类型枚举
type BucketType string

const (
	BucketTypeUserAvatars      BucketType = "user-avatars"
	BucketTypeResourceChunks   BucketType = "resource-chunks"
	BucketTypeResourcePreviews BucketType = "resource-previews"
	BucketTypeDocumentImages   BucketType = "document-images"
	BucketTypeArticleImages    BucketType = "article-images"
	BucketTypeTempFiles        BucketType = "temp-files"
	BucketTypeSystemAssets     BucketType = "system-assets"
)

// MultiBucketStorage 多桶存储服务
type MultiBucketStorage struct {
	client  *minio.Client
	cfg     *config.Config
	logger  utils.Logger
	buckets map[BucketType]config.BucketConfig
}

// NewMultiBucketStorage 创建多桶存储服务
func NewMultiBucketStorage(cfg *config.Config) (*MultiBucketStorage, error) {
	logger := utils.GetLogger()

	// 初始化MinIO客户端
	client, err := minio.New(cfg.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKeyID, cfg.MinIO.SecretAccessKey, ""),
		Secure: cfg.MinIO.UseSSL,
	})
	if err != nil {
		logger.Error("初始化MinIO客户端失败", "error", err.Error())
		return nil, err
	}

	// 初始化桶配置映射
	buckets := map[BucketType]config.BucketConfig{
		BucketTypeUserAvatars:      cfg.BucketUserAvatars,
		BucketTypeResourceChunks:   cfg.BucketResourceChunks,
		BucketTypeResourcePreviews: cfg.BucketResourcePreviews,
		BucketTypeDocumentImages:   cfg.BucketDocumentImages,
		BucketTypeArticleImages:    cfg.BucketArticleImages,
		BucketTypeTempFiles:        cfg.BucketTempFiles,
		BucketTypeSystemAssets:     cfg.BucketSystemAssets,
	}

	storage := &MultiBucketStorage{
		client:  client,
		cfg:     cfg,
		logger:  logger,
		buckets: buckets,
	}

	// 初始化所有桶
	if err := storage.initializeBuckets(); err != nil {
		return nil, err
	}

	logger.Info("✅ 多桶存储服务初始化成功", "buckets", len(buckets))
	return storage, nil
}

// initializeBuckets 初始化所有桶（程序启动时自动执行）
func (s *MultiBucketStorage) initializeBuckets() error {
	s.logger.Info("🚀 开始自动初始化7个MinIO桶...")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.cfg.MinIO.OperationTimeout)*time.Second)
	defer cancel()

	createdCount := 0
	existingCount := 0

	for bucketType, bucketCfg := range s.buckets {
		bucketName := bucketCfg.Name

		// 检查桶是否存在
		exists, err := s.client.BucketExists(ctx, bucketName)
		if err != nil {
			s.logger.Error("检查桶失败", "bucket", bucketName, "error", err.Error())
			return fmt.Errorf("检查桶 %s 失败: %w", bucketName, err)
		}

		// 创建桶
		if !exists {
			if err := s.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
				s.logger.Error("创建桶失败", "bucket", bucketName, "error", err.Error())
				return fmt.Errorf("创建桶 %s 失败: %w", bucketName, err)
			}
			s.logger.Info("✅ 已创建桶", "bucket", bucketName, "type", bucketType, "url", bucketCfg.PublicBaseURL)
			createdCount++
		} else {
			s.logger.Debug("✓ 桶已存在", "bucket", bucketName)
			existingCount++
		}

		// 设置桶策略
		if err := s.setBucketPolicy(ctx, bucketName, bucketCfg); err != nil {
			s.logger.Warn("设置桶策略失败（不影响使用）", "bucket", bucketName, "error", err.Error())
			// 不中断初始化流程，策略可以后续手动设置
		}
	}

	s.logger.Info("🎉 MinIO桶初始化完成",
		"总数", len(s.buckets),
		"新创建", createdCount,
		"已存在", existingCount,
	)

	// 打印桶访问地址（方便调试）
	s.logger.Info("📦 桶访问地址：")
	for bucketType, bucketCfg := range s.buckets {
		publicStatus := "公开"
		if bucketCfg.PublicRead != nil && !*bucketCfg.PublicRead {
			publicStatus = "私有"
		}
		s.logger.Info("  → "+string(bucketType), "url", bucketCfg.PublicBaseURL, "status", publicStatus)
	}

	return nil
}

// setBucketPolicy 设置桶策略
func (s *MultiBucketStorage) setBucketPolicy(ctx context.Context, bucketName string, bucketCfg config.BucketConfig) error {
	// 判断是否公开读取（默认为true）
	publicRead := true
	if bucketCfg.PublicRead != nil {
		publicRead = *bucketCfg.PublicRead
	}

	if !publicRead {
		// 私有桶，移除所有公开策略
		if err := s.client.SetBucketPolicy(ctx, bucketName, ""); err != nil {
			return fmt.Errorf("设置私有策略失败: %w", err)
		}
		s.logger.Info("🔒 桶设置为私有", "bucket", bucketName)
		return nil
	}

	// 公开只读策略
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

	if err := s.client.SetBucketPolicy(ctx, bucketName, policy); err != nil {
		return fmt.Errorf("设置公开策略失败: %w", err)
	}

	s.logger.Info("🌐 桶设置为公开只读", "bucket", bucketName)
	return nil
}

// PutObject 上传文件到指定桶
func (s *MultiBucketStorage) PutObject(ctx context.Context, bucketType BucketType, objectPath string, contentType string, reader io.Reader, size int64) (string, error) {
	bucketCfg, ok := s.buckets[bucketType]
	if !ok {
		return "", fmt.Errorf("未知的桶类型: %s", bucketType)
	}

	opts := minio.PutObjectOptions{
		ContentType:  contentType,
		CacheControl: bucketCfg.CacheControl,
	}

	_, err := s.client.PutObject(ctx, bucketCfg.Name, objectPath, reader, size, opts)
	if err != nil {
		s.logger.Error("上传文件失败", "bucket", bucketCfg.Name, "object", objectPath, "error", err.Error())
		return "", err
	}

	// 返回公共URL
	publicURL := fmt.Sprintf("%s/%s", bucketCfg.PublicBaseURL, objectPath)
	s.logger.Info("文件上传成功", "bucket", bucketCfg.Name, "path", objectPath, "url", publicURL)

	return publicURL, nil
}

// GetObject 从指定桶获取对象
func (s *MultiBucketStorage) GetObject(ctx context.Context, bucketType BucketType, objectPath string, opts minio.GetObjectOptions) (io.ReadCloser, error) {
	bucketCfg, ok := s.buckets[bucketType]
	if !ok {
		return nil, fmt.Errorf("未知的桶类型: %s", bucketType)
	}

	obj, err := s.client.GetObject(ctx, bucketCfg.Name, objectPath, opts)
	if err != nil {
		s.logger.Error("获取对象失败", "bucket", bucketCfg.Name, "object", objectPath, "error", err.Error())
		return nil, err
	}

	return obj, nil
}

// ObjectExists 检查对象是否存在
func (s *MultiBucketStorage) ObjectExists(ctx context.Context, bucketType BucketType, objectPath string) (bool, error) {
	bucketCfg, ok := s.buckets[bucketType]
	if !ok {
		return false, fmt.Errorf("未知的桶类型: %s", bucketType)
	}

	_, err := s.client.StatObject(ctx, bucketCfg.Name, objectPath, minio.StatObjectOptions{})
	if err != nil {
		// 检查是否是对象不存在错误
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// RemoveObject 删除对象
func (s *MultiBucketStorage) RemoveObject(ctx context.Context, bucketType BucketType, objectPath string) error {
	bucketCfg, ok := s.buckets[bucketType]
	if !ok {
		return fmt.Errorf("未知的桶类型: %s", bucketType)
	}

	err := s.client.RemoveObject(ctx, bucketCfg.Name, objectPath, minio.RemoveObjectOptions{})
	if err != nil {
		s.logger.Error("删除对象失败", "bucket", bucketCfg.Name, "object", objectPath, "error", err.Error())
		return err
	}

	s.logger.Info("对象删除成功", "bucket", bucketCfg.Name, "object", objectPath)
	return nil
}

// CopyObject 复制对象
func (s *MultiBucketStorage) CopyObject(ctx context.Context, srcBucketType, dstBucketType BucketType, srcPath, dstPath string) error {
	srcBucketCfg, ok := s.buckets[srcBucketType]
	if !ok {
		return fmt.Errorf("未知的源桶类型: %s", srcBucketType)
	}

	dstBucketCfg, ok := s.buckets[dstBucketType]
	if !ok {
		return fmt.Errorf("未知的目标桶类型: %s", dstBucketType)
	}

	src := minio.CopySrcOptions{
		Bucket: srcBucketCfg.Name,
		Object: srcPath,
	}

	dst := minio.CopyDestOptions{
		Bucket: dstBucketCfg.Name,
		Object: dstPath,
	}

	_, err := s.client.CopyObject(ctx, dst, src)
	if err != nil {
		s.logger.Error("复制对象失败",
			"srcBucket", srcBucketCfg.Name,
			"dstBucket", dstBucketCfg.Name,
			"srcPath", srcPath,
			"dstPath", dstPath,
			"error", err.Error())
		return err
	}

	s.logger.Info("对象复制成功", "from", srcPath, "to", dstPath)
	return nil
}

// ListObjects 列举对象
func (s *MultiBucketStorage) ListObjects(ctx context.Context, bucketType BucketType, prefix string) ([]ObjectInfo, error) {
	bucketCfg, ok := s.buckets[bucketType]
	if !ok {
		return nil, fmt.Errorf("未知的桶类型: %s", bucketType)
	}

	objectCh := s.client.ListObjects(ctx, bucketCfg.Name, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	var objects []ObjectInfo
	for object := range objectCh {
		if object.Err != nil {
			s.logger.Error("列举对象失败", "bucket", bucketCfg.Name, "error", object.Err.Error())
			return nil, object.Err
		}
		objects = append(objects, ObjectInfo{
			Key:          object.Key,
			Size:         object.Size,
			LastModified: object.LastModified,
		})
	}

	return objects, nil
}

// GetPublicBaseURL 获取指定桶的公共基础URL
func (s *MultiBucketStorage) GetPublicBaseURL(bucketType BucketType) string {
	if bucketCfg, ok := s.buckets[bucketType]; ok {
		return bucketCfg.PublicBaseURL
	}
	return ""
}

// GetBucketName 获取桶名称
func (s *MultiBucketStorage) GetBucketName(bucketType BucketType) string {
	if bucketCfg, ok := s.buckets[bucketType]; ok {
		return bucketCfg.Name
	}
	return ""
}

// GetBucketConfig 获取桶配置
func (s *MultiBucketStorage) GetBucketConfig(bucketType BucketType) (config.BucketConfig, bool) {
	cfg, ok := s.buckets[bucketType]
	return cfg, ok
}
