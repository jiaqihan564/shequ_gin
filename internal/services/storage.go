package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"gin/internal/config"
	"gin/internal/utils"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// StorageService 封装对象存储（MinIO）
type StorageService struct {
	client *minio.Client
	cfg    *config.Config
	logger utils.Logger
}

// NewStorageService 初始化存储服务
func NewStorageService(cfg *config.Config) (*StorageService, error) {
	logger := utils.GetLogger()
	cli, err := minio.New(cfg.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKeyID, cfg.MinIO.SecretAccessKey, ""),
		Secure: cfg.MinIO.UseSSL,
	})
	if err != nil {
		logger.Error("初始化 MinIO 客户端失败", "error", err.Error())
		return nil, err
	}

	// 确保桶存在
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	exists, err := cli.BucketExists(ctx, cfg.MinIO.Bucket)
	if err != nil {
		logger.Error("检查桶失败", "bucket", cfg.MinIO.Bucket, "error", err.Error())
		return nil, err
	}
	if !exists {
		if err := cli.MakeBucket(ctx, cfg.MinIO.Bucket, minio.MakeBucketOptions{}); err != nil {
			logger.Error("创建桶失败", "bucket", cfg.MinIO.Bucket, "error", err.Error())
			return nil, err
		}
		logger.Info("已创建桶", "bucket", cfg.MinIO.Bucket)
	}

	return &StorageService{client: cli, cfg: cfg, logger: logger}, nil
}

// PutObject 覆盖上传对象
func (s *StorageService) PutObject(ctx context.Context, objectPath string, contentType string, reader io.Reader, size int64) (string, error) {
	if size < 0 {
		// 将未知大小的 Reader 读入内存缓冲（头像通常较小）
		buf := new(bytes.Buffer)
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
	info, err := s.client.PutObject(ctx, s.cfg.MinIO.Bucket, objectPath, reader, size, opts)
	if err != nil {
		s.logger.Error("上传对象失败", "bucket", s.cfg.MinIO.Bucket, "object", objectPath, "error", err.Error())
		return "", err
	}
	_ = info

	publicBase := s.cfg.Assets.PublicBaseURL
	if publicBase == "" {
		return "", fmt.Errorf("未配置 Assets.PublicBaseURL")
	}
	return fmt.Sprintf("%s/%s", publicBase, objectPath), nil
}

// ObjectExists 判断对象是否存在
func (s *StorageService) ObjectExists(ctx context.Context, objectPath string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.cfg.MinIO.Bucket, objectPath, minio.StatObjectOptions{})
	if err != nil {
		// 将错误转为通用响应，判断是否为不存在
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" || errResp.Code == "NotFound" || errResp.StatusCode == 404 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CopyObject 在桶内复制对象
func (s *StorageService) CopyObject(ctx context.Context, srcPath, dstPath string) error {
	src := minio.CopySrcOptions{Bucket: s.cfg.MinIO.Bucket, Object: srcPath}
	dst := minio.CopyDestOptions{Bucket: s.cfg.MinIO.Bucket, Object: dstPath}
	_, err := s.client.CopyObject(ctx, dst, src)
	if err != nil {
		s.logger.Error("复制对象失败", "src", srcPath, "dst", dstPath, "error", err.Error())
		return err
	}
	return nil
}

// RemoveObject 删除对象
func (s *StorageService) RemoveObject(ctx context.Context, objectPath string) error {
	err := s.client.RemoveObject(ctx, s.cfg.MinIO.Bucket, objectPath, minio.RemoveObjectOptions{})
	if err != nil {
		s.logger.Error("删除对象失败", "object", objectPath, "error", err.Error())
		return err
	}
	return nil
}
