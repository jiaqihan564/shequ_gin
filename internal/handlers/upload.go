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

	"gin/internal/config"
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
	config             *config.Config
}

// NewUploadHandler 创建上传处理器
func NewUploadHandler(storage services.StorageClient, resourceStorage *services.ResourceStorageService, userService services.UserServiceInterface, maxAvatarSizeBytes int64, maxAvatarHistory int, historyRepo *services.HistoryRepository, cfg *config.Config) *UploadHandler {
	return &UploadHandler{
		storage:            storage,
		resourceStorage:    resourceStorage,
		userService:        userService,
		historyRepo:        historyRepo,
		logger:             utils.GetLogger(),
		maxAvatarSizeBytes: maxAvatarSizeBytes,
		maxAvatarHistory:   maxAvatarHistory,
		config:             cfg,
	}
}

// UploadAvatar 上传头像到对象存储
func (h *UploadHandler) UploadAvatar(c *gin.Context) {
	reqCtx := extractRequestContext(c)

	// 检查存储服务状态
	if h.storage == nil {
		h.logger.Error("存储服务未初始化", "ip", reqCtx.ClientIP)
		utils.CodeErrorResponse(c, http.StatusServiceUnavailable, utils.ErrCodeUploadFailed, "存储服务不可用")
		return
	}

	// 验证用户认证
	userID, username, err := h.getUserInfo(c)
	if err != nil {
		return // 错误已在函数内处理
	}

	// 接收并验证文件
	fileHeader, err := h.receiveAndValidateFile(c, userID)
	if err != nil {
		return // 错误已在函数内处理
	}

	// 打开文件准备上传（前端已经裁剪和压缩好了）
	file, err := fileHeader.Open()
	if err != nil {
		h.logger.Error("打开上传文件失败", "userID", userID, "error", err.Error())
		utils.CodeErrorResponse(c, http.StatusInternalServerError, utils.ErrCodeUploadFailed, "无法读取文件")
		return
	}
	defer file.Close()

	// 归档旧头像（不阻塞上传流程）
	timestamp := time.Now().Unix()
	objectKey := fmt.Sprintf("%s/avatar.png", username)

	// 获取旧头像URL（在归档之前）
	oldProfile, _ := h.userService.GetUserProfile(c.Request.Context(), userID)
	oldAvatarURL := ""
	archivedAvatarURL := ""
	if oldProfile != nil && oldProfile.AvatarURL != "" {
		oldAvatarURL = oldProfile.AvatarURL
		// 生成归档文件的URL
		archivedAvatarURL = h.getArchivedAvatarURL(username, timestamp)
	}

	h.archiveOldAvatar(c.Request.Context(), userID, username, objectKey, timestamp)

	// 直接上传（前端已处理好，统一为JPEG格式）
	contentType := "image/jpeg"
	url, err := h.storage.PutObject(c.Request.Context(), objectKey, contentType, file, fileHeader.Size)
	if err != nil {
		h.logger.Error("上传到对象存储失败",
			"userID", userID,
			"username", username,
			"error", err.Error())
		utils.CodeErrorResponse(c, http.StatusInternalServerError, utils.ErrCodeUploadFailed, "上传失败")
		return
	}

	// 更新数据库中的头像URL（带回滚机制）
	dbUpdateSuccess := false
	if h.userService != nil {

		prof := &models.UserExtraProfile{
			UserID:    userID,
			AvatarURL: url,
		}
		err := h.userService.UpdateUserAvatar(c.Request.Context(), prof)
		if err != nil {
			// 数据库更新失败，需要回滚（删除已上传的文件）
			h.logger.Error("更新数据库头像URL失败，开始回滚",
				"userID", userID,
				"error", err.Error())

			// 尝试删除刚上传的文件
			if deleteErr := h.storage.RemoveObject(c.Request.Context(), objectKey); deleteErr != nil {
				h.logger.Error("回滚失败：无法删除已上传的头像",
					"userID", userID,
					"objectKey", objectKey,
					"error", deleteErr.Error())
			} else {
				h.logger.Info("回滚成功：已删除上传的头像", "userID", userID)
			}

			utils.CodeErrorResponse(c, http.StatusInternalServerError,
				utils.ErrCodeUploadFailed, "头像上传失败，请重试")
			return
		}

		dbUpdateSuccess = true

		// 使用Worker Pool记录头像修改历史（避免goroutine泄漏）
		if h.historyRepo != nil {
			taskID := fmt.Sprintf("avatar_history_%d_%d", userID, time.Now().Unix())
			// 使用归档后的URL作为旧值，这样历史记录中可以看到真正的旧头像
			historyOldURL := archivedAvatarURL
			if historyOldURL == "" {
				historyOldURL = oldAvatarURL
			}
			_ = utils.SubmitTask(taskID, func(taskCtx context.Context) error {
				h.historyRepo.RecordProfileChange(userID, "avatar", historyOldURL, url, reqCtx.ClientIP)
				h.historyRepo.RecordOperationHistory(userID, username, "修改头像",
					fmt.Sprintf("上传新头像: %s (大小: %d字节)",
						fileHeader.Filename,
						fileHeader.Size), reqCtx.ClientIP)
				return nil
			}, time.Duration(h.config.AsyncTasks.UploadHistoryTimeout)*time.Second)
		}
	}

	// 如果数据库更新失败，这里已经返回了，不会执行下面的代码
	if !dbUpdateSuccess {
		h.logger.Error("用户服务未初始化", "userID", userID)
		utils.CodeErrorResponse(c, http.StatusInternalServerError,
			utils.ErrCodeUploadFailed, "服务暂时不可用")
		return
	}

	// 返回成功响应（URL带时间戳防缓存）
	urlWithTS := fmt.Sprintf("%s?t=%d", url, time.Now().Unix())

	h.logger.Info("上传头像成功",
		"userID", userID,
		"username", username,
		"filename", fileHeader.Filename,
		"fileSize", fileHeader.Size,
		"duration", time.Since(reqCtx.StartTime))

	utils.SuccessResponse(c, 200, "上传成功", gin.H{
		"url":  urlWithTS,
		"mime": contentType,
		"size": fileHeader.Size,
	})

	// 使用Worker Pool异步清理历史头像（避免goroutine泄漏）
	taskID := fmt.Sprintf("cleanup_avatar_%s_%d", username, time.Now().Unix())
	_ = utils.SubmitTask(taskID, func(ctx context.Context) error {
		h.cleanupAvatarHistory(username)
		return nil
	}, time.Duration(h.config.AsyncTasks.AvatarCleanupTimeout)*time.Second)
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
	fileHeader, err := c.FormFile("file")
	if err != nil {
		h.logger.Warn("上传头像失败：缺少文件", "userID", userID, "error", err.Error())
		utils.BadRequestResponse(c, "请选择要上传的文件")
		return nil, err
	}

	// 强制使用5KB限制（极限压缩）
	maxSize := int64(5 * 1024) // 固定5KB

	// 先检查文件大小（在验证器之前，这样可以给出更友好的提示）
	if fileHeader.Size > maxSize {
		actualKB := float64(fileHeader.Size) / 1024
		h.logger.Warn("❌ 文件过大",
			"userID", userID,
			"filename", fileHeader.Filename,
			"fileSize", fileHeader.Size,
			"fileSizeKB", actualKB,
			"maxAllowed", maxSize,
			"maxAllowedKB", 5)

		// 友好的错误提示（不暴露具体数据）
		utils.CodeErrorResponse(c, 413, utils.ErrCodeUploadTooLarge,
			"图片过大，请选择更小的图片或裁剪后重试")
		return nil, fmt.Errorf("file too large: %d bytes", fileHeader.Size)
	}

	validator := utils.NewFileValidator(maxSize, []string{"image/png", "image/jpeg"})
	if err := validator.Validate(fileHeader); err != nil {
		h.logger.Warn("❌ 文件验证失败",
			"userID", userID,
			"filename", fileHeader.Filename,
			"fileSize", fileHeader.Size,
			"maxAllowed", maxSize,
			"error", err.Error())
		statusCode := utils.GetHTTPStatusCode(err)
		if statusCode == 413 {
			c.Header("Connection", "close")
			// 不暴露具体大小，给出友好提示
			utils.CodeErrorResponse(c, statusCode, utils.ErrCodeUploadTooLarge,
				"图片过大，请选择更小的图片或裁剪后重试")
		} else {
			utils.CodeErrorResponse(c, statusCode, utils.ErrCodeUploadInvalidType, "仅支持PNG或JPEG格式图片")
		}
		return nil, err
	}

	// 记录验证成功
	h.logger.Info("✅ 文件验证通过",
		"userID", userID,
		"filename", fileHeader.Filename,
		"fileSize", fileHeader.Size,
		"fileSizeKB", fileHeader.Size/1024,
		"maxAllowedKB", maxSize/1024)

	return fileHeader, nil
}

