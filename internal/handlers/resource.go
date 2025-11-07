package handlers

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gin/internal/config"
	"gin/internal/models"
	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

// ResourceHandler 资源处理器
type ResourceHandler struct {
	resourceRepo        *services.ResourceRepository
	resourceCommentRepo *services.ResourceCommentRepository
	resourceStorage     *services.ResourceStorageService
	userRepo            *services.UserRepository
	logger              utils.Logger
	config              *config.Config
}

// NewResourceHandler 创建资源处理器
func NewResourceHandler(resourceRepo *services.ResourceRepository, resourceCommentRepo *services.ResourceCommentRepository, resourceStorage *services.ResourceStorageService, userRepo *services.UserRepository, cfg *config.Config) *ResourceHandler {
	return &ResourceHandler{
		resourceRepo:        resourceRepo,
		resourceCommentRepo: resourceCommentRepo,
		resourceStorage:     resourceStorage,
		userRepo:            userRepo,
		logger:              utils.GetLogger(),
		config:              cfg,
	}
}

// CreateResource 创建资源
func (h *ResourceHandler) CreateResource(c *gin.Context) {
	userID, isOK := getUserIDOrFail(c)
	if !isOK {
		return
	}

	var req models.CreateResourceRequest
	if !bindJSONOrFail(c, &req, h.logger, "CreateResource") {
		return
	}

	// 记录接收到的请求数据
	h.logger.Info("接收创建资源请求",
		"userID", userID,
		"title", req.Title,
		"imageCount", len(req.ImageURLs),
		"imageURLs", req.ImageURLs,
	)

	// 提取文件扩展名
	fileExt := ""
	for i := len(req.FileName) - 1; i >= 0; i-- {
		if req.FileName[i] == '.' {
			fileExt = req.FileName[i+1:]
			break
		}
	}
	if fileExt == "" {
		fileExt = "unknown"
	}

	// 创建资源对象
	resource := &models.Resource{
		UserID:        userID,
		Title:         req.Title,
		Description:   req.Description,
		Document:      req.Document,
		CategoryID:    req.CategoryID,
		FileName:      req.FileName,
		FileSize:      req.FileSize,
		FileType:      req.FileType,
		FileExtension: fileExt,
		StoragePath:   req.StoragePath,
		Status:        1,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	ctx := c.Request.Context()

	// 先创建资源记录以获取resourceID
	err := h.resourceRepo.CreateResource(ctx, resource, []string{}, req.Tags)
	if err != nil {
		h.logger.Error("创建资源失败", "userID", userID, "error", err.Error())
		utils.ErrorResponse(c, 500, "创建资源失败")
		return
	}

	// 如果有临时图片URL，移动到正式目录
	finalImageURLs := req.ImageURLs
	if len(req.ImageURLs) > 0 && h.resourceStorage != nil {
		movedURLs, err := h.resourceStorage.MoveResourceImages(ctx, req.ImageURLs, resource.ID)
		if err != nil {
			h.logger.Warn("移动资源图片失败", "resourceID", resource.ID, "error", err.Error())
			// 不中断创建流程，使用原始URL
		} else {
			finalImageURLs = movedURLs
			h.logger.Info("成功移动资源图片", "resourceID", resource.ID, "count", len(movedURLs))
		}

		// 更新资源的图片记录
		if len(finalImageURLs) > 0 {
			// 这里需要调用repository更新图片URLs
			// 暂时通过重新保存实现
			_ = h.resourceRepo.UpdateResourceImages(ctx, resource.ID, finalImageURLs)
		}
	}

	h.logger.Info("创建资源成功", "resourceID", resource.ID, "userID", userID)
	utils.SuccessResponse(c, 201, "创建成功", gin.H{
		"resource_id": resource.ID,
	})
}

// GetResourceDetail 获取资源详情
func (h *ResourceHandler) GetResourceDetail(c *gin.Context) {
	resourceIDStr := c.Param("id")
	resourceID, err := strconv.ParseUint(resourceIDStr, 10, 32)
	if err != nil {
		utils.BadRequestResponse(c, "无效的资源ID")
		return
	}

	// 获取当前用户ID（可能未登录）
	userID, _ := utils.GetUserIDFromContext(c)

	ctx := c.Request.Context()
	resource, err := h.resourceRepo.GetResourceByID(ctx, uint(resourceID), userID)
	if err != nil {
		h.logger.Warn("获取资源详情失败", "resourceID", resourceID, "error", err.Error())
		utils.ErrorResponse(c, 404, "资源不存在")
		return
	}

	// 使用Worker Pool异步增加浏览次数（避免goroutine泄漏）
	taskID := fmt.Sprintf("incr_resource_view_%d", resourceID)
	err = utils.SubmitTask(taskID, func(taskCtx context.Context) error {
		return h.resourceRepo.IncrementViewCount(taskCtx, uint(resourceID))
	}, time.Duration(h.config.AsyncTasks.ResourceViewCountTimeout)*time.Second)

	if err != nil {
		h.logger.Debug("提交浏览次数更新任务失败", "resourceID", resourceID, "error", err.Error())
	}

	h.logger.Info("获取资源详情成功", "resourceID", resourceID)
	utils.SuccessResponse(c, 200, "获取成功", resource)
}

// GetResourceList 获取资源列表
func (h *ResourceHandler) GetResourceList(c *gin.Context) {
	var query models.ResourceListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.logger.Warn("获取资源列表参数错误", "error", err.Error())
		utils.ValidationErrorResponse(c, "请求参数错误")
		return
	}

	ctx := c.Request.Context()
	response, err := h.resourceRepo.ListResources(ctx, query)
	if err != nil {
		h.logger.Error("获取资源列表失败", "error", err.Error())
		utils.ErrorResponse(c, 500, "获取资源列表失败")
		return
	}

	h.logger.Info("获取资源列表成功", "total", response.Total)
	utils.SuccessResponse(c, 200, "获取成功", response)
}

