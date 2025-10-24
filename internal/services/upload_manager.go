package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"gin/internal/models"
	"gin/internal/utils"

	"github.com/minio/minio-go/v7"
)

// UploadManager 上传管理器（处理断点续传）
type UploadManager struct {
	db      *Database
	storage StorageClient
	logger  utils.Logger
}

// NewUploadManager 创建上传管理器
func NewUploadManager(db *Database, storage StorageClient) *UploadManager {
	return &UploadManager{
		db:      db,
		storage: storage,
		logger:  utils.GetLogger(),
	}
}

// InitUpload 初始化上传
func (m *UploadManager) InitUpload(ctx context.Context, userID uint, req models.InitUploadRequest) (*models.InitUploadResponse, error) {
	// 检查是否有未完成的上传
	var existing models.UploadChunk
	query := `SELECT id, upload_id, uploaded_chunks, chunk_size, status FROM upload_chunks WHERE upload_id = ? AND user_id = ?`
	err := m.db.DB.QueryRowContext(ctx, query, req.UploadID, userID).Scan(
		&existing.ID, &existing.UploadID, &existing.UploadedChunks, &existing.ChunkSize, &existing.Status,
	)

	chunkSize := 2 * 1024 * 1024 // 2MB

	if err == nil && existing.Status == 0 {
		// 有未完成的上传，返回进度
		var uploadedChunks []int
		if existing.UploadedChunks != "" {
			_ = json.Unmarshal([]byte(existing.UploadedChunks), &uploadedChunks)
		}

		m.logger.Info("恢复上传任务", "uploadID", req.UploadID, "uploadedChunks", len(uploadedChunks))
		return &models.InitUploadResponse{
			UploadID:       req.UploadID,
			UploadedChunks: uploadedChunks,
			ChunkSize:      existing.ChunkSize,
		}, nil
	}

	// 创建新的上传记录（如果重复则更新）
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour) // 24小时过期

	insertQuery := `INSERT INTO upload_chunks (upload_id, user_id, file_name, file_size, chunk_size, 
	                total_chunks, uploaded_chunks, status, expires_at, created_at, updated_at)
	                VALUES (?, ?, ?, ?, ?, ?, '[]', 0, ?, ?, ?)
	                ON DUPLICATE KEY UPDATE 
	                file_name = VALUES(file_name),
	                file_size = VALUES(file_size),
	                total_chunks = VALUES(total_chunks),
	                uploaded_chunks = '[]',
	                status = 0,
	                expires_at = VALUES(expires_at),
	                updated_at = VALUES(updated_at)`

	_, err = m.db.DB.ExecContext(ctx, insertQuery,
		req.UploadID, userID, req.FileName, req.FileSize, chunkSize,
		req.TotalChunks, expiresAt, now, now)

	if err != nil {
		m.logger.Error("创建上传记录失败", "error", err.Error())
		return nil, fmt.Errorf("创建上传记录失败")
	}

	m.logger.Info("初始化上传", "uploadID", req.UploadID, "fileName", req.FileName, "totalChunks", req.TotalChunks)
	return &models.InitUploadResponse{
		UploadID:       req.UploadID,
		UploadedChunks: []int{},
		ChunkSize:      chunkSize,
	}, nil
}

