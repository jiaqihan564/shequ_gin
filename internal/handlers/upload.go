package handlers

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"gin/internal/models"
	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// UploadHandler 处理上传
type UploadHandler struct {
	storage            services.StorageClient
	userService        services.UserServiceInterface
	logger             utils.Logger
	maxAvatarSizeBytes int64
	maxAvatarHistory   int
}

// NewUploadHandler 创建上传处理器
func NewUploadHandler(storage services.StorageClient, userService services.UserServiceInterface, maxBytes int64, maxHistory int) *UploadHandler {
	return &UploadHandler{
		storage:            storage,
		userService:        userService,
		logger:             utils.GetLogger(),
		maxAvatarSizeBytes: maxBytes,
		maxAvatarHistory:   maxHistory,
	}
}

// UploadAvatar 上传头像
func (h *UploadHandler) UploadAvatar(c *gin.Context) {
	if h.storage == nil {
		utils.CodeErrorResponse(c, http.StatusServiceUnavailable, utils.ErrCodeUploadFailed, "存储服务不可用")
		return
	}

	// 验证认证并获取用户信息
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.CodeErrorResponse(c, http.StatusUnauthorized, utils.ErrCodeAuthRequired, "需要登录")
		return
	}

	usernameVal, _ := c.Get("username")
	username, _ := usernameVal.(string)
	if username == "" {
		utils.CodeErrorResponse(c, http.StatusInternalServerError, utils.ErrCodeUploadFailed, "用户信息错误")
		return
	}

	// 接收文件
	fileHeader, err := c.FormFile("file")
	if err != nil {
		utils.BadRequestResponse(c, "请选择文件")
		return
	}

	// 验证文件（大小+魔数）
	maxSize := h.maxAvatarSizeBytes
	if maxSize <= 0 {
		maxSize = 5 * 1024 * 1024
	}

	validator := utils.NewFileValidator(maxSize, []string{"image/png"})
	if err := validator.Validate(fileHeader); err != nil {
		statusCode := utils.GetHTTPStatusCode(err)
		if statusCode == 413 {
			c.Header("Connection", "close")
			utils.CodeErrorResponse(c, statusCode, utils.ErrCodeUploadTooLarge,
				fmt.Sprintf("文件过大，最大%dMB", maxSize/(1024*1024)))
		} else {
			utils.CodeErrorResponse(c, statusCode, utils.ErrCodeUploadInvalidType, "仅支持PNG格式")
		}
		return
	}

	// 打开文件
	file, err := fileHeader.Open()
	if err != nil {
		utils.CodeErrorResponse(c, http.StatusInternalServerError, utils.ErrCodeUploadFailed, "无法读取文件")
		return
	}
	defer file.Close()

	// 归档旧头像并上传新头像
	objectKey := fmt.Sprintf("%s/avatar.png", username)
	h.archiveOldAvatar(c.Request.Context(), userID, username, objectKey, time.Now().Unix())

	url, err := h.storage.PutObject(c.Request.Context(), objectKey, "image/png", file, fileHeader.Size)
	if err != nil {
		h.logger.Error("上传失败", "userID", userID, "error", err.Error())
		utils.CodeErrorResponse(c, http.StatusInternalServerError, utils.ErrCodeUploadFailed, "上传失败")
		return
	}

	// 更新数据库
	if h.userService != nil {
		if err := h.userService.UpdateUserAvatar(c.Request.Context(), &models.UserExtraProfile{
			UserID:    userID,
			AvatarURL: url,
		}); err != nil {
			h.logger.Warn("更新数据库头像URL失败", "userID", userID, "error", err.Error())
		}
	}

	// 返回成功响应
	urlWithTS := fmt.Sprintf("%s?t=%d", url, time.Now().Unix())
	h.logger.Info("上传成功", "userID", userID, "size", fileHeader.Size)

	utils.SuccessResponse(c, 200, "上传成功", gin.H{
		"url": urlWithTS,
	})

	// 异步清理历史头像
	go h.cleanupAvatarHistory(username)
}

// archiveOldAvatar 归档旧头像
func (h *UploadHandler) archiveOldAvatar(ctx context.Context, userID uint, username, objectKey string, timestamp int64) {
	exists, err := h.storage.ObjectExists(ctx, objectKey)
	if err != nil || !exists {
		return
	}

	archiveKey := fmt.Sprintf("%s/%d.png", username, timestamp)
	if err := h.storage.CopyObject(ctx, objectKey, archiveKey); err != nil {
		h.logger.Warn("归档失败", "userID", userID, "error", err.Error())
		return
	}

	if err := h.storage.RemoveObject(ctx, objectKey); err != nil {
		h.logger.Warn("删除旧头像失败", "userID", userID, "error", err.Error())
	}
}

// cleanupAvatarHistory 清理历史头像
func (h *UploadHandler) cleanupAvatarHistory(username string) {
	defer func() { _ = recover() }()

	if h.storage == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	objects, err := h.storage.ListObjects(ctx, fmt.Sprintf("%s/", username))
	if err != nil {
		return
	}

	// 过滤历史头像
	history := make([]services.ObjectInfo, 0, len(objects))
	for _, obj := range objects {
		base := path.Base(obj.Key)
		if !strings.EqualFold(base, "avatar.png") && strings.ToLower(path.Ext(base)) == ".png" {
			history = append(history, obj)
		}
	}

	if len(history) <= h.maxAvatarHistory {
		return
	}

	// 按时间戳排序
	sort.Slice(history, func(i, j int) bool {
		ti := h.parseTimestamp(history[i].Key, history[i].LastModified)
		tj := h.parseTimestamp(history[j].Key, history[j].LastModified)
		if ti == tj {
			return history[i].LastModified.After(history[j].LastModified)
		}
		return ti > tj
	})

	// 删除超出限制的历史
	for _, obj := range history[h.maxAvatarHistory:] {
		_ = h.storage.RemoveObject(ctx, obj.Key)
	}

	h.logger.Info("清理历史头像", "username", username, "deleted", len(history)-h.maxAvatarHistory)
}

// parseTimestamp 从文件名提取时间戳
func (h *UploadHandler) parseTimestamp(key string, fallback time.Time) int64 {
	base := path.Base(key)
	name := strings.TrimSuffix(base, path.Ext(base))
	if ts, err := strconv.ParseInt(name, 10, 64); err == nil {
		return ts
	}
	return fallback.Unix()
}

// ListAvatarHistory 获取历史头像列表
func (h *UploadHandler) ListAvatarHistory(c *gin.Context) {
	if h.storage == nil {
		utils.CodeErrorResponse(c, http.StatusServiceUnavailable, utils.ErrCodeUploadFailed, "服务不可用")
		return
	}

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
		utils.InternalServerErrorResponse(c, "获取失败")
		return
	}

	// 构建历史头像列表
	baseURL := h.storage.GetPublicBaseURL()
	items := make([]gin.H, 0)
	for _, obj := range objects {
		base := path.Base(obj.Key)
		if !strings.EqualFold(base, "avatar.png") && strings.ToLower(path.Ext(base)) == ".png" {
			items = append(items, gin.H{
				"key":           obj.Key,
				"url":           fmt.Sprintf("%s/%s", baseURL, obj.Key),
				"size":          obj.Size,
				"last_modified": obj.LastModified.Unix(),
			})
			if len(items) >= 50 {
				break
			}
		}
	}

	utils.SuccessResponse(c, 200, "成功", gin.H{"items": items, "total": len(items)})
}
