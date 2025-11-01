package handlers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gin/internal/config"
	"gin/internal/models"
	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// ArticleHandler 文章处理器
type ArticleHandler struct {
	articleRepo *services.ArticleRepository
	cacheSvc    *services.CacheService
	logger      utils.Logger
	config      *config.Config
}

// NewArticleHandler 创建文章处理器
func NewArticleHandler(articleRepo *services.ArticleRepository, cacheSvc *services.CacheService, cfg *config.Config) *ArticleHandler {
	return &ArticleHandler{
		articleRepo: articleRepo,
		cacheSvc:    cacheSvc,
		logger:      utils.GetLogger(),
		config:      cfg,
	}
}

// CreateArticle 创建文章
func (h *ArticleHandler) CreateArticle(c *gin.Context) {
	userID, isOK := getUserIDOrFail(c)
	if !isOK {
		return
	}

	var req models.CreateArticleRequest
	if !bindJSONOrFail(c, &req, h.logger, "CreateArticle") {
		return
	}

	// 处理标签（创建新标签或获取已有标签ID）
	ctx := c.Request.Context()
	var tagIDs []uint
	tagIDs = append(tagIDs, req.TagIDs...)

	for _, tagName := range req.TagNames {
		if tagName != "" {
			tagID, err := h.articleRepo.CreateOrGetTag(ctx, tagName)
			if err == nil {
				tagIDs = append(tagIDs, tagID)
			}
		}
	}

	// 创建文章
	article := &models.Article{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		Content:     req.Content,
		Status:      req.Status,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := h.articleRepo.CreateArticle(ctx, article, req.CodeBlocks, req.CategoryIDs, tagIDs)
	if err != nil {
		h.logger.Error("创建文章失败", "userID", userID, "error", err.Error())
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, "创建文章失败")
		return
	}

	h.logger.Info("创建文章成功", "articleID", article.ID, "userID", userID, "title", article.Title)
	utils.SuccessResponse(c, 201, "创建成功", gin.H{
		"article_id": article.ID,
	})
}

// GetArticleDetail 获取文章详情
func (h *ArticleHandler) GetArticleDetail(c *gin.Context) {
	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 32)
	if err != nil {
		utils.BadRequestResponse(c, "无效的文章ID")
		return
	}

	// 获取当前用户ID（可能未登录）
	userID, _ := utils.GetUserIDFromContext(c)

	ctx := c.Request.Context()
	article, err := h.articleRepo.GetArticleByID(ctx, uint(articleID), userID)
	if err != nil {
		h.logger.Warn("获取文章详情失败", "articleID", articleID, "error", err.Error())
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, "文章不存在")
		return
	}

	// 增加浏览次数（使用Worker Pool，避免无限制goroutine）
	taskID := fmt.Sprintf("incr_view_%d", articleID)
	err = utils.SubmitTask(taskID, func(taskCtx context.Context) error {
		return h.articleRepo.IncrementViewCount(taskCtx, uint(articleID))
	}, time.Duration(h.config.AsyncTasks.ArticleViewCountTimeout)*time.Second)

	if err != nil {
		h.logger.Debug("提交浏览次数更新任务失败", "articleID", articleID, "error", err.Error())
	}

	h.logger.Info("获取文章详情成功", "articleID", articleID)
	utils.SuccessResponse(c, 200, "获取成功", article)
}

// GetArticleList 获取文章列表
func (h *ArticleHandler) GetArticleList(c *gin.Context) {
	var query models.ArticleListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.logger.Warn("获取文章列表参数错误", "error", err.Error())
		utils.ValidationErrorResponse(c, "请求参数错误")
		return
	}

	ctx := c.Request.Context()
	response, err := h.articleRepo.ListArticles(ctx, query)
	if err != nil {
		h.logger.Error("获取文章列表失败", "error", err.Error())
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, "获取文章列表失败")
		return
	}

	h.logger.Info("获取文章列表成功", "total", response.Total, "page", query.Page)
	utils.SuccessResponse(c, 200, "获取成功", response)
}

