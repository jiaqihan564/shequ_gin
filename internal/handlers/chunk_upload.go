package handlers

import (
	"fmt"
	"io"

	"gin/internal/models"
	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// ChunkUploadHandler 分片上传处理器
type ChunkUploadHandler struct {
	uploadMgr *services.UploadManager
	logger    utils.Logger
}

// NewChunkUploadHandler 创建分片上传处理器
func NewChunkUploadHandler(uploadMgr *services.UploadManager) *ChunkUploadHandler {
	return &ChunkUploadHandler{
		uploadMgr: uploadMgr,
		logger:    utils.GetLogger(),
	}
}

// InitUpload 初始化上传
func (h *ChunkUploadHandler) InitUpload(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	var req models.InitUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, "请求参数错误")
		return
	}

	ctx := c.Request.Context()
	response, err := h.uploadMgr.InitUpload(ctx, userID, req)
	if err != nil {
		h.logger.Error("初始化上传失败", "error", err.Error())
		utils.ErrorResponse(c, 500, "初始化上传失败")
		return
	}

	utils.SuccessResponse(c, 200, "初始化成功", response)
}

// UploadChunk 上传分片
func (h *ChunkUploadHandler) UploadChunk(c *gin.Context) {
	_, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	uploadID := c.PostForm("upload_id")
	chunkIndexStr := c.PostForm("chunk_index")

	if uploadID == "" || chunkIndexStr == "" {
		utils.ValidationErrorResponse(c, "缺少必要参数")
		return
	}

	chunkIndex := 0
	if _, err := fmt.Sscanf(chunkIndexStr, "%d", &chunkIndex); err != nil {
		utils.ValidationErrorResponse(c, "分片索引无效")
		return
	}

	// 读取分片数据
	file, err := c.FormFile("chunk")
	if err != nil {
		utils.ValidationErrorResponse(c, "分片文件读取失败")
		return
	}

	src, err := file.Open()
	if err != nil {
		utils.ErrorResponse(c, 500, "打开文件失败")
		return
	}
	defer src.Close()

	chunkData, err := io.ReadAll(src)
	if err != nil {
		utils.ErrorResponse(c, 500, "读取文件失败")
		return
	}

	// 上传分片
	ctx := c.Request.Context()
	err = h.uploadMgr.UploadChunk(ctx, uploadID, chunkIndex, chunkData)
	if err != nil {
		h.logger.Error("上传分片失败", "uploadID", uploadID, "chunkIndex", chunkIndex, "error", err.Error())
		utils.ErrorResponse(c, 500, "上传分片失败")
		return
	}

	utils.SuccessResponse(c, 200, "分片上传成功", nil)
}

// MergeChunks 合并分片
func (h *ChunkUploadHandler) MergeChunks(c *gin.Context) {
	_, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	var req models.MergeChunksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationErrorResponse(c, "请求参数错误")
		return
	}

	ctx := c.Request.Context()
	response, err := h.uploadMgr.MergeChunks(ctx, req.UploadID)
	if err != nil {
		h.logger.Error("合并分片失败", "uploadID", req.UploadID, "error", err.Error())
		utils.ErrorResponse(c, 500, err.Error())
		return
	}

	utils.SuccessResponse(c, 200, "合并成功", response)
}

// GetUploadStatus 查询上传进度
func (h *ChunkUploadHandler) GetUploadStatus(c *gin.Context) {
	uploadID := c.Param("upload_id")

	ctx := c.Request.Context()
	status, err := h.uploadMgr.GetUploadStatus(ctx, uploadID)
	if err != nil {
		utils.ErrorResponse(c, 404, "上传任务不存在")
		return
	}

	utils.SuccessResponse(c, 200, "获取成功", status)
}

// CancelUpload 取消上传
func (h *ChunkUploadHandler) CancelUpload(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	uploadID := c.Param("upload_id")

	ctx := c.Request.Context()
	err = h.uploadMgr.CancelUpload(ctx, uploadID, userID)
	if err != nil {
		utils.ErrorResponse(c, 500, err.Error())
		return
	}

	utils.SuccessResponse(c, 200, "取消成功", nil)
}
