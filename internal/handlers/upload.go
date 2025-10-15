package handlers

import (
	"context"
	"fmt"
	"mime/multipart"
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
func NewUploadHandler(storage services.StorageClient, userService services.UserServiceInterface, maxAvatarSizeBytes int64, maxAvatarHistory int) *UploadHandler {
	return &UploadHandler{
		storage:            storage,
		userService:        userService,
		logger:             utils.GetLogger(),
		maxAvatarSizeBytes: maxAvatarSizeBytes,
		maxAvatarHistory:   maxAvatarHistory,
	}
}

// UploadAvatar 上传头像到对象存储
//
// 功能说明：
//   - 接收用户上传的PNG图片文件
//   - 验证文件大小和格式（使用魔数验证）
//   - 将旧头像归档为历史版本（{username}/{timestamp}.png）
//   - 保存新头像为固定文件名（{username}/avatar.png）
//   - 异步清理超出数量限制的历史头像
//
// 请求：POST /api/files/upload 或 /api/upload
// Content-Type: multipart/form-data
// 表单字段：file (PNG图片)
//
// 响应：
//   - 200: 上传成功，返回头像URL（带时间戳防缓存）
//   - 400: 文件类型不支持
//   - 401: 未认证
//   - 413: 文件过大
//   - 503: 存储服务不可用
func (h *UploadHandler) UploadAvatar(c *gin.Context) {
	// 1. 检查存储服务状态
	if h.storage == nil {
		h.logger.Error("存储服务未初始化")
		utils.CodeErrorResponse(c, http.StatusServiceUnavailable, utils.ErrCodeUploadFailed, "存储服务不可用")
		return
	}

	// 2. 验证用户认证
	userID, username, err := h.getUserInfo(c)
	if err != nil {
		return // 错误已在函数内处理
	}

	// 3. 接收并验证文件
	fileHeader, err := h.receiveAndValidateFile(c, userID)
	if err != nil {
		return // 错误已在函数内处理
	}

	// 4. 打开文件准备上传
	file, err := fileHeader.Open()
	if err != nil {
		h.logger.Error("打开上传文件失败", "userID", userID, "error", err.Error())
		utils.CodeErrorResponse(c, http.StatusInternalServerError, utils.ErrCodeUploadFailed, "无法读取文件")
		return
	}
	defer file.Close()

	// 5. 归档旧头像（不阻塞上传流程）
	timestamp := time.Now().Unix()
	objectKey := fmt.Sprintf("%s/avatar.png", username)
	h.archiveOldAvatar(c.Request.Context(), userID, username, objectKey, timestamp)

	// 6. 上传新头像（PNG格式）
	url, err := h.storage.PutObject(c.Request.Context(), objectKey, "image/png", file, fileHeader.Size)
	if err != nil {
		h.logger.Error("上传到对象存储失败", "userID", userID, "error", err.Error())
		utils.CodeErrorResponse(c, http.StatusInternalServerError, utils.ErrCodeUploadFailed, "上传失败")
		return
	}

	// 7. 更新数据库中的头像URL（不带时间戳，因为文件名固定）
	if h.userService != nil {
		prof := &models.UserExtraProfile{
			UserID:    userID,
			AvatarURL: url, // 使用不带时间戳的URL存储到数据库
		}
		if err := h.userService.UpdateUserAvatar(c.Request.Context(), prof); err != nil {
			// 数据库更新失败不影响上传结果，仅记录日志
			h.logger.Warn("更新数据库头像URL失败", "userID", userID, "error", err.Error())
		} else {
			h.logger.Debug("数据库头像URL已更新", "userID", userID, "url", url)
		}
	}

	// 8. 返回成功响应（URL带时间戳防缓存）
	urlWithTS := fmt.Sprintf("%s?t=%d", url, time.Now().Unix())
	h.logger.Info("上传头像成功", "userID", userID, "username", username, "size", fileHeader.Size)

	utils.SuccessResponse(c, 200, "上传成功", gin.H{
		"url":    urlWithTS,
		"width":  0, // 可扩展：添加图片尺寸检测
		"height": 0,
		"mime":   "image/png",
		"size":   fileHeader.Size,
	})

	// 9. 异步清理历史头像
	go h.cleanupAvatarHistory(username)
}

// getUserInfo 获取用户身份信息
func (h *UploadHandler) getUserInfo(c *gin.Context) (userID uint, username string, err error) {
	userID, err = utils.GetUserIDFromContext(c)
	if err != nil {
		h.logger.Warn("上传头像失败：未认证", "ip", c.ClientIP())
		utils.CodeErrorResponse(c, http.StatusUnauthorized, utils.ErrCodeAuthRequired, "需要登录")
		return 0, "", err
	}

	usernameVal, _ := c.Get("username")
	username, ok := usernameVal.(string)
	if !ok || username == "" {
		h.logger.Error("无法获取用户名", "userID", userID)
		utils.CodeErrorResponse(c, http.StatusInternalServerError, utils.ErrCodeUploadFailed, "用户信息错误")
		return 0, "", fmt.Errorf("username not found")
	}

	return userID, username, nil
}

// receiveAndValidateFile 接收并验证上传的文件
func (h *UploadHandler) receiveAndValidateFile(c *gin.Context, userID uint) (*multipart.FileHeader, error) {
	// 接收文件
	fileHeader, err := c.FormFile("file")
	if err != nil {
		h.logger.Warn("上传头像失败：缺少文件", "userID", userID, "error", err.Error())
		utils.BadRequestResponse(c, "请选择要上传的文件")
		return nil, err
	}

	// 确定文件大小限制
	maxSize := h.maxAvatarSizeBytes
	if maxSize <= 0 {
		maxSize = 5 * 1024 * 1024 // 默认5MB
	}

	// 使用文件验证器（包含大小和魔数验证）
	// 仅支持 PNG 格式
	validator := utils.NewFileValidator(maxSize, []string{"image/png"})
	if err := validator.Validate(fileHeader); err != nil {
		h.logger.Warn("文件验证失败",
			"userID", userID,
			"filename", fileHeader.Filename,
			"size", fileHeader.Size,
			"error", err.Error())

		statusCode := utils.GetHTTPStatusCode(err)
		if statusCode == 413 {
			c.Header("Connection", "close")
			utils.CodeErrorResponse(c, statusCode, utils.ErrCodeUploadTooLarge,
				fmt.Sprintf("文件过大，最大允许%dMB", maxSize/(1024*1024)))
		} else {
			utils.CodeErrorResponse(c, statusCode, utils.ErrCodeUploadInvalidType,
				"仅支持PNG格式图片")
		}
		return nil, err
	}

	return fileHeader, nil
}

// archiveOldAvatar 归档旧头像为历史版本
func (h *UploadHandler) archiveOldAvatar(ctx context.Context, userID uint, username, objectKey string, timestamp int64) {
	// 检查是否存在旧头像
	exists, err := h.storage.ObjectExists(ctx, objectKey)
	if err != nil {
		h.logger.Debug("检查旧头像失败", "userID", userID, "error", err.Error())
		return
	}

	if !exists {
		h.logger.Debug("无旧头像需要归档", "userID", userID)
		return
	}

	// 归档：复制为时间戳命名的历史版本
	archiveKey := fmt.Sprintf("%s/%d.png", username, timestamp)
	if err := h.storage.CopyObject(ctx, objectKey, archiveKey); err != nil {
		h.logger.Warn("归档旧头像失败（不影响上传）",
			"userID", userID,
			"from", objectKey,
			"to", archiveKey,
			"error", err.Error())
		return
	}

	// 删除旧头像（为新头像腾出位置）
	if err := h.storage.RemoveObject(ctx, objectKey); err != nil {
		h.logger.Warn("删除旧头像失败（不影响上传）",
			"userID", userID,
			"object", objectKey,
			"error", err.Error())
	} else {
		h.logger.Debug("旧头像已归档", "userID", userID, "archiveKey", archiveKey)
	}
}

// cleanupAvatarHistory 清理超出限制的历史头像
func (h *UploadHandler) cleanupAvatarHistory(username string) {
	defer func() {
		if r := recover(); r != nil {
			h.logger.Error("清理历史头像panic", "username", username, "panic", r)
		}
	}()

	if h.storage == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 列举用户目录下的所有对象
	objects, err := h.storage.ListObjects(ctx, fmt.Sprintf("%s/", username))
	if err != nil {
		h.logger.Warn("列举历史头像失败", "username", username, "error", err.Error())
		return
	}

	// 过滤出历史头像文件（排除当前头像 avatar.png）
	history := h.filterHistoryAvatars(objects)

	// 如果历史头像数量未超限，无需清理
	if len(history) <= h.maxAvatarHistory {
		h.logger.Debug("历史头像未超限", "username", username, "count", len(history), "limit", h.maxAvatarHistory)
		return
	}

	// 按时间戳降序排序（最新的在前）
	h.sortAvatarsByTimestamp(history)

	// 删除超出限制的旧头像
	toDelete := history[h.maxAvatarHistory:]
	deletedCount := 0
	for _, obj := range toDelete {
		if err := h.storage.RemoveObject(ctx, obj.Key); err != nil {
			h.logger.Warn("删除历史头像失败", "key", obj.Key, "error", err.Error())
		} else {
			deletedCount++
		}
	}

	if deletedCount > 0 {
		h.logger.Info("清理历史头像完成",
			"username", username,
			"deleted", deletedCount,
			"remaining", len(history)-deletedCount)
	}
}

// filterHistoryAvatars 过滤出历史头像文件
func (h *UploadHandler) filterHistoryAvatars(objects []services.ObjectInfo) []services.ObjectInfo {
	history := make([]services.ObjectInfo, 0, len(objects))
	for _, obj := range objects {
		base := path.Base(obj.Key)

		// 排除当前头像
		if strings.EqualFold(base, "avatar.png") {
			continue
		}

		// 仅保留 PNG 文件
		if strings.ToLower(path.Ext(base)) != ".png" {
			continue
		}

		history = append(history, obj)
	}
	return history
}

// sortAvatarsByTimestamp 按时间戳降序排序头像列表
func (h *UploadHandler) sortAvatarsByTimestamp(avatars []services.ObjectInfo) {
	// 从文件名提取时间戳的辅助函数
	parseTimestamp := func(key string, fallback time.Time) int64 {
		base := path.Base(key)
		name := strings.TrimSuffix(base, path.Ext(base))
		if ts, err := strconv.ParseInt(name, 10, 64); err == nil {
			return ts
		}
		return fallback.Unix()
	}

	sort.Slice(avatars, func(i, j int) bool {
		ti := parseTimestamp(avatars[i].Key, avatars[i].LastModified)
		tj := parseTimestamp(avatars[j].Key, avatars[j].LastModified)
		if ti == tj {
			// 时间戳相同时按修改时间排序
			return avatars[i].LastModified.After(avatars[j].LastModified)
		}
		return ti > tj // 降序：新的在前
	})
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
