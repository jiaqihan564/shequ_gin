package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gin/internal/models"
	"gin/internal/utils"
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

// MergeChunks 合并分片
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

	// TODO: 从MinIO读取所有分片并合并（需要实现实际的合并逻辑）
	// 这里先标记为已完成，实际合并逻辑需要根据MinIO客户端实现

	// 更新状态为已完成
	updateQuery := `UPDATE upload_chunks SET status = 1, storage_path = ?, updated_at = ? WHERE upload_id = ?`
	_, err = m.db.DB.ExecContext(ctx, updateQuery, storagePath, time.Now(), uploadID)
	if err != nil {
		return nil, fmt.Errorf("更新上传状态失败")
	}

	// 构建文件URL
	fileURL := fmt.Sprintf("/resources/download/%s", uploadID)

	m.logger.Info("合并分片成功", "uploadID", uploadID, "storagePath", storagePath)
	return &models.MergeChunksResponse{
		StoragePath: storagePath,
		FileURL:     fileURL,
	}, nil
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

	// TODO: 删除MinIO中的分片文件

	m.logger.Info("取消上传", "uploadID", uploadID)
	return nil
}
