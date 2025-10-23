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
	resourceStorage    *services.ResourceStorageService
	userService        services.UserServiceInterface
	historyRepo        *services.HistoryRepository
	logger             utils.Logger
	maxAvatarSizeBytes int64
	maxAvatarHistory   int
}

// NewUploadHandler 创建上传处理器
func NewUploadHandler(storage services.StorageClient, resourceStorage *services.ResourceStorageService, userService services.UserServiceInterface, maxAvatarSizeBytes int64, maxAvatarHistory int, historyRepo *services.HistoryRepository) *UploadHandler {
	return &UploadHandler{
		storage:            storage,
		resourceStorage:    resourceStorage,
		userService:        userService,
		historyRepo:        historyRepo,
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
	startTime := time.Now()
	clientIP := c.ClientIP()

	h.logger.Debug("【UploadAvatar】开始处理头像上传请求",
		"ip", clientIP,
		"contentType", c.Request.Header.Get("Content-Type"),
		"contentLength", c.Request.ContentLength)

	// 1. 检查存储服务状态
	if h.storage == nil {
		h.logger.Error("【UploadAvatar】存储服务未初始化",
			"ip", clientIP,
			"duration", time.Since(startTime))
		utils.CodeErrorResponse(c, http.StatusServiceUnavailable, utils.ErrCodeUploadFailed, "存储服务不可用")
		return
	}
	h.logger.Debug("【UploadAvatar】存储服务检查通过")

	// 2. 验证用户认证
	userID, username, err := h.getUserInfo(c)
	if err != nil {
		h.logger.Warn("【UploadAvatar】用户认证失败",
			"ip", clientIP,
			"duration", time.Since(startTime))
		return // 错误已在函数内处理
	}
	h.logger.Debug("【UploadAvatar】用户认证通过",
		"userID", userID,
		"username", username)

	// 3. 接收并验证文件
	fileValidateStart := time.Now()
	fileHeader, err := h.receiveAndValidateFile(c, userID)
	fileValidateLatency := time.Since(fileValidateStart)

	if err != nil {
		h.logger.Warn("【UploadAvatar】文件验证失败",
			"userID", userID,
			"username", username,
			"fileValidateLatency", fileValidateLatency,
			"duration", time.Since(startTime))
		return // 错误已在函数内处理
	}
	h.logger.Debug("【UploadAvatar】文件验证通过",
		"userID", userID,
		"filename", fileHeader.Filename,
		"fileSize", fileHeader.Size,
		"fileValidateLatency", fileValidateLatency)

	// 4. 打开文件准备上传
	file, err := fileHeader.Open()
	if err != nil {
		h.logger.Error("【UploadAvatar】打开上传文件失败",
			"userID", userID,
			"filename", fileHeader.Filename,
			"error", err.Error(),
			"duration", time.Since(startTime))
		utils.CodeErrorResponse(c, http.StatusInternalServerError, utils.ErrCodeUploadFailed, "无法读取文件")
		return
	}
	defer file.Close()
	h.logger.Debug("【UploadAvatar】文件已打开", "userID", userID)

	// 5. 归档旧头像（不阻塞上传流程）
	timestamp := time.Now().Unix()
	objectKey := fmt.Sprintf("%s/avatar.png", username)

	h.logger.Debug("【UploadAvatar】开始归档旧头像",
		"userID", userID,
		"objectKey", objectKey,
		"timestamp", timestamp)
	archiveStart := time.Now()
	h.archiveOldAvatar(c.Request.Context(), userID, username, objectKey, timestamp)
	archiveLatency := time.Since(archiveStart)
	h.logger.Debug("【UploadAvatar】旧头像归档完成",
		"userID", userID,
		"archiveLatency", archiveLatency)

	// 6. 上传新头像（PNG格式）
	h.logger.Debug("【UploadAvatar】开始上传新头像到对象存储",
		"userID", userID,
		"objectKey", objectKey,
		"fileSize", fileHeader.Size)
	uploadStart := time.Now()
	url, err := h.storage.PutObject(c.Request.Context(), objectKey, "image/png", file, fileHeader.Size)
	uploadLatency := time.Since(uploadStart)

	if err != nil {
		h.logger.Error("【UploadAvatar】上传到对象存储失败",
			"userID", userID,
			"username", username,
			"objectKey", objectKey,
			"error", err.Error(),
			"uploadLatency", uploadLatency,
			"duration", time.Since(startTime))
		utils.CodeErrorResponse(c, http.StatusInternalServerError, utils.ErrCodeUploadFailed, "上传失败")
		return
	}
	h.logger.Debug("【UploadAvatar】新头像上传成功",
		"userID", userID,
		"url", url,
		"uploadLatency", uploadLatency)

	// 7. 更新数据库中的头像URL（不带时间戳，因为文件名固定）
	if h.userService != nil {
		h.logger.Debug("【UploadAvatar】开始更新数据库头像URL", "userID", userID)
		dbUpdateStart := time.Now()

		// 先获取旧头像URL（用于历史记录）
		oldProfile, _ := h.userService.GetUserProfile(c.Request.Context(), userID)
		oldAvatarURL := ""
		if oldProfile != nil {
			oldAvatarURL = oldProfile.AvatarURL
		}

		prof := &models.UserExtraProfile{
			UserID:    userID,
			AvatarURL: url, // 使用不带时间戳的URL存储到数据库
		}
		err := h.userService.UpdateUserAvatar(c.Request.Context(), prof)
		dbUpdateLatency := time.Since(dbUpdateStart)

		if err != nil {
			// 数据库更新失败不影响上传结果，仅记录日志
			h.logger.Warn("【UploadAvatar】更新数据库头像URL失败",
				"userID", userID,
				"url", url,
				"error", err.Error(),
				"dbUpdateLatency", dbUpdateLatency)
		} else {
			h.logger.Debug("【UploadAvatar】数据库头像URL已更新",
				"userID", userID,
				"url", url,
				"dbUpdateLatency", dbUpdateLatency)

			// 异步记录头像修改历史
			if h.historyRepo != nil {
				go func() {
					h.historyRepo.RecordProfileChange(userID, "avatar", oldAvatarURL, url, clientIP)
					h.historyRepo.RecordOperationHistory(userID, username, "修改头像",
						fmt.Sprintf("上传新头像: %s", fileHeader.Filename), clientIP)
					h.logger.Debug("【UploadAvatar】头像修改历史已记录", "userID", userID)
				}()
			}
		}
	}

	// 8. 返回成功响应（URL带时间戳防缓存）
	urlWithTS := fmt.Sprintf("%s?t=%d", url, time.Now().Unix())

	h.logger.Info("【UploadAvatar】上传头像成功",
		"userID", userID,
		"username", username,
		"filename", fileHeader.Filename,
		"fileSize", fileHeader.Size,
		"url", url,
		"totalDuration", time.Since(startTime),
		"breakdown", map[string]interface{}{
			"fileValidate": fileValidateLatency.Milliseconds(),
			"archive":      archiveLatency.Milliseconds(),
			"upload":       uploadLatency.Milliseconds(),
		})

	utils.SuccessResponse(c, 200, "上传成功", gin.H{
		"url":    urlWithTS,
		"width":  0, // 可扩展：添加图片尺寸检测
		"height": 0,
		"mime":   "image/png",
		"size":   fileHeader.Size,
	})

	// 9. 异步清理历史头像
	h.logger.Debug("【UploadAvatar】启动异步清理历史头像任务",
		"userID", userID,
		"username", username)
	go h.cleanupAvatarHistory(username)
}

// getUserInfo 获取用户身份信息
func (h *UploadHandler) getUserInfo(c *gin.Context) (userID uint, username string, err error) {
	h.logger.Debug("【getUserInfo】获取用户身份信息", "ip", c.ClientIP())

	userID, err = utils.GetUserIDFromContext(c)
	if err != nil {
		h.logger.Warn("【getUserInfo】上传头像失败：未认证",
			"ip", c.ClientIP(),
			"error", err.Error())
		utils.CodeErrorResponse(c, http.StatusUnauthorized, utils.ErrCodeAuthRequired, "需要登录")
		return 0, "", err
	}

	usernameVal, _ := c.Get("username")
	username, ok := usernameVal.(string)
	if !ok || username == "" {
		h.logger.Error("【getUserInfo】无法获取用户名",
			"userID", userID,
			"usernameVal", usernameVal,
			"ok", ok)
		utils.CodeErrorResponse(c, http.StatusInternalServerError, utils.ErrCodeUploadFailed, "用户信息错误")
		return 0, "", fmt.Errorf("username not found")
	}

	h.logger.Debug("【getUserInfo】用户身份信息获取成功",
		"userID", userID,
		"username", username)
	return userID, username, nil
}

// receiveAndValidateFile 接收并验证上传的文件
func (h *UploadHandler) receiveAndValidateFile(c *gin.Context, userID uint) (*multipart.FileHeader, error) {
	h.logger.Debug("【receiveAndValidateFile】开始接收文件", "userID", userID)

	// 接收文件
	fileHeader, err := c.FormFile("file")
	if err != nil {
		h.logger.Warn("【receiveAndValidateFile】上传头像失败：缺少文件",
			"userID", userID,
			"error", err.Error())
		utils.BadRequestResponse(c, "请选择要上传的文件")
		return nil, err
	}

	h.logger.Debug("【receiveAndValidateFile】文件接收成功",
		"userID", userID,
		"filename", fileHeader.Filename,
		"size", fileHeader.Size,
		"header", fileHeader.Header)

	// 确定文件大小限制
	maxSize := h.maxAvatarSizeBytes
	if maxSize <= 0 {
		maxSize = 5 * 1024 * 1024 // 默认5MB
	}

	h.logger.Debug("【receiveAndValidateFile】开始验证文件",
		"userID", userID,
		"filename", fileHeader.Filename,
		"fileSize", fileHeader.Size,
		"maxSize", maxSize,
		"maxSizeMB", maxSize/(1024*1024))

	// 使用文件验证器（包含大小和魔数验证）
	// 仅支持 PNG 格式
	validator := utils.NewFileValidator(maxSize, []string{"image/png"})
	if err := validator.Validate(fileHeader); err != nil {
		h.logger.Warn("【receiveAndValidateFile】文件验证失败",
			"userID", userID,
			"filename", fileHeader.Filename,
			"size", fileHeader.Size,
			"maxSize", maxSize,
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

	h.logger.Debug("【receiveAndValidateFile】文件验证通过",
		"userID", userID,
		"filename", fileHeader.Filename,
		"size", fileHeader.Size)

	return fileHeader, nil
}

// archiveOldAvatar 归档旧头像为历史版本
func (h *UploadHandler) archiveOldAvatar(ctx context.Context, userID uint, username, objectKey string, timestamp int64) {
	h.logger.Debug("【archiveOldAvatar】开始归档旧头像",
		"userID", userID,
		"username", username,
		"objectKey", objectKey,
		"timestamp", timestamp)

	// 检查存储服务
	if h.storage == nil {
		h.logger.Warn("【archiveOldAvatar】存储服务未配置，跳过归档", "userID", userID)
		return
	}

	// 检查是否存在旧头像
	checkStart := time.Now()
	exists, err := h.storage.ObjectExists(ctx, objectKey)
	checkLatency := time.Since(checkStart)

	if err != nil {
		h.logger.Debug("【archiveOldAvatar】检查旧头像失败",
			"userID", userID,
			"objectKey", objectKey,
			"error", err.Error(),
			"checkLatency", checkLatency)
		return
	}

	if !exists {
		h.logger.Debug("【archiveOldAvatar】无旧头像需要归档",
			"userID", userID,
			"objectKey", objectKey,
			"checkLatency", checkLatency)
		return
	}

	h.logger.Debug("【archiveOldAvatar】检测到旧头像，开始归档",
		"userID", userID,
		"objectKey", objectKey)

	// 归档：复制为时间戳命名的历史版本
	archiveKey := fmt.Sprintf("%s/%d.png", username, timestamp)
	copyStart := time.Now()
	err = h.storage.CopyObject(ctx, objectKey, archiveKey)
	copyLatency := time.Since(copyStart)

	if err != nil {
		h.logger.Warn("【archiveOldAvatar】归档旧头像失败（不影响上传）",
			"userID", userID,
			"from", objectKey,
			"to", archiveKey,
			"error", err.Error(),
			"copyLatency", copyLatency)
		return
	}

	h.logger.Debug("【archiveOldAvatar】旧头像复制成功",
		"userID", userID,
		"archiveKey", archiveKey,
		"copyLatency", copyLatency)

	// 删除旧头像（为新头像腾出位置）
	deleteStart := time.Now()
	err = h.storage.RemoveObject(ctx, objectKey)
	deleteLatency := time.Since(deleteStart)

	if err != nil {
		h.logger.Warn("【archiveOldAvatar】删除旧头像失败（不影响上传）",
			"userID", userID,
			"object", objectKey,
			"error", err.Error(),
			"deleteLatency", deleteLatency)
	} else {
		h.logger.Debug("【archiveOldAvatar】旧头像已归档并删除",
			"userID", userID,
			"archiveKey", archiveKey,
			"deleteLatency", deleteLatency)
	}
}

// cleanupAvatarHistory 清理超出限制的历史头像
func (h *UploadHandler) cleanupAvatarHistory(username string) {
	startTime := time.Now()
	h.logger.Debug("【cleanupAvatarHistory】开始清理历史头像",
		"username", username,
		"maxHistory", h.maxAvatarHistory)

	defer func() {
		if r := recover(); r != nil {
			h.logger.Error("【cleanupAvatarHistory】清理历史头像panic",
				"username", username,
				"panic", r,
				"duration", time.Since(startTime))
		}
	}()

	if h.storage == nil {
		h.logger.Warn("【cleanupAvatarHistory】存储服务未初始化", "username", username)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 列举用户目录下的所有对象
	listStart := time.Now()
	objects, err := h.storage.ListObjects(ctx, fmt.Sprintf("%s/", username))
	listLatency := time.Since(listStart)

	if err != nil {
		h.logger.Warn("【cleanupAvatarHistory】列举历史头像失败",
			"username", username,
			"error", err.Error(),
			"listLatency", listLatency,
			"duration", time.Since(startTime))
		return
	}

	h.logger.Debug("【cleanupAvatarHistory】历史头像列举完成",
		"username", username,
		"totalObjects", len(objects),
		"listLatency", listLatency)

	// 过滤出历史头像文件（排除当前头像 avatar.png）
	filterStart := time.Now()
	history := h.filterHistoryAvatars(objects)
	filterLatency := time.Since(filterStart)

	h.logger.Debug("【cleanupAvatarHistory】历史头像过滤完成",
		"username", username,
		"historyCount", len(history),
		"filterLatency", filterLatency)

	// 如果历史头像数量未超限，无需清理
	if len(history) <= h.maxAvatarHistory {
		h.logger.Debug("【cleanupAvatarHistory】历史头像未超限，无需清理",
			"username", username,
			"count", len(history),
			"limit", h.maxAvatarHistory,
			"duration", time.Since(startTime))
		return
	}

	h.logger.Debug("【cleanupAvatarHistory】历史头像超限，开始清理",
		"username", username,
		"count", len(history),
		"limit", h.maxAvatarHistory,
		"toDelete", len(history)-h.maxAvatarHistory)

	// 按时间戳降序排序（最新的在前）
	sortStart := time.Now()
	h.sortAvatarsByTimestamp(history)
	sortLatency := time.Since(sortStart)

	h.logger.Debug("【cleanupAvatarHistory】历史头像排序完成",
		"username", username,
		"sortLatency", sortLatency)

	// 删除超出限制的旧头像
	toDelete := history[h.maxAvatarHistory:]
	deletedCount := 0
	deleteStart := time.Now()

	for _, obj := range toDelete {
		if err := h.storage.RemoveObject(ctx, obj.Key); err != nil {
			h.logger.Warn("【cleanupAvatarHistory】删除历史头像失败",
				"username", username,
				"key", obj.Key,
				"error", err.Error())
		} else {
			deletedCount++
			h.logger.Debug("【cleanupAvatarHistory】已删除历史头像",
				"username", username,
				"key", obj.Key)
		}
	}
	deleteLatency := time.Since(deleteStart)

	if deletedCount > 0 {
		h.logger.Info("【cleanupAvatarHistory】清理历史头像完成",
			"username", username,
			"deleted", deletedCount,
			"remaining", len(history)-deletedCount,
			"deleteLatency", deleteLatency,
			"totalDuration", time.Since(startTime))
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

// UploadResourceImage 上传资源预览图
func (h *UploadHandler) UploadResourceImage(c *gin.Context) {
	// 验证用户登录
	_, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, "未登录")
		return
	}

	// 检查资源存储服务是否可用
	if h.resourceStorage == nil {
		utils.InternalServerErrorResponse(c, "资源存储服务未配置")
		return
	}

	// 解析上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		h.logger.Warn("解析上传文件失败", "error", err.Error())
		utils.BadRequestResponse(c, "未找到上传文件")
		return
	}
	defer file.Close()

	// 验证文件大小（最大5MB）
	maxSize := int64(5 * 1024 * 1024)
	if header.Size > maxSize {
		utils.BadRequestResponse(c, "图片大小不能超过5MB")
		return
	}

	// 验证文件类型
	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		utils.BadRequestResponse(c, "只能上传图片文件")
		return
	}

	// 上传到临时目录
	ctx := c.Request.Context()
	imageURL, err := h.resourceStorage.UploadResourceImage(ctx, file, header.Filename, header.Size)
	if err != nil {
		h.logger.Error("上传资源图片失败", "error", err.Error())
		utils.InternalServerErrorResponse(c, "上传失败")
		return
	}

	h.logger.Info("资源图片上传成功", "filename", header.Filename, "url", imageURL)
	utils.SuccessResponse(c, 200, "上传成功", gin.H{
		"image_url": imageURL,
	})
}

// UploadDocumentImage 上传文档图片
func (h *UploadHandler) UploadDocumentImage(c *gin.Context) {
	// 验证用户登录
	_, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, "未登录")
		return
	}

	// 检查资源存储服务是否可用
	if h.resourceStorage == nil {
		utils.InternalServerErrorResponse(c, "资源存储服务未配置")
		return
	}

	// 解析上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		h.logger.Warn("解析上传文件失败", "error", err.Error())
		utils.BadRequestResponse(c, "未找到上传文件")
		return
	}
	defer file.Close()

	// 验证文件大小（最大5MB）
	maxSize := int64(5 * 1024 * 1024)
	if header.Size > maxSize {
		utils.BadRequestResponse(c, "图片大小不能超过5MB")
		return
	}

	// 验证文件类型
	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		utils.BadRequestResponse(c, "只能上传图片文件")
		return
	}

	// 上传到文档目录
	ctx := c.Request.Context()
	imageURL, err := h.resourceStorage.UploadDocumentImage(ctx, file, header.Filename, header.Size)
	if err != nil {
		h.logger.Error("上传文档图片失败", "error", err.Error())
		utils.InternalServerErrorResponse(c, "上传失败")
		return
	}

	h.logger.Info("文档图片上传成功", "filename", header.Filename, "url", imageURL)
	utils.SuccessResponse(c, 200, "上传成功", gin.H{
		"image_url": imageURL,
	})
}