// ToggleResourceLike 切换资源点赞
func (h *ResourceHandler) ToggleResourceLike(c *gin.Context) {
	userID, isOK := getUserIDOrFail(c)
	if !isOK {
		return
	}

	resourceIDStr := c.Param("id")
	resourceID, err := strconv.ParseUint(resourceIDStr, 10, 32)
	if err != nil {
		utils.BadRequestResponse(c, "无效的资源ID")
		return
	}

	ctx := c.Request.Context()
	isLiked, err := h.resourceRepo.ToggleResourceLike(ctx, uint(resourceID), userID)
	if err != nil {
		h.logger.Error("切换点赞失败", "resourceID", resourceID, "error", err.Error())
		utils.ErrorResponse(c, 500, "操作失败")
		return
	}

	utils.SuccessResponse(c, 200, "操作成功", gin.H{
		"is_liked": isLiked,
	})
}

// DeleteResource 删除资源
func (h *ResourceHandler) DeleteResource(c *gin.Context) {
	userID, isOK := getUserIDOrFail(c)
	if !isOK {
		return
	}

	resourceIDStr := c.Param("id")
	resourceID, err := strconv.ParseUint(resourceIDStr, 10, 32)
	if err != nil {
		utils.BadRequestResponse(c, "无效的资源ID")
		return
	}

	ctx := c.Request.Context()
	err = h.resourceRepo.DeleteResource(ctx, uint(resourceID), userID)
	if err != nil {
		h.logger.Error("删除资源失败", "resourceID", resourceID, "error", err.Error())
		utils.ErrorResponse(c, 500, "删除资源失败")
		return
	}

	h.logger.Info("删除资源成功", "resourceID", resourceID)
	utils.SuccessResponse(c, 200, "删除成功", nil)
}

