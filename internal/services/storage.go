package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
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

	// 确保桶存在并设置公开访问权限
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

	// 设置桶策略为公开只读（允许匿名访问）
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
	}`, cfg.MinIO.Bucket)

	if err := cli.SetBucketPolicy(ctx, cfg.MinIO.Bucket, policy); err != nil {
		// 设置策略失败不阻塞服务启动，但要记录警告
		logger.Warn("设置桶公开访问策略失败",
			"bucket", cfg.MinIO.Bucket,
			"error", err.Error(),
			"suggestion", "请手动在MinIO控制台设置桶为Public访问")
	} else {
		logger.Info("桶策略已设置为公开只读", "bucket", cfg.MinIO.Bucket)
	}

	return &StorageService{client: cli, cfg: cfg, logger: logger}, nil
}

// PutObject 覆盖上传对象
func (s *StorageService) PutObject(ctx context.Context, objectPath string, contentType string, reader io.Reader, size int64) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("MinIO 客户端未初始化")
	}
	if size < 0 {
		// 使用对象池获取Buffer（减少内存分配）
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
	if s.client == nil {
		return false, fmt.Errorf("MinIO 客户端未初始化")
	}
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
	if s.client == nil {
		return fmt.Errorf("MinIO 客户端未初始化")
	}
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
	if s.client == nil {
		return fmt.Errorf("MinIO 客户端未初始化")
	}
	err := s.client.RemoveObject(ctx, s.cfg.MinIO.Bucket, objectPath, minio.RemoveObjectOptions{})
	if err != nil {
		s.logger.Error("删除对象失败", "object", objectPath, "error", err.Error())
		return err
	}
	return nil
}

// ListObjects 列举指定前缀下的对象（非递归）
func (s *StorageService) ListObjects(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	if s.client == nil {
		return nil, fmt.Errorf("MinIO 客户端未初始化")
	}
	ops := minio.ListObjectsOptions{Prefix: prefix, Recursive: false}
	ch := s.client.ListObjects(ctx, s.cfg.MinIO.Bucket, ops)
	var list []ObjectInfo
	for obj := range ch {
		if obj.Err != nil {
			s.logger.Error("列举对象失败", "prefix", prefix, "error", obj.Err.Error())
			return nil, obj.Err
		}
		if len(obj.Key) > 0 && obj.Key[len(obj.Key)-1] == '/' {
			continue
		}
		list = append(list, ObjectInfo{Key: obj.Key, Size: obj.Size, LastModified: obj.LastModified})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].LastModified.After(list[j].LastModified) })
	return list, nil
}

// GetObject 获取对象（支持Range请求）
func (s *StorageService) GetObject(ctx context.Context, objectPath string, opts interface{}) (io.ReadCloser, error) {
	if s.client == nil {
		return nil, fmt.Errorf("MinIO 客户端未初始化")
	}
	// 类型断言获取minio选项
	var getOpts minio.GetObjectOptions
	if opts != nil {
		if minioOpts, ok := opts.(minio.GetObjectOptions); ok {
			getOpts = minioOpts
		}
	}
	obj, err := s.client.GetObject(ctx, s.cfg.MinIO.Bucket, objectPath, getOpts)
	if err != nil {
		s.logger.Error("获取对象失败", "object", objectPath, "error", err.Error())
		return nil, err
	}
	return obj, nil
}

// StatObject 获取对象信息
func (s *StorageService) StatObject(ctx context.Context, objectPath string) (minio.ObjectInfo, error) {
	if s.client == nil {
		return minio.ObjectInfo{}, fmt.Errorf("MinIO 客户端未初始化")
	}
	info, err := s.client.StatObject(ctx, s.cfg.MinIO.Bucket, objectPath, minio.StatObjectOptions{})
	if err != nil {
		s.logger.Error("获取对象信息失败", "object", objectPath, "error", err.Error())
		return minio.ObjectInfo{}, err
	}
	return info, nil
}

// GetPublicBaseURL 返回公共访问基地址
func (s *StorageService) GetPublicBaseURL() string {
	return s.cfg.Assets.PublicBaseURL
}
