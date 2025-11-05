package handlers

import (
	"time"

	"gin/internal/models"
	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// CumulativeStatsHandler 累计统计处理器
type CumulativeStatsHandler struct {
	cumulativeRepo *services.CumulativeStatsRepository
	logger         utils.Logger
}

// NewCumulativeStatsHandler 创建累计统计处理器
func NewCumulativeStatsHandler(cumulativeRepo *services.CumulativeStatsRepository) *CumulativeStatsHandler {
	return &CumulativeStatsHandler{
		cumulativeRepo: cumulativeRepo,
		logger:         utils.GetLogger(),
	}
}

// GetCumulativeStats 获取累计统计数据
func (h *CumulativeStatsHandler) GetCumulativeStats(c *gin.Context) {
	data, err := h.cumulativeRepo.GetAllCumulativeStats()
	if err != nil {
		h.logger.Error("获取累计统计失败", "error", err.Error())
		utils.ErrorResponse(c, 500, "获取累计统计失败")
		return
	}

	utils.SuccessResponse(c, 200, "获取成功", data)
}

// GetDailyMetrics 获取每日指标
func (h *CumulativeStatsHandler) GetDailyMetrics(c *gin.Context) {
	// 获取日期范围，默认最近30天（优化：缓存time.Now()调用）
	now := time.Now()
	endDate := c.DefaultQuery("end", now.Format("2006-01-02"))
	startDate := c.DefaultQuery("start", now.AddDate(0, 0, -30).Format("2006-01-02"))

	// 获取趋势数据
	trend, err := h.cumulativeRepo.GetDailyMetrics(startDate, endDate)
	if err != nil {
		h.logger.Error("获取每日指标趋势失败",
			"startDate", startDate,
			"endDate", endDate,
			"error", err.Error())
		utils.ErrorResponse(c, 500, "获取每日指标失败")
		return
	}

	// 获取今日数据
	today, err := h.cumulativeRepo.GetTodayDailyMetric()
	if err != nil {
		h.logger.Error("获取今日指标失败", "error", err.Error())
		utils.ErrorResponse(c, 500, "获取今日指标失败")
		return
	}

	// 如果trend是空的，至少包含今天的数据
	if len(trend) == 0 && today != nil {
		trend = []models.DailyMetrics{*today}
	}

	// 计算汇总统计
	summary := calculateDailySummary(trend)

	utils.SuccessResponse(c, 200, "获取成功", gin.H{
		"today":   today,
		"trend":   trend,
		"summary": summary,
	})
}

// GetRealtimeMetrics 获取实时指标
func (h *CumulativeStatsHandler) GetRealtimeMetrics(c *gin.Context) {
	// 从实时管理器获取最新数据
	realtimeMgr := services.GetRealtimeMetricsManager()

	// 优先使用WebSocket的精确在线人数（基于真实连接）
	onlineUsers := GetWebSocketOnlineCount()

	currentQPS := realtimeMgr.GetCurrentQPS()
	lastErrorTime := realtimeMgr.GetLastErrorTime()
	cpuPercent, memoryPercent := realtimeMgr.GetSystemMetrics()

	data := gin.H{
		"online_users":    onlineUsers,
		"current_qps":     currentQPS,
		"system_cpu":      cpuPercent,
		"system_memory":   memoryPercent,
		"service_status":  "running",
		"last_error_time": lastErrorTime,
	}

	utils.SuccessResponse(c, 200, "获取成功", data)
}

// GetWebSocketOnlineCount 获取WebSocket在线人数（精确统计）
func GetWebSocketOnlineCount() int {
	// 从websocket_chat.go中定义的全局hub获取在线人数
	if globalHub != nil {
		return globalHub.GetOnlineCount()
	}
	// 如果WebSocket hub未初始化，返回0
	return 0
}

// calculateDailySummary 计算每日指标汇总
func calculateDailySummary(metrics []models.DailyMetrics) models.DailyMetricsStats {
	if len(metrics) == 0 {
		return models.DailyMetricsStats{}
	}

	var totalActiveUsers, totalNewUsers int
	var totalResponseTime, totalSuccessRate float64
	var maxPeakConcurrent int

	for _, m := range metrics {
		totalActiveUsers += m.ActiveUsers
		totalResponseTime += m.AvgResponseTime
		totalSuccessRate += m.SuccessRate
		totalNewUsers += m.NewUsers
		if m.PeakConcurrent > maxPeakConcurrent {
			maxPeakConcurrent = m.PeakConcurrent
		}
	}

	count := float64(len(metrics))
	return models.DailyMetricsStats{
		AvgActiveUsers:    float64(totalActiveUsers) / count,
		AvgResponseTime:   totalResponseTime / count,
		AvgSuccessRate:    totalSuccessRate / count,
		MaxPeakConcurrent: maxPeakConcurrent,
		TotalNewUsers:     totalNewUsers,
	}
}