// UploadChunk 上传分片
func (m *UploadManager) UploadChunk(ctx context.Context, uploadID string, chunkIndex int, chunkData []byte) error {
	// 保存分片到MinIO
	objectKey := fmt.Sprintf("chunks/%s/chunk_%d", uploadID, chunkIndex)

	// 将[]byte转换为io.Reader
	reader := bytes.NewReader(chunkData)
	_, err := m.storage.PutObject(ctx, objectKey, "application/octet-stream", reader, int64(len(chunkData)))
	if err != nil {
		m.logger.Error("保存分片失败", "uploadID", uploadID, "chunkIndex", chunkIndex, "error", err.Error())
		return fmt.Errorf("保存分片失败")
	}

	// 更新上传记录
	var uploadedChunks []int
	query := `SELECT uploaded_chunks FROM upload_chunks WHERE upload_id = ?`
	var chunksJSON string
	err = m.db.DB.QueryRowContext(ctx, query, uploadID).Scan(&chunksJSON)
	if err != nil {
		return fmt.Errorf("查询上传记录失败")
	}

	if chunksJSON != "" {
		_ = json.Unmarshal([]byte(chunksJSON), &uploadedChunks)
	}

	// 添加当前分片索引
	found := false
	for _, idx := range uploadedChunks {
		if idx == chunkIndex {
			found = true
			break
		}
	}
	if !found {
		uploadedChunks = append(uploadedChunks, chunkIndex)
	}

	newChunksJSON, _ := json.Marshal(uploadedChunks)
	updateQuery := `UPDATE upload_chunks SET uploaded_chunks = ?, updated_at = ? WHERE upload_id = ?`
	_, err = m.db.DB.ExecContext(ctx, updateQuery, string(newChunksJSON), time.Now(), uploadID)

	m.logger.Info("分片上传成功", "uploadID", uploadID, "chunkIndex", chunkIndex, "progress", fmt.Sprintf("%d chunks", len(uploadedChunks)))
	return err
}

// MergeChunks 合并分片（真正实现文件合并）
func (m *UploadManager) MergeChunks(ctx context.Context, uploadID string) (*models.MergeChunksResponse, error) {
	// 获取上传记录
	var chunk models.UploadChunk
	query := `SELECT user_id, file_name, file_size, total_chunks, uploaded_chunks FROM upload_chunks WHERE upload_id = ?`
	var chunksJSON string
	err := m.db.DB.QueryRowContext(ctx, query, uploadID).Scan(
		&chunk.UserID, &chunk.FileName, &chunk.FileSize, &chunk.TotalChunks, &chunksJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("查询上传记录失败")
	}

	// 检查是否所有分片都已上传
	var uploadedChunks []int
	if chunksJSON != "" {
		_ = json.Unmarshal([]byte(chunksJSON), &uploadedChunks)
	}

	if len(uploadedChunks) != chunk.TotalChunks {
		return nil, fmt.Errorf("分片未全部上传，进度：%d/%d", len(uploadedChunks), chunk.TotalChunks)
	}

	// 生成最终存储路径
	now := time.Now()
	storagePath := fmt.Sprintf("resources/%d/%02d/%s_%s", now.Year(), now.Month(), uploadID[:8], chunk.FileName)

	m.logger.Info("开始合并分片", "uploadID", uploadID, "totalChunks", chunk.TotalChunks, "storagePath", storagePath)

	// 实际合并分片文件
	err = m.mergeChunksToStorage(ctx, uploadID, chunk.TotalChunks, storagePath, chunk.FileSize)
	if err != nil {
		m.logger.Error("合并分片失败", "uploadID", uploadID, "error", err.Error())
		return nil, fmt.Errorf("合并分片失败: %v", err)
	}

	// 更新状态为已完成
	updateQuery := `UPDATE upload_chunks SET status = 1, storage_path = ?, updated_at = ? WHERE upload_id = ?`
	_, err = m.db.DB.ExecContext(ctx, updateQuery, storagePath, time.Now(), uploadID)
	if err != nil {
		return nil, fmt.Errorf("更新上传状态失败")
	}

	// 异步清理临时分片文件
	go m.cleanupChunks(context.Background(), uploadID, chunk.TotalChunks)

	// 构建文件URL（使用公共访问URL）
	fileURL := fmt.Sprintf("%s/%s", m.storage.GetPublicBaseURL(), storagePath)

	m.logger.Info("合并分片成功", "uploadID", uploadID, "storagePath", storagePath, "fileURL", fileURL)
	return &models.MergeChunksResponse{
		StoragePath: fileURL, // 返回完整的公共URL
		FileURL:     fileURL,
	}, nil
}

