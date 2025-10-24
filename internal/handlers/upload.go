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

	// 打开文件准备上传
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
	h.archiveOldAvatar(c.Request.Context(), userID, username, objectKey, timestamp)

	// 上传新头像
	url, err := h.storage.PutObject(c.Request.Context(), objectKey, "image/png", file, fileHeader.Size)
	if err != nil {
		h.logger.Error("上传到对象存储失败",
			"userID", userID,
			"username", username,
			"error", err.Error())
		utils.CodeErrorResponse(c, http.StatusInternalServerError, utils.ErrCodeUploadFailed, "上传失败")
		return
	}

	// 更新数据库中的头像URL
	if h.userService != nil {
		oldProfile, _ := h.userService.GetUserProfile(c.Request.Context(), userID)
		oldAvatarURL := ""
		if oldProfile != nil {
			oldAvatarURL = oldProfile.AvatarURL
		}

		prof := &models.UserExtraProfile{
			UserID:    userID,
			AvatarURL: url,
		}
		err := h.userService.UpdateUserAvatar(c.Request.Context(), prof)
		if err != nil {
			h.logger.Warn("更新数据库头像URL失败", "userID", userID, "error", err.Error())
		} else {
			// 异步记录头像修改历史
			if h.historyRepo != nil {
				go func() {
					h.historyRepo.RecordProfileChange(userID, "avatar", oldAvatarURL, url, reqCtx.ClientIP)
					h.historyRepo.RecordOperationHistory(userID, username, "修改头像",
						fmt.Sprintf("上传新头像: %s", fileHeader.Filename), reqCtx.ClientIP)
				}()
			}
		}
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
		"url":    urlWithTS,
		"width":  0,
		"height": 0,
		"mime":   "image/png",
		"size":   fileHeader.Size,
	})

	// 异步清理历史头像
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
	fileHeader, err := c.FormFile("file")
	if err != nil {
		h.logger.Warn("上传头像失败：缺少文件", "userID", userID, "error", err.Error())
		utils.BadRequestResponse(c, "请选择要上传的文件")
		return nil, err
	}

	maxSize := h.maxAvatarSizeBytes
	if maxSize <= 0 {
		maxSize = 5 * 1024 * 1024 // 默认5MB
	}

	validator := utils.NewFileValidator(maxSize, []string{"image/png"})
	if err := validator.Validate(fileHeader); err != nil {
		h.logger.Warn("文件验证失败", "userID", userID, "filename", fileHeader.Filename, "error", err.Error())
		statusCode := utils.GetHTTPStatusCode(err)
		if statusCode == 413 {
			c.Header("Connection", "close")
			utils.CodeErrorResponse(c, statusCode, utils.ErrCodeUploadTooLarge,
				fmt.Sprintf("文件过大，最大允许%dMB", maxSize/(1024*1024)))
		} else {
			utils.CodeErrorResponse(c, statusCode, utils.ErrCodeUploadInvalidType, "仅支持PNG格式图片")
		}
		return nil, err
	}

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
		if count >= 50 {
			break
		}
	}

	utils.SuccessResponse(c, 200, "OK", gin.H{"items": items})
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

	maxSize := int64(5 * 1024 * 1024)
	if header.Size > maxSize {
		utils.BadRequestResponse(c, "图片大小不能超过5MB")
		return
	}

	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		utils.BadRequestResponse(c, "只能上传图片文件")
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

	maxSize := int64(5 * 1024 * 1024)
	if header.Size > maxSize {
		utils.BadRequestResponse(c, "图片大小不能超过5MB")
		return
	}

	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		utils.BadRequestResponse(c, "只能上传图片文件")
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