// DownloadResource 下载资源（返回直接下载链接）
func (h *ResourceHandler) DownloadResource(c *gin.Context) {
	resourceIDStr := c.Param("id")
	resourceID, err := strconv.ParseUint(resourceIDStr, 10, 32)
	if err != nil {
		utils.BadRequestResponse(c, "无效的资源ID")
		return
	}

	ctx := c.Request.Context()
	resource, err := h.resourceRepo.GetResourceByID(ctx, uint(resourceID), 0)
	if err != nil {
		utils.ErrorResponse(c, 404, "资源不存在")
		return
	}

	// Increment download count asynchronously using Worker Pool
	taskID := fmt.Sprintf("incr_download_%d", resourceID)
	_ = utils.SubmitTask(taskID, func(taskCtx context.Context) error {
		return h.resourceRepo.IncrementDownloadCount(taskCtx, uint(resourceID))
	}, time.Duration(h.config.AsyncTasks.ResourceDownloadCountTimeout)*time.Second)

	// Return download URL for client to download directly from MinIO
	// 直接返回下载链接比代理更高效
	utils.SuccessResponse(c, 200, "获取下载链接成功", gin.H{
		"download_url": resource.StoragePath,
		"file_name":    resource.FileName,
		"file_size":    resource.FileSize,
	})
}

// ProxyDownloadResource 代理下载资源（支持Range请求和大文件流式传输）
func (h *ResourceHandler) ProxyDownloadResource(c *gin.Context) {
	resourceIDStr := c.Param("id")
	resourceID, err := strconv.ParseUint(resourceIDStr, 10, 32)
	if err != nil {
		utils.BadRequestResponse(c, "无效的资源ID")
		return
	}

	ctx := c.Request.Context()
	resource, err := h.resourceRepo.GetResourceByID(ctx, uint(resourceID), 0)
	if err != nil {
		utils.ErrorResponse(c, 404, "资源不存在")
		return
	}

	if h.resourceStorage == nil {
		utils.ErrorResponse(c, 500, "资源存储服务未配置")
		return
	}

	// 性能优化：使用net/url解析URL，避免多次字符串操作
	objectPath := resource.StoragePath
	bucketName := ""

	// 尝试解析URL提取bucket和object（仅当路径是完整URL时）
	if strings.HasPrefix(objectPath, "http://") || strings.HasPrefix(objectPath, "https://") {
		if parsedURL, err := url.Parse(objectPath); err == nil && parsedURL.Path != "" {
			// 提取路径部分: /bucket-name/object/path
			pathParts := strings.SplitN(strings.TrimPrefix(parsedURL.Path, "/"), "/", 2)
			if len(pathParts) >= 2 {
				bucketName = pathParts[0]
				objectPath = pathParts[1]
			}
		}
	}

	// 仅在DEBUG模式记录详细日志
	if h.logger != nil {
		h.logger.Debug("解析存储路径", "桶", bucketName, "对象路径", objectPath)
	}

	// 获取对象信息（如果解析出了bucket，从指定bucket读取；否则使用默认bucket）
	var objInfo minio.ObjectInfo
	if bucketName != "" {
		objInfo, err = h.resourceStorage.StatObjectFromBucket(ctx, bucketName, objectPath)
	} else {
		objInfo, err = h.resourceStorage.StatObject(ctx, objectPath)
	}
	if err != nil {
		h.logger.Error("获取资源对象信息失败", "resourceID", resourceID, "bucket", bucketName, "path", objectPath, "error", err.Error())
		utils.ErrorResponse(c, 404, "资源文件不存在")
		return
	}

	// 解析Range请求头
	rangeHeader := c.GetHeader("Range")
	var start, end int64
	var isRangeRequest bool

	if rangeHeader != "" {
		// 解析 "bytes=start-end" 或 "bytes=start-"
		var rangeStart, rangeEnd int64 = 0, objInfo.Size - 1
		if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &rangeStart, &rangeEnd); err != nil {
			// 尝试解析 "bytes=start-" 格式
			if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-", &rangeStart); err == nil {
				rangeEnd = objInfo.Size - 1
			}
		}
		start, end = rangeStart, rangeEnd
		isRangeRequest = true
		h.logger.Debug("Range请求", "range", rangeHeader, "start", start, "end", end)
	} else {
		start, end = 0, objInfo.Size-1
	}

	// 验证范围
	if start < 0 || end >= objInfo.Size || start > end {
		c.Header("Content-Range", fmt.Sprintf("bytes */%d", objInfo.Size))
		c.Status(416) // Range Not Satisfiable
		return
	}

	// 设置GetObject选项（支持Range）
	opts := minio.GetObjectOptions{}
	if isRangeRequest {
		opts.SetRange(start, end)
	}

	// 从MinIO获取对象（使用解析出的bucket）
	var object io.ReadCloser
	if bucketName != "" {
		object, err = h.resourceStorage.GetObjectFromBucket(ctx, bucketName, objectPath, opts)
	} else {
		object, err = h.resourceStorage.GetObject(ctx, objectPath, opts)
	}
	if err != nil {
		h.logger.Error("获取资源对象失败", "resourceID", resourceID, "bucket", bucketName, "error", err.Error())
		utils.ErrorResponse(c, 500, "读取文件失败")
		return
	}
	defer object.Close()

	// 设置响应头
	contentLength := end - start + 1
	c.Header("Content-Type", resource.FileType)
	c.Header("Content-Disposition", utils.EncodeFileName(resource.FileName))
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Length", fmt.Sprintf("%d", contentLength))
	c.Header("Content-Encoding", "identity") // 避免压缩中间件干扰
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	if isRangeRequest {
		c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, objInfo.Size))
		c.Status(206) // Partial Content
	} else {
		c.Status(200)
	}

	// 流式传输文件内容（避免大文件占用内存）
	written, err := io.Copy(c.Writer, object)
	if err != nil {
		h.logger.Error("传输文件失败", "resourceID", resourceID, "written", written, "error", err.Error())
		return
	}

	// 异步增加下载次数
	taskID := fmt.Sprintf("incr_download_%d", resourceID)
	_ = utils.SubmitTask(taskID, func(taskCtx context.Context) error {
		return h.resourceRepo.IncrementDownloadCount(taskCtx, uint(resourceID))
	}, time.Duration(h.config.AsyncTasks.ResourceDownloadCountTimeout)*time.Second)

	// 性能优化：仅在DEBUG模式记录详细信息
	h.logger.Debug("代理下载成功", "resourceID", resourceID, "fileName", resource.FileName, "size", written, "range", isRangeRequest)
}