// mergeChunksToStorage 将所有分片合并为一个完整文件并上传到MinIO
func (m *UploadManager) mergeChunksToStorage(ctx context.Context, uploadID string, totalChunks int, destPath string, fileSize int64) error {
	// 使用管道实现流式合并，避免大文件占用内存
	pr, pw := io.Pipe()

	// 在goroutine中写入所有分片数据到管道
	go func() {
		defer pw.Close()

		for i := 0; i < totalChunks; i++ {
			chunkPath := fmt.Sprintf("chunks/%s/chunk_%d", uploadID, i)

			// 获取分片对象
			object, err := m.storage.GetObject(ctx, chunkPath, minio.GetObjectOptions{})
			if err != nil {
				m.logger.Error("获取分片失败", "uploadID", uploadID, "chunkIndex", i, "error", err.Error())
				pw.CloseWithError(fmt.Errorf("获取分片%d失败: %w", i, err))
				return
			}

			// 流式复制分片数据到管道
			_, err = io.Copy(pw, object)
			object.Close()

			if err != nil {
				m.logger.Error("复制分片数据失败", "uploadID", uploadID, "chunkIndex", i, "error", err.Error())
				pw.CloseWithError(fmt.Errorf("复制分片%d数据失败: %w", i, err))
				return
			}

			m.logger.Debug("分片已合并", "uploadID", uploadID, "chunkIndex", i, "progress", fmt.Sprintf("%d/%d", i+1, totalChunks))
		}
	}()

	// 上传合并后的完整文件到MinIO
	_, err := m.storage.PutObject(ctx, destPath, "application/octet-stream", pr, fileSize)
	if err != nil {
		m.logger.Error("上传合并文件失败", "uploadID", uploadID, "destPath", destPath, "error", err.Error())
		return fmt.Errorf("上传合并文件失败: %w", err)
	}

	m.logger.Info("文件合并并上传成功", "uploadID", uploadID, "destPath", destPath, "size", fileSize)
	return nil
}

// cleanupChunks 清理临时分片文件
func (m *UploadManager) cleanupChunks(ctx context.Context, uploadID string, totalChunks int) {
	m.logger.Info("开始清理临时分片", "uploadID", uploadID, "totalChunks", totalChunks)

	deletedCount := 0
	for i := 0; i < totalChunks; i++ {
		chunkPath := fmt.Sprintf("chunks/%s/chunk_%d", uploadID, i)
		err := m.storage.RemoveObject(ctx, chunkPath)
		if err != nil {
			m.logger.Warn("删除临时分片失败", "uploadID", uploadID, "chunkIndex", i, "error", err.Error())
		} else {
			deletedCount++
		}
	}

	m.logger.Info("临时分片清理完成", "uploadID", uploadID, "deleted", deletedCount, "total", totalChunks)
}

// GetUploadStatus 查询上传进度
func (m *UploadManager) GetUploadStatus(ctx context.Context, uploadID string) (map[string]interface{}, error) {
	query := `SELECT total_chunks, uploaded_chunks, status FROM upload_chunks WHERE upload_id = ?`
	var totalChunks, status int
	var chunksJSON string
	err := m.db.DB.QueryRowContext(ctx, query, uploadID).Scan(&totalChunks, &chunksJSON, &status)
	if err != nil {
		return nil, fmt.Errorf("查询失败")
	}

	var uploadedChunks []int
	if chunksJSON != "" {
		_ = json.Unmarshal([]byte(chunksJSON), &uploadedChunks)
	}

	progress := float64(len(uploadedChunks)) / float64(totalChunks) * 100

	return map[string]interface{}{
		"upload_id":       uploadID,
		"total_chunks":    totalChunks,
		"uploaded_chunks": uploadedChunks,
		"progress":        progress,
		"status":          status,
	}, nil
}

// CancelUpload 取消上传
func (m *UploadManager) CancelUpload(ctx context.Context, uploadID string, userID uint) error {
	// 验证用户权限
	var ownerID uint
	err := m.db.DB.QueryRowContext(ctx, `SELECT user_id FROM upload_chunks WHERE upload_id = ?`, uploadID).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("上传任务不存在")
	}
	if ownerID != userID {
		return utils.ErrUnauthorized
	}

	// 更新状态为已取消
	_, err = m.db.DB.ExecContext(ctx, `UPDATE upload_chunks SET status = 2, updated_at = ? WHERE upload_id = ?`, time.Now(), uploadID)
	if err != nil {
		return fmt.Errorf("取消上传失败")
	}

	// 注：未来版本将清理MinIO中的临时分片文件

	m.logger.Info("取消上传", "uploadID", uploadID)
	return nil
}