// UpdateArticle 更新文章
func (h *ArticleHandler) UpdateArticle(c *gin.Context) {
	userID, isOK := getUserIDOrFail(c)
	if !isOK {
		return
	}

	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 32)
	if err != nil {
		utils.BadRequestResponse(c, "无效的文章ID")
		return
	}

	var req models.UpdateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("更新文章请求参数错误", "userID", userID, "articleID", articleID, "error", err.Error())
		utils.ValidationErrorResponse(c, "请求参数错误: "+err.Error())
		return
	}

	// 处理新标签
	if len(req.TagNames) > 0 {
		ctx := c.Request.Context()
		for _, tagName := range req.TagNames {
			if tagName != "" {
				tagID, err := h.articleRepo.CreateOrGetTag(ctx, tagName)
				if err == nil {
					req.TagIDs = append(req.TagIDs, tagID)
				}
			}
		}
	}

	ctx := c.Request.Context()
	err = h.articleRepo.UpdateArticle(ctx, uint(articleID), userID, req)
	if err != nil {
		h.logger.Error("更新文章失败", "articleID", articleID, "userID", userID, "error", err.Error())
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, "更新文章失败")
		return
	}

	h.logger.Info("更新文章成功", "articleID", articleID, "userID", userID)
	utils.SuccessResponse(c, 200, "更新成功", nil)
}

// DeleteArticle 删除文章
func (h *ArticleHandler) DeleteArticle(c *gin.Context) {
	userID, isOK := getUserIDOrFail(c)
	if !isOK {
		return
	}

	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 32)
	if err != nil {
		utils.BadRequestResponse(c, "无效的文章ID")
		return
	}

	ctx := c.Request.Context()
	err = h.articleRepo.DeleteArticle(ctx, uint(articleID), userID)
	if err != nil {
		h.logger.Error("删除文章失败", "articleID", articleID, "userID", userID, "error", err.Error())
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, "删除文章失败")
		return
	}

	h.logger.Info("删除文章成功", "articleID", articleID, "userID", userID)
	utils.SuccessResponse(c, 200, "删除成功", nil)
}

// ToggleArticleLike 切换文章点赞
func (h *ArticleHandler) ToggleArticleLike(c *gin.Context) {
	userID, isOK := getUserIDOrFail(c)
	if !isOK {
		return
	}

	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 32)
	if err != nil {
		utils.BadRequestResponse(c, "无效的文章ID")
		return
	}

	ctx := c.Request.Context()
	isLiked, err := h.articleRepo.ToggleArticleLike(ctx, uint(articleID), userID)
	if err != nil {
		h.logger.Error("切换文章点赞失败", "articleID", articleID, "userID", userID, "error", err.Error())
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, "操作失败")
		return
	}

	h.logger.Info("切换文章点赞成功", "articleID", articleID, "userID", userID, "isLiked", isLiked)
	utils.SuccessResponse(c, 200, "操作成功", gin.H{
		"is_liked": isLiked,
	})
}

// CreateComment 创建评论
func (h *ArticleHandler) CreateComment(c *gin.Context) {
	userID, isOK := getUserIDOrFail(c)
	if !isOK {
		return
	}

	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 32)
	if err != nil {
		utils.BadRequestResponse(c, "无效的文章ID")
		return
	}

	var req models.CreateCommentRequest
	if !bindJSONOrFail(c, &req, h.logger, "CreateComment") {
		return
	}

	comment := &models.ArticleComment{
		ArticleID:     uint(articleID),
		UserID:        userID,
		ParentID:      req.ParentID,
		RootID:        0, // 将在repository中自动计算
		ReplyToUserID: req.ReplyToUserID,
		Content:       req.Content,
		Status:        1,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	ctx := c.Request.Context()
	err = h.articleRepo.CreateComment(ctx, comment)
	if err != nil {
		h.logger.Error("创建评论失败", "articleID", articleID, "userID", userID, "error", err.Error())
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, "创建评论失败")
		return
	}

	h.logger.Info("创建评论成功", "commentID", comment.ID, "articleID", articleID, "userID", userID)
	utils.SuccessResponse(c, 201, "评论成功", gin.H{
		"comment_id": comment.ID,
	})
}