// GetCategories 获取所有分类
func (h *ResourceHandler) GetCategories(c *gin.Context) {
	ctx := c.Request.Context()
	categories, err := h.resourceRepo.GetAllCategories(ctx)
	if err != nil {
		utils.ErrorResponse(c, 500, "获取分类失败")
		return
	}

	utils.SuccessResponse(c, 200, "获取成功", gin.H{
		"categories": categories,
	})
}

// ====== 资源评论相关处理器 ======

// CreateResourceComment 创建资源评论
func (h *ResourceHandler) CreateResourceComment(c *gin.Context) {
	userID, isOK := getUserIDOrFail(c)
	if !isOK {
		return
	}

	resourceIDStr := c.Param("id")
	resourceID, err := strconv.ParseUint(resourceIDStr, 10, 32)
	if err != nil {
		utils.BadRequestResponse(c, "无效的资源ID")
		return
	}

	var req models.CreateResourceCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("创建评论参数错误", "userID", userID, "error", err.Error())
		utils.ValidationErrorResponse(c, "请求参数错误")
		return
	}

	ctx := c.Request.Context()

	// 创建评论对象
	comment := &models.ResourceComment{
		ResourceID:    uint(resourceID),
		UserID:        userID,
		ParentID:      0,
		RootID:        0,
		ReplyToUserID: req.ReplyToUserID,
		Content:       req.Content,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// 如果是回复
	if req.ParentID != nil && *req.ParentID > 0 {
		comment.ParentID = *req.ParentID
		// 获取父评论的root_id
		comment.RootID = h.resourceCommentRepo.GetParentRootID(ctx, *req.ParentID)
	}

	err = h.resourceCommentRepo.CreateComment(ctx, comment)
	if err != nil {
		h.logger.Error("创建评论失败", "userID", userID, "error", err.Error())
		utils.ErrorResponse(c, 500, "创建评论失败")
		return
	}

	h.logger.Info("创建评论成功", "commentID", comment.ID, "userID", userID)

	// 获取用户信息用于 WebSocket 通知
	userInfo, err := GetUserWithProfile(ctx, h.userRepo, userID)
	if err != nil {
		h.logger.Warn("获取用户信息失败，无法发送 WebSocket 通知", "userID", userID, "error", err.Error())
	} else {
		commentUser := &models.CommentUser{
			ID:       userInfo.User.ID,
			Username: userInfo.User.Username,
			Nickname: userInfo.Nickname,
			Avatar:   userInfo.Avatar,
		}

		var replyToUser *models.CommentUser
		if req.ReplyToUserID != nil && *req.ReplyToUserID > 0 {
			replyInfo, err := GetUserWithProfile(ctx, h.userRepo, *req.ReplyToUserID)
			if err != nil {
				h.logger.Warn("获取回复用户信息失败", "replyToUserID", *req.ReplyToUserID, "error", err.Error())
			} else if replyInfo != nil {
				replyToUser = &models.CommentUser{
					ID:       replyInfo.User.ID,
					Username: replyInfo.User.Username,
					Nickname: replyInfo.Nickname,
					Avatar:   replyInfo.Avatar,
				}
			}
		}

		NotifyResourceComment(comment, commentUser, replyToUser)
	}

	utils.SuccessResponse(c, 201, "评论成功", gin.H{
		"comment_id": comment.ID,
	})
}