// archiveOldAvatar 归档旧头像为历史版本
func (h *UploadHandler) archiveOldAvatar(ctx context.Context, userID uint, username, objectKey string, timestamp int64) {
	if h.storage == nil {
		return
	}

	exists, err := h.storage.ObjectExists(ctx, objectKey)
	if err != nil || !exists {
		return
	}

	// 归档：复制为时间戳命名的历史版本
	archiveKey := fmt.Sprintf("%s/%d.png", username, timestamp)
	err = h.storage.CopyObject(ctx, objectKey, archiveKey)
	if err != nil {
		h.logger.Warn("归档旧头像失败（不影响上传）", "userID", userID, "error", err.Error())
		return
	}

	// 删除旧头像
	err = h.storage.RemoveObject(ctx, objectKey)
	if err != nil {
		h.logger.Warn("删除旧头像失败（不影响上传）", "userID", userID, "error", err.Error())
	}
}

// getArchivedAvatarURL 生成归档头像的URL
func (h *UploadHandler) getArchivedAvatarURL(username string, timestamp int64) string {
	if h.storage == nil {
		return ""
	}
	archiveKey := fmt.Sprintf("%s/%d.png", username, timestamp)
	publicBase := h.config.Assets.PublicBaseURL
	return fmt.Sprintf("%s/%s", publicBase, archiveKey)
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(h.config.AsyncTasks.AvatarOperationTimeout)*time.Second)
	defer cancel()

	objects, err := h.storage.ListObjects(ctx, fmt.Sprintf("%s/", username))
	if err != nil {
		h.logger.Warn("列举历史头像失败", "username", username, "error", err.Error())
		return
	}

	// 过滤出历史头像文件（排除当前头像 avatar.png）
	history := h.filterHistoryAvatars(objects)

	// 如果历史头像数量未超限，无需清理
	if len(history) <= h.maxAvatarHistory {
		return
	}

	// 按时间戳降序排序（最新的在前）
	h.sortAvatarsByTimestamp(history)

	// 删除超出限制的旧头像
	toDelete := history[h.maxAvatarHistory:]
	deletedCount := 0

	for _, obj := range toDelete {
		if err := h.storage.RemoveObject(ctx, obj.Key); err != nil {
			h.logger.Warn("删除历史头像失败", "username", username, "key", obj.Key, "error", err.Error())
		} else {
			deletedCount++
		}
	}

	if deletedCount > 0 {
		h.logger.Info("清理历史头像完成", "username", username, "deleted", deletedCount)
	}
}

