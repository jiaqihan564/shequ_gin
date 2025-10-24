package handlers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gin/internal/models"
	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// ResourceHandler 资源处理器
type ResourceHandler struct {
	resourceRepo        *services.ResourceRepository
	resourceCommentRepo *services.ResourceCommentRepository
	resourceStorage     *services.ResourceStorageService
	logger              utils.Logger
}

// NewResourceHandler 创建资源处理器
func NewResourceHandler(resourceRepo *services.ResourceRepository, resourceCommentRepo *services.ResourceCommentRepository, resourceStorage *services.ResourceStorageService) *ResourceHandler {
	return &ResourceHandler{
		resourceRepo:        resourceRepo,
		resourceCommentRepo: resourceCommentRepo,
		resourceStorage:     resourceStorage,
		logger:              utils.GetLogger(),
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
	}, 3*time.Second)

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

// DownloadResource 下载资源
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
	}, 3*time.Second)

	// Return download URL for client to download directly from MinIO
	// 直接返回下载链接比代理更高效
	utils.SuccessResponse(c, 200, "获取下载链接成功", gin.H{
		"download_url": resource.StoragePath,
		"file_name":    resource.FileName,
		"file_size":    resource.FileSize,
	})
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
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

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