// GetComments 获取评论列表
func (h *ArticleHandler) GetComments(c *gin.Context) {
	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 32)
	if err != nil {
		utils.BadRequestResponse(c, "无效的文章ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(h.config.Pagination.DefaultPageSize)))

	// 获取当前用户ID（可能未登录）
	userID, _ := utils.GetUserIDFromContext(c)

	ctx := c.Request.Context()
	response, err := h.articleRepo.GetComments(ctx, uint(articleID), page, pageSize, userID)
	if err != nil {
		h.logger.Error("获取评论列表失败", "articleID", articleID, "error", err.Error())
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, "获取评论失败")
		return
	}

	h.logger.Info("获取评论列表成功", "articleID", articleID, "total", response.Total)
	utils.SuccessResponse(c, 200, "获取成功", response)
}

// ToggleCommentLike 切换评论点赞
func (h *ArticleHandler) ToggleCommentLike(c *gin.Context) {
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
	isLiked, err := h.articleRepo.ToggleCommentLike(ctx, uint(commentID), userID)
	if err != nil {
		h.logger.Error("切换评论点赞失败", "commentID", commentID, "userID", userID, "error", err.Error())
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, "操作失败")
		return
	}

	h.logger.Info("切换评论点赞成功", "commentID", commentID, "userID", userID, "isLiked", isLiked)
	utils.SuccessResponse(c, 200, "操作成功", gin.H{
		"is_liked": isLiked,
	})
}

// DeleteComment 删除评论
func (h *ArticleHandler) DeleteComment(c *gin.Context) {
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
	err = h.articleRepo.DeleteComment(ctx, uint(commentID), userID)
	if err != nil {
		h.logger.Error("删除评论失败", "commentID", commentID, "userID", userID, "error", err.Error())
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, "删除评论失败")
		return
	}

	h.logger.Info("删除评论成功", "commentID", commentID, "userID", userID)
	utils.SuccessResponse(c, 200, "删除成功", nil)
}

// CreateReport 创建举报
func (h *ArticleHandler) CreateReport(c *gin.Context) {
	userID, isOK := getUserIDOrFail(c)
	if !isOK {
		return
	}

	var req models.CreateReportRequest
	if !bindJSONOrFail(c, &req, h.logger, "CreateReport") {
		return
	}

	// 必须指定文章ID或评论ID中的一个
	if req.ArticleID == nil && req.CommentID == nil {
		utils.ValidationErrorResponse(c, "必须指定文章或评论")
		return
	}

	report := &models.ArticleReport{
		ArticleID: req.ArticleID,
		CommentID: req.CommentID,
		UserID:    userID,
		Reason:    req.Reason,
		Status:    0,
		CreatedAt: time.Now(),
	}

	ctx := c.Request.Context()
	err := h.articleRepo.CreateReport(ctx, report)
	if err != nil {
		h.logger.Error("创建举报失败", "userID", userID, "error", err.Error())
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, "创建举报失败")
		return
	}

	h.logger.Info("创建举报成功", "reportID", report.ID, "userID", userID)
	utils.SuccessResponse(c, 201, "举报成功，我们会尽快处理", nil)
}

// GetCategories 获取所有分类（带缓存）
func (h *ArticleHandler) GetCategories(c *gin.Context) {
	ctx := c.Request.Context()

	// 使用缓存服务获取分类
	categories, err := h.cacheSvc.GetArticleCategories(ctx)
	if err != nil {
		h.logger.Error("获取分类列表失败", "error", err.Error())
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, "获取分类失败")
		return
	}

	h.logger.Debug("获取分类列表成功（可能来自缓存）", "count", len(categories))
	utils.SuccessResponse(c, 200, "获取成功", gin.H{
		"categories": categories,
	})
}

// GetTags 获取所有标签（带缓存）
func (h *ArticleHandler) GetTags(c *gin.Context) {
	ctx := c.Request.Context()

	// 使用缓存服务获取标签
	tags, err := h.cacheSvc.GetArticleTags(ctx)
	if err != nil {
		h.logger.Error("获取标签列表失败", "error", err.Error())
		statusCode := utils.GetHTTPStatusCode(err)
		utils.ErrorResponse(c, statusCode, "获取标签失败")
		return
	}

	h.logger.Debug("获取标签列表成功（可能来自缓存）", "count", len(tags))
	utils.SuccessResponse(c, 200, "获取成功", gin.H{
		"tags": tags,
	})
}