// filterHistoryAvatars 过滤出历史头像文件
func (h *UploadHandler) filterHistoryAvatars(objects []services.ObjectInfo) []services.ObjectInfo {
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
	return history
}

// sortAvatarsByTimestamp 按时间戳降序排序头像列表
func (h *UploadHandler) sortAvatarsByTimestamp(avatars []services.ObjectInfo) {
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
			return avatars[i].LastModified.After(avatars[j].LastModified)
		}
		return ti > tj
	})
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
		if count >= h.config.Pagination.AvatarHistoryMaxList {
			break
		}
	}

	utils.SuccessResponse(c, 200, "OK", gin.H{"items": items})
}

// validateImageFile 验证图片文件（通用方法）
func (h *UploadHandler) validateImageFile(header *multipart.FileHeader) error {
	maxSize := int64(h.config.ImageUpload.MaxSizeMB * 1024 * 1024)
	validator := utils.NewFileValidator(maxSize, []string{
		"image/png", "image/jpeg", "image/gif", "image/webp",
	})
	return validator.Validate(header)
}

// UploadResourceImage 上传资源预览图
func (h *UploadHandler) UploadResourceImage(c *gin.Context) {
	_, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, "未登录")
		return
	}

	if h.resourceStorage == nil {
		utils.InternalServerErrorResponse(c, "资源存储服务未配置")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		h.logger.Warn("解析上传文件失败", "error", err.Error())
		utils.BadRequestResponse(c, "未找到上传文件")
		return
	}
	defer file.Close()

	// 验证图片文件
	if err := h.validateImageFile(header); err != nil {
		h.logger.Warn("文件验证失败", "filename", header.Filename, "error", err.Error())
		statusCode := utils.GetHTTPStatusCode(err)
		if statusCode == 413 {
			utils.BadRequestResponse(c, fmt.Sprintf("图片大小不能超过%dMB", h.config.ImageUpload.MaxSizeMB))
		} else {
			utils.BadRequestResponse(c, "只能上传PNG、JPEG、GIF或WebP格式的图片")
		}
		return
	}

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
	_, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, "未登录")
		return
	}

	if h.resourceStorage == nil {
		utils.InternalServerErrorResponse(c, "资源存储服务未配置")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		h.logger.Warn("解析上传文件失败", "error", err.Error())
		utils.BadRequestResponse(c, "未找到上传文件")
		return
	}
	defer file.Close()

	// 验证图片文件
	if err := h.validateImageFile(header); err != nil {
		h.logger.Warn("文件验证失败", "filename", header.Filename, "error", err.Error())
		statusCode := utils.GetHTTPStatusCode(err)
		if statusCode == 413 {
			utils.BadRequestResponse(c, fmt.Sprintf("图片大小不能超过%dMB", h.config.ImageUpload.MaxSizeMB))
		} else {
			utils.BadRequestResponse(c, "只能上传PNG、JPEG、GIF或WebP格式的图片")
		}
		return
	}

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
