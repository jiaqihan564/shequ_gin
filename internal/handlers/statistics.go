package handlers

import (
	"strconv"
	"time"

	"gin/internal/config"
	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// StatisticsHandler 统计处理器
type StatisticsHandler struct {
	statsRepo *services.StatisticsRepository
	logger    utils.Logger
	config    *config.Config
}

// NewStatisticsHandler 创建统计处理器
func NewStatisticsHandler(statsRepo *services.StatisticsRepository, cfg *config.Config) *StatisticsHandler {
	return &StatisticsHandler{
		statsRepo: statsRepo,
		logger:    utils.GetLogger(),
		config:    cfg,
	}
}

// GetOverview 获取总览数据
func (h *StatisticsHandler) GetOverview(c *gin.Context) {
	overview, err := h.statsRepo.GetTodayOverview()
	if err != nil {
		h.logger.Error("获取统计总览失败", "error", err.Error())
		utils.ErrorResponse(c, 500, "获取统计总览失败")
		return
	}

	utils.SuccessResponse(c, 200, "获取成功", overview)
}

// GetUserStatistics 获取用户统计
func (h *StatisticsHandler) GetUserStatistics(c *gin.Context) {
	// 获取日期范围参数，默认从配置读取天数（优化：缓存time.Now()调用）
	now := time.Now()
	dateFormat := h.config.DateTimeFormats.DateOnly
	defaultDays := h.config.StatisticsQueryExtended.DefaultDateRangeDays
	endDate := c.DefaultQuery("end", now.Format(dateFormat))
	startDate := c.DefaultQuery("start", now.AddDate(0, 0, -defaultDays).Format(dateFormat))

	stats, err := h.statsRepo.GetUserStatistics(startDate, endDate)
	if err != nil {
		h.logger.Error("获取用户统计失败",
			"startDate", startDate,
			"endDate", endDate,
			"error", err.Error())
		utils.ErrorResponse(c, 500, "获取用户统计失败")
		return
	}

	// 计算总计
	totalLogin := 0
	totalRegister := 0
	for _, stat := range stats {
		totalLogin += stat.LoginCount
		totalRegister += stat.RegisterCount
	}

	utils.SuccessResponse(c, 200, "获取成功", gin.H{
		"stats": stats,
		"total": gin.H{
			"total_login":    totalLogin,
			"total_register": totalRegister,
		},
	})
}

// GetApiStatistics 获取API统计
func (h *StatisticsHandler) GetApiStatistics(c *gin.Context) {
	// 获取日期范围参数，默认从配置读取天数（优化：缓存time.Now()调用）
	now := time.Now()
	dateFormat := h.config.DateTimeFormats.DateOnly
	defaultDays := h.config.StatisticsQueryExtended.DefaultDateRangeDays
	endDate := c.DefaultQuery("end", now.Format(dateFormat))
	startDate := c.DefaultQuery("start", now.AddDate(0, 0, -defaultDays).Format(dateFormat))

	stats, err := h.statsRepo.GetApiStatistics(startDate, endDate)
	if err != nil {
		h.logger.Error("获取API统计失败",
			"startDate", startDate,
			"endDate", endDate,
			"error", err.Error())
		utils.ErrorResponse(c, 500, "获取API统计失败")
		return
	}

	// 计算总计
	totalCalls := 0
	successCalls := 0
	errorCalls := 0
	totalLatency := 0.0

	for _, stat := range stats {
		totalCalls += stat.TotalCount
		successCalls += stat.SuccessCount
		errorCalls += stat.ErrorCount
		totalLatency += stat.AvgLatencyMs * float64(stat.TotalCount)
	}

	successRate := 0.0
	avgLatency := 0.0
	if totalCalls > 0 {
		successRate = float64(successCalls) * 100.0 / float64(totalCalls)
		avgLatency = totalLatency / float64(totalCalls)
	}

	utils.SuccessResponse(c, 200, "获取成功", gin.H{
		"stats": stats,
		"total": gin.H{
			"total_calls":  totalCalls,
			"success_rate": successRate,
			"avg_latency":  avgLatency,
		},
	})
}

// GetEndpointRanking 获取接口排行
func (h *StatisticsHandler) GetEndpointRanking(c *gin.Context) {
	// 获取日期范围参数，默认从配置读取天数
	now := time.Now()
	dateFormat := h.config.DateTimeFormats.DateOnly
	defaultDays := h.config.StatisticsQueryExtended.DefaultDateRangeDays
	endDate := c.DefaultQuery("end", now.Format(dateFormat))
	startDate := c.DefaultQuery("start", now.AddDate(0, 0, -defaultDays).Format(dateFormat))
	
	// 获取排序和数量参数
	sortBy := c.DefaultQuery("sort_by", "total_count") // 默认按调用次数排序
	order := c.DefaultQuery("order", "desc")           // 默认降序
	limitStr := c.Query("limit")                       // 不设置默认值，支持"全部"选项
	
	// 验证排序参数
	validSorts := map[string]bool{
		"total_count": true,
		"success_rate": true,
		"avg_latency_ms": true,
	}
	if !validSorts[sortBy] {
		sortBy = "total_count"
	}
	
	// 验证排序方向
	if order != "asc" && order != "desc" {
		order = "desc"
	}
	
	// 转换limit参数，如果未提供或为"all"则设为-1表示无限制
	limit := -1 // 默认无限制
	if limitStr != "" && limitStr != "all" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	rankings, err := h.statsRepo.GetEndpointRanking(startDate, endDate, sortBy, order, limit)
	if err != nil {
		h.logger.Error("获取接口排行失败",
			"startDate", startDate,
			"endDate", endDate,
			"sortBy", sortBy,
			"order", order,
			"limit", limit,
			"error", err.Error())
		utils.ErrorResponse(c, 500, "获取接口排行失败")
		return
	}

	utils.SuccessResponse(c, 200, "获取成功", rankings)
}
