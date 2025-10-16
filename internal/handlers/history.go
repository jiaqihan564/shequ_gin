package handlers

import (
	"strconv"

	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// HistoryHandler 历史记录处理器
type HistoryHandler struct {
	historyRepo *services.HistoryRepository
	logger      utils.Logger
}

// NewHistoryHandler 创建历史记录处理器
func NewHistoryHandler(historyRepo *services.HistoryRepository) *HistoryHandler {
	return &HistoryHandler{
		historyRepo: historyRepo,
		logger:      utils.GetLogger(),
	}
}

// GetLoginHistory 获取登录历史
func (h *HistoryHandler) GetLoginHistory(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	// 获取查询参数
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	history, err := h.historyRepo.GetLoginHistory(userID, limit)
	if err != nil {
		h.logger.Error("获取登录历史失败",
			"userID", userID,
			"error", err.Error())
		utils.ErrorResponse(c, 500, "获取登录历史失败")
		return
	}

	utils.SuccessResponse(c, 200, "获取成功", history)
}

// GetOperationHistory 获取操作历史
func (h *HistoryHandler) GetOperationHistory(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	// 获取查询参数
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	history, err := h.historyRepo.GetOperationHistory(userID, limit)
	if err != nil {
		h.logger.Error("获取操作历史失败",
			"userID", userID,
			"error", err.Error())
		utils.ErrorResponse(c, 500, "获取操作历史失败")
		return
	}

	utils.SuccessResponse(c, 200, "获取成功", history)
}

// GetProfileChangeHistory 获取资料修改历史
func (h *HistoryHandler) GetProfileChangeHistory(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	// 获取查询参数
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	history, err := h.historyRepo.GetProfileChangeHistory(userID, limit)
	if err != nil {
		h.logger.Error("获取资料修改历史失败",
			"userID", userID,
			"error", err.Error())
		utils.ErrorResponse(c, 500, "获取资料修改历史失败")
		return
	}

	utils.SuccessResponse(c, 200, "获取成功", history)
}

// GetLocationDistribution 获取用户地区分布
func (h *HistoryHandler) GetLocationDistribution(c *gin.Context) {
	data, err := h.historyRepo.GetLocationDistribution()
	if err != nil {
		h.logger.Error("获取地区分布失败", "error", err.Error())
		utils.ErrorResponse(c, 500, "获取地区分布失败")
		return
	}

	utils.SuccessResponse(c, 200, "获取成功", data)
}
