package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"gin/internal/utils"
)

// ResourceImageService 资源图片服务（7桶架构）
type ResourceImageService struct {
	multiBucket *MultiBucketStorage
	logger      utils.Logger
}

// NewResourceImageService 创建资源图片服务
func NewResourceImageService(multiBucket *MultiBucketStorage) *ResourceImageService {
	return &ResourceImageService{
		multiBucket: multiBucket,
		logger:      utils.GetLogger(),
	}
}

// MovePreviewImagesToFormal 将临时预览图移动到正式目录
func (s *ResourceImageService) MovePreviewImagesToFormal(ctx context.Context, tempURLs []string, resourceID uint) ([]string, error) {
	if s.multiBucket == nil {
		return nil, fmt.Errorf("多桶存储服务未初始化")
	}

	var finalURLs []string
	tempBucket := BucketTypeTempFiles
	formalBucket := BucketTypeResourcePreviews
	baseURL := s.multiBucket.GetPublicBaseURL(formalBucket)

	for i, tempURL := range tempURLs {
		// 提取临时路径（去掉base URL）
		tempBaseURL := s.multiBucket.GetPublicBaseURL(tempBucket)
		tempPath := strings.TrimPrefix(tempURL, tempBaseURL+"/")

		// 构建正式路径: {resourceID}/preview_{i}.png
		ext := filepath.Ext(tempPath)
		if ext == "" {
			ext = ".png"
		}
		finalPath := fmt.Sprintf("%d/preview_%d%s", resourceID, i, ext)

		// 复制对象（从temp-files到resource-previews）
		err := s.multiBucket.CopyObject(ctx, tempBucket, formalBucket, tempPath, finalPath)
		if err != nil {
			s.logger.Error("移动预览图失败", "src", tempPath, "dst", finalPath, "error", err.Error())
			continue
		}

		// 删除临时文件
		_ = s.multiBucket.RemoveObject(ctx, tempBucket, tempPath)

		// 构建最终URL
		finalURL := fmt.Sprintf("%s/%s", baseURL, finalPath)
		finalURLs = append(finalURLs, finalURL)

		s.logger.Info("成功移动预览图", "from", tempPath, "to", finalPath)
	}

	return finalURLs, nil
}

// DeleteResourceImages 删除资源的所有预览图
func (s *ResourceImageService) DeleteResourceImages(ctx context.Context, resourceID uint) error {
	if s.multiBucket == nil {
		return fmt.Errorf("多桶存储服务未初始化")
	}

	prefix := fmt.Sprintf("%d/", resourceID)
	
	// 列举所有图片
	objects, err := s.multiBucket.ListObjects(ctx, BucketTypeResourcePreviews, prefix)
	if err != nil {
		s.logger.Error("列举资源图片失败", "resourceID", resourceID, "error", err.Error())
		return err
	}

	// 删除所有图片
	deletedCount := 0
	for _, obj := range objects {
		if err := s.multiBucket.RemoveObject(ctx, BucketTypeResourcePreviews, obj.Key); err != nil {
			s.logger.Warn("删除图片失败", "key", obj.Key, "error", err.Error())
		} else {
			deletedCount++
		}
	}

	s.logger.Info("删除资源图片完成", "resourceID", resourceID, "deleted", deletedCount)
	return nil
}