// GetResourceComments 获取资源评论列表
func (h *ResourceHandler) GetResourceComments(c *gin.Context) {
	resourceIDStr := c.Param("id")
	resourceID, err := strconv.ParseUint(resourceIDStr, 10, 32)
	if err != nil {
		utils.BadRequestResponse(c, "无效的资源ID")
		return
	}

	// 获取当前用户ID（可能未登录）
	userID, _ := utils.GetUserIDFromContext(c)

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(h.config.Pagination.DefaultPageSize)))

	ctx := c.Request.Context()
	response, err := h.resourceCommentRepo.GetCommentsByResourceID(ctx, uint(resourceID), userID, page, pageSize)
	if err != nil {
		h.logger.Error("获取评论列表失败", "resourceID", resourceID, "error", err.Error())
		utils.ErrorResponse(c, 500, "获取评论失败")
		return
	}

	h.logger.Info("获取评论列表成功", "resourceID", resourceID, "total", response.Total)
	utils.SuccessResponse(c, 200, "获取成功", response)
}

// ToggleResourceCommentLike 切换资源评论点赞
func (h *ResourceHandler) ToggleResourceCommentLike(c *gin.Context) {
	userID, isOK := getUserIDOrFail(c)
	if !isOK {
		return
	}

	commentIDStr := c.Param("id")
	commentID, err := strconv.ParseUint(commentIDStr, 10, 32)
	if err != nil {
		utils.BadRequestResponse(c, "无效的评论ID")
		return
	}

	ctx := c.Request.Context()
	isLiked, err := h.resourceCommentRepo.ToggleCommentLike(ctx, uint(commentID), userID)
	if err != nil {
		h.logger.Error("切换评论点赞失败", "commentID", commentID, "error", err.Error())
		utils.ErrorResponse(c, 500, "操作失败")
		return
	}

	utils.SuccessResponse(c, 200, "操作成功", gin.H{
		"is_liked": isLiked,
	})
}
