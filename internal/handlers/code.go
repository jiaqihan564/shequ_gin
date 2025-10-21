package handlers

import (
	"fmt"
	"gin/internal/models"
	"gin/internal/services"
	"gin/internal/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CodeHandler 代码处理器
type CodeHandler struct {
	repo     services.CodeRepository
	executor services.CodeExecutor
}

// NewCodeHandler 创建新的代码处理器
func NewCodeHandler(repo services.CodeRepository, executor services.CodeExecutor) *CodeHandler {
	return &CodeHandler{
		repo:     repo,
		executor: executor,
	}
}

// ExecuteCode 执行代码
func (h *CodeHandler) ExecuteCode(c *gin.Context) {
	var req models.ExecuteCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "请求参数无效: "+err.Error())
		return
	}

	// 获取用户ID
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, "未授权")
		return
	}

	// 执行代码
	result, err := h.executor.Execute(c.Request.Context(), req.Language, req.Code, req.Stdin)
	if err != nil {
		utils.GetLogger().Error("执行代码失败", "error", err, "user_id", userID)
		utils.InternalServerErrorResponse(c, "执行代码失败: "+err.Error())
		return
	}

	// 保存执行记录
	execution := &models.CodeExecution{
		UserID:        userID,
		Language:      req.Language,
		Code:          req.Code,
		Stdin:         req.Stdin,
		Output:        result.Output,
		Error:         result.Error,
		ExecutionTime: &result.ExecutionTime,
		Status:        result.Status,
	}

	if result.MemoryUsage > 0 {
		execution.MemoryUsage = &result.MemoryUsage
	}

	if err := h.repo.CreateExecution(execution); err != nil {
		utils.GetLogger().Error("保存执行记录失败", "error", err, "user_id", userID)
		// 不影响返回结果，只记录错误
	}

	// 如果请求中包含保存标题，则保存代码片段
	if req.SaveAs != "" {
		snippet := &models.CodeSnippet{
			UserID:      userID,
			Title:       req.SaveAs,
			Language:    req.Language,
			Code:        req.Code,
			Description: "",
			IsPublic:    false,
		}
		if err := h.repo.CreateSnippet(snippet); err != nil {
			utils.GetLogger().Error("保存代码片段失败", "error", err, "user_id", userID)
		} else {
			result.SnippetID = &snippet.ID
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "执行成功", result)
}

// CreateSnippet 创建代码片段
func (h *CodeHandler) CreateSnippet(c *gin.Context) {
	var req models.SaveSnippetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "请求参数无效: "+err.Error())
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, "未授权")
		return
	}

	snippet := &models.CodeSnippet{
		UserID:      userID,
		Title:       req.Title,
		Language:    req.Language,
		Code:        req.Code,
		Description: req.Description,
		IsPublic:    req.IsPublic,
	}

	if err := h.repo.CreateSnippet(snippet); err != nil {
		utils.GetLogger().Error("创建代码片段失败", "error", err, "user_id", userID)
		utils.InternalServerErrorResponse(c, "创建代码片段失败")
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "创建成功", snippet)
}

// GetSnippets 获取代码片段列表
func (h *CodeHandler) GetSnippets(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, "未授权")
		return
	}

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	snippets, total, err := h.repo.GetSnippetsByUserID(userID, pageSize, offset)
	if err != nil {
		utils.GetLogger().Error("获取代码片段列表失败", "error", err, "user_id", userID)
		utils.InternalServerErrorResponse(c, "获取代码片段列表失败")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "获取成功", gin.H{
		"items":     snippets,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetSnippetByID 获取代码片段详情
func (h *CodeHandler) GetSnippetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequestResponse(c, "无效的ID")
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, "未授权")
		return
	}

	snippet, err := h.repo.GetSnippetByID(uint(id))
	if err != nil {
		utils.NotFoundResponse(c, "代码片段不存在")
		return
	}

	// 检查权限（只有创建者或公开的代码片段可以访问）
	if snippet.UserID != userID && !snippet.IsPublic {
		utils.ForbiddenResponse(c, "无权访问此代码片段")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "获取成功", snippet)
}

// UpdateSnippet 更新代码片段
func (h *CodeHandler) UpdateSnippet(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequestResponse(c, "无效的ID")
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, "未授权")
		return
	}

	var req models.UpdateSnippetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "请求参数无效: "+err.Error())
		return
	}

	// 获取原有的代码片段
	snippet, err := h.repo.GetSnippetByID(uint(id))
	if err != nil {
		utils.NotFoundResponse(c, "代码片段不存在")
		return
	}

	// 检查权限
	if snippet.UserID != userID {
		utils.ForbiddenResponse(c, "无权修改此代码片段")
		return
	}

	// 更新字段
	if req.Title != "" {
		snippet.Title = req.Title
	}
	if req.Code != "" {
		snippet.Code = req.Code
	}
	if req.Description != "" {
		snippet.Description = req.Description
	}
	if req.IsPublic != nil {
		snippet.IsPublic = *req.IsPublic
	}

	if err := h.repo.UpdateSnippet(snippet); err != nil {
		utils.GetLogger().Error("更新代码片段失败", "error", err, "snippet_id", id)
		utils.InternalServerErrorResponse(c, "更新代码片段失败")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "更新成功", snippet)
}

// DeleteSnippet 删除代码片段
func (h *CodeHandler) DeleteSnippet(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequestResponse(c, "无效的ID")
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, "未授权")
		return
	}

	if err := h.repo.DeleteSnippet(uint(id), userID); err != nil {
		utils.GetLogger().Error("删除代码片段失败", "error", err, "snippet_id", id)
		utils.InternalServerErrorResponse(c, "删除代码片段失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "删除成功", nil)
}

// GetExecutions 获取执行记录列表
func (h *CodeHandler) GetExecutions(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, "未授权")
		return
	}

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	executions, total, err := h.repo.GetExecutionsByUserID(userID, pageSize, offset)
	if err != nil {
		utils.GetLogger().Error("获取执行记录失败", "error", err, "user_id", userID)
		utils.InternalServerErrorResponse(c, "获取执行记录失败")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "获取成功", gin.H{
		"items":     executions,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetSharedSnippet 通过分享令牌获取代码片段
func (h *CodeHandler) GetSharedSnippet(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		utils.BadRequestResponse(c, "缺少分享令牌")
		return
	}

	snippet, err := h.repo.GetSnippetByShareToken(token)
	if err != nil {
		utils.NotFoundResponse(c, "分享链接无效或已过期")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "获取成功", snippet)
}

// GenerateShareLink 生成分享链接
func (h *CodeHandler) GenerateShareLink(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequestResponse(c, "无效的ID")
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, "未授权")
		return
	}

	token, err := h.repo.GenerateShareToken(uint(id), userID)
	if err != nil {
		utils.GetLogger().Error("生成分享令牌失败", "error", err, "snippet_id", id)
		utils.InternalServerErrorResponse(c, "生成分享链接失败: "+err.Error())
		return
	}

	// 构建完整的分享链接
	shareURL := fmt.Sprintf("/code-share/%s", token)

	response := models.ShareSnippetResponse{
		ShareToken: token,
		ShareURL:   shareURL,
	}

	utils.SuccessResponse(c, http.StatusOK, "生成成功", response)
}

// GetLanguages 获取支持的语言列表
func (h *CodeHandler) GetLanguages(c *gin.Context) {
	languages := h.executor.GetSupportedLanguages()
	utils.SuccessResponse(c, http.StatusOK, "获取成功", languages)
}

