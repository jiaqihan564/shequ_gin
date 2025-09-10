package handlers

import (
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// UploadHandler 处理上传
type UploadHandler struct {
	storage            *services.StorageService
	logger             utils.Logger
	maxAvatarSizeBytes int64
}

// NewUploadHandler 创建上传处理器
func NewUploadHandler(storage *services.StorageService, maxAvatarSizeBytes int64) *UploadHandler {
	return &UploadHandler{storage: storage, logger: utils.GetLogger(), maxAvatarSizeBytes: maxAvatarSizeBytes}
}

// UploadAvatar 上传头像：仅允许 PNG，并以时间戳覆盖当前头像
// 表单字段：file (multipart/form-data)
func (h *UploadHandler) UploadAvatar(c *gin.Context) {
	if h.storage == nil {
		utils.ErrorResponse(c, http.StatusServiceUnavailable, utils.ErrCodeUploadFailed)
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		h.logger.Warn("上传头像失败：未认证", "ip", c.ClientIP())
		utils.UnauthorizedResponse(c, utils.ErrCodeAuthRequired)
		return
	}

	usernameVal, _ := c.Get("username")
	username, _ := usernameVal.(string)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		h.logger.Warn("上传头像失败：缺少文件", "userID", userID, "error", err.Error())
		utils.BadRequestResponse(c, "缺少文件")
		return
	}

	// 限制大小：使用配置上限（默认5MB）
	maxSize := h.maxAvatarSizeBytes
	if maxSize <= 0 {
		maxSize = 5 * 1024 * 1024
	}
	if fileHeader.Size > maxSize {
		h.logger.Warn("上传头像失败：文件过大", "userID", userID, "size", fileHeader.Size)
		c.Header("Connection", "close")
		utils.ErrorResponse(c, http.StatusRequestEntityTooLarge, utils.ErrCodeUploadTooLarge)
		return
	}

	// 只允许 PNG（扩展名校验）
	if !isPNG(fileHeader.Filename) {
		h.logger.Warn("上传头像失败：类型不支持", "userID", userID, "filename", fileHeader.Filename)
		utils.BadRequestResponse(c, utils.ErrCodeUploadInvalidType)
		return
	}

	// 魔数校验（PNG signature: 89 50 4E 47 0D 0A 1A 0A）
	probe, err := c.FormFile("file")
	if err != nil {
		h.logger.Error("读取上传文件失败(探测)", "userID", userID, "error", err.Error())
		utils.InternalServerErrorResponse(c, utils.ErrCodeUploadFailed)
		return
	}
	pr, err := probe.Open()
	if err != nil {
		h.logger.Error("打开上传文件失败(探测)", "userID", userID, "error", err.Error())
		utils.InternalServerErrorResponse(c, utils.ErrCodeUploadFailed)
		return
	}
	buf := make([]byte, 8)
	n, _ := io.ReadFull(pr, buf)
	_ = pr.Close()
	if n != 8 || !(buf[0] == 0x89 && buf[1] == 0x50 && buf[2] == 0x4E && buf[3] == 0x47 && buf[4] == 0x0D && buf[5] == 0x0A && buf[6] == 0x1A && buf[7] == 0x0A) {
		h.logger.Warn("上传头像失败：PNG魔数不匹配", "userID", userID)
		utils.BadRequestResponse(c, utils.ErrCodeUploadInvalidType)
		return
	}

	// 重新打开文件用于上传
	file, err := fileHeader.Open()
	if err != nil {
		h.logger.Error("打开上传文件失败", "userID", userID, "error", err.Error())
		utils.InternalServerErrorResponse(c, utils.ErrCodeUploadFailed)
		return
	}
	defer file.Close()

	// 固定路径 {username}/avatar.png，返回 URL 时追加时间戳参数用于刷新缓存
	timestamp := time.Now().Unix()
	objectKey := fmt.Sprintf("%s/avatar.png", username)

	// 若已有旧头像，先归档为 {username}/{timestamp}.png，再删除旧的 avatar.png
	if exists, err := h.storage.ObjectExists(c.Request.Context(), objectKey); err == nil && exists {
		archiveKey := fmt.Sprintf("%s/%d.png", username, timestamp)
		if err := h.storage.CopyObject(c.Request.Context(), objectKey, archiveKey); err != nil {
			// 归档失败不阻塞上传，仅记录日志
			h.logger.Warn("归档旧头像失败", "userID", userID, "from", objectKey, "to", archiveKey, "error", err.Error())
		} else {
			if err := h.storage.RemoveObject(c.Request.Context(), objectKey); err != nil {
				h.logger.Warn("删除旧头像失败", "userID", userID, "object", objectKey, "error", err.Error())
			}
		}
	}

	url, err := h.storage.PutObject(c.Request.Context(), objectKey, "image/png", file, fileHeader.Size)
	if err != nil {
		h.logger.Error("上传到对象存储失败", "userID", userID, "error", err.Error())
		utils.InternalServerErrorResponse(c, utils.ErrCodeUploadFailed)
		return
	}

	urlWithTS := fmt.Sprintf("%s?t=%d", url, time.Now().Unix())
	h.logger.Info("上传头像成功", "userID", userID, "url", urlWithTS)
	utils.SuccessResponse(c, 200, "OK", gin.H{
		"url":    urlWithTS,
		"width":  0,
		"height": 0,
		"mime":   "image/png",
		"size":   fileHeader.Size,
	})
}

func isPNG(filename string) bool {
	name := strings.ToLower(filename)
	ext := strings.ToLower(path.Ext(name))
	return ext == ".png"
}
