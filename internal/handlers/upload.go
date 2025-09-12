package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// UploadHandler 处理上传
type UploadHandler struct {
	storage            services.StorageClient
	logger             utils.Logger
	maxAvatarSizeBytes int64
}

// NewUploadHandler 创建上传处理器
func NewUploadHandler(storage services.StorageClient, maxAvatarSizeBytes int64) *UploadHandler {
	return &UploadHandler{storage: storage, logger: utils.GetLogger(), maxAvatarSizeBytes: maxAvatarSizeBytes}
}

// UploadAvatar 上传头像：仅允许 PNG，并以时间戳覆盖当前头像
// 表单字段：file (multipart/form-data)
func (h *UploadHandler) UploadAvatar(c *gin.Context) {
	if h.storage == nil {
		utils.CodeErrorResponse(c, http.StatusServiceUnavailable, utils.ErrCodeUploadFailed, "服务不可用")
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		h.logger.Warn("上传头像失败：未认证", "ip", c.ClientIP())
		utils.CodeErrorResponse(c, http.StatusUnauthorized, utils.ErrCodeAuthRequired, "未认证")
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
		utils.CodeErrorResponse(c, http.StatusRequestEntityTooLarge, utils.ErrCodeUploadTooLarge, "文件过大")
		return
	}

	// 只允许 PNG（扩展名校验）
	if !isPNG(fileHeader.Filename) {
		h.logger.Warn("上传头像失败：类型不支持", "userID", userID, "filename", fileHeader.Filename)
		utils.CodeErrorResponse(c, http.StatusBadRequest, utils.ErrCodeUploadInvalidType, "类型不支持")
		return
	}

	// 魔数校验（PNG signature: 89 50 4E 47 0D 0A 1A 0A）
	probe, err := c.FormFile("file")
	if err != nil {
		h.logger.Error("读取上传文件失败(探测)", "userID", userID, "error", err.Error())
		utils.CodeErrorResponse(c, http.StatusInternalServerError, utils.ErrCodeUploadFailed, "服务器错误")
		return
	}
	pr, err := probe.Open()
	if err != nil {
		h.logger.Error("打开上传文件失败(探测)", "userID", userID, "error", err.Error())
		utils.CodeErrorResponse(c, http.StatusInternalServerError, utils.ErrCodeUploadFailed, "服务器错误")
		return
	}
	buf := make([]byte, 8)
	n, _ := io.ReadFull(pr, buf)
	_ = pr.Close()
	if n != 8 || !(buf[0] == 0x89 && buf[1] == 0x50 && buf[2] == 0x4E && buf[3] == 0x47 && buf[4] == 0x0D && buf[5] == 0x0A && buf[6] == 0x1A && buf[7] == 0x0A) {
		h.logger.Warn("上传头像失败：PNG魔数不匹配", "userID", userID)
		utils.CodeErrorResponse(c, http.StatusBadRequest, utils.ErrCodeUploadInvalidType, "PNG签名不匹配")
		return
	}

	// 重新打开文件用于上传
	file, err := fileHeader.Open()
	if err != nil {
		h.logger.Error("打开上传文件失败", "userID", userID, "error", err.Error())
		utils.CodeErrorResponse(c, http.StatusInternalServerError, utils.ErrCodeUploadFailed, "服务器错误")
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
		utils.CodeErrorResponse(c, http.StatusInternalServerError, utils.ErrCodeUploadFailed, "服务器错误")
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

	// 异步清理历史头像，仅保留最新9个（{username}/{timestamp}.png）
	go func(username string) {
		defer func() { _ = recover() }()
		if h.storage == nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		objects, err := h.storage.ListObjects(ctx, fmt.Sprintf("%s/", username))
		if err != nil {
			h.logger.Warn("列举历史头像失败", "user", username, "error", err.Error())
			return
		}

		// 仅保留历史 PNG（排除 avatar.png）
		history := make([]services.ObjectInfo, 0, len(objects))
		for _, obj := range objects {
			base := path.Base(obj.Key)
			if strings.EqualFold(base, "avatar.png") {
				continue
			}
			if strings.ToLower(path.Ext(base)) != ".png" {
				continue
			}
			history = append(history, obj)
		}
		if len(history) <= 9 {
			return
		}

		// 优先按文件名中的时间戳降序排序；解析失败则退化为按 LastModified
		parseTs := func(key string, fallback time.Time) int64 {
			base := path.Base(key)
			name := strings.TrimSuffix(base, path.Ext(base))
			if v, err := strconv.ParseInt(name, 10, 64); err == nil {
				return v
			}
			return fallback.Unix()
		}
		sort.Slice(history, func(i, j int) bool {
			ti := parseTs(history[i].Key, history[i].LastModified)
			tj := parseTs(history[j].Key, history[j].LastModified)
			if ti == tj {
				return history[i].LastModified.After(history[j].LastModified)
			}
			return ti > tj
		})

		// 删除第 10 个之后的历史（仅保留 9 个）
		for _, obj := range history[9:] {
			key := obj.Key
			if err := h.storage.RemoveObject(ctx, key); err != nil {
				h.logger.Warn("删除历史头像失败", "key", key, "error", err.Error())
			}
		}
	}(username)
}

func isPNG(filename string) bool {
	name := strings.ToLower(filename)
	ext := strings.ToLower(path.Ext(name))
	return ext == ".png"
}

// ListAvatarHistory 获取历史头像列表（按时间倒序，最多返回50条）
func (h *UploadHandler) ListAvatarHistory(c *gin.Context) {
	if h.storage == nil {
		utils.CodeErrorResponse(c, http.StatusServiceUnavailable, utils.ErrCodeUploadFailed, "服务不可用")
		return
	}

	// 需要认证
	_, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.CodeErrorResponse(c, http.StatusUnauthorized, utils.ErrCodeAuthRequired, "未认证")
		return
	}
	usernameVal, _ := c.Get("username")
	username, _ := usernameVal.(string)
	if username == "" {
		utils.BadRequestResponse(c, "缺少用户名")
		return
	}

	objects, err := h.storage.ListObjects(c.Request.Context(), fmt.Sprintf("%s/", username))
	if err != nil {
		utils.InternalServerErrorResponse(c, "列举历史头像失败")
		return
	}
	baseURL := h.storage.GetPublicBaseURL()
	items := make([]gin.H, 0, len(objects))
	count := 0
	for _, obj := range objects {
		base := path.Base(obj.Key)
		if strings.EqualFold(base, "avatar.png") || strings.ToLower(path.Ext(base)) != ".png" {
			continue
		}
		url := fmt.Sprintf("%s/%s", baseURL, obj.Key)
		items = append(items, gin.H{
			"key":           obj.Key,
			"url":           url,
			"size":          obj.Size,
			"last_modified": obj.LastModified.Unix(),
		})
		count++
		if count >= 50 {
			break
		}
	}

	utils.SuccessResponse(c, 200, "OK", gin.H{"items": items})
}
