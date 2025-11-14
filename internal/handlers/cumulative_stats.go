package handlers

import (
	"time"

	"gin/internal/models"
	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// CumulativeStatsHandler �ۼ�ͳ�ƴ�����
type CumulativeStatsHandler struct {
	cumulativeRepo *services.CumulativeStatsRepository
	logger         utils.Logger
}

// NewCumulativeStatsHandler �����ۼ�ͳ�ƴ�����
func NewCumulativeStatsHandler(cumulativeRepo *services.CumulativeStatsRepository) *CumulativeStatsHandler {
	return &CumulativeStatsHandler{
		cumulativeRepo: cumulativeRepo,
		logger:         utils.GetLogger(),
	}
}

// GetCumulativeStats ��ȡ�ۼ�ͳ������
func (h *CumulativeStatsHandler) GetCumulativeStats(c *gin.Context) {
	// ���ڼ���֮ǰ����̨ʵ�����ݸ���һ�εۼ�ͳ������
	if h.cumulativeRepo != nil {
		h.cumulativeRepo.RefreshFromSources()
	}

	data, err := h.cumulativeRepo.GetAllCumulativeStats()
	if err != nil {
		h.logger.Error("��ȡ�ۼ�ͳ��ʧ��", "error", err.Error())
		utils.ErrorResponse(c, 500, "��ȡ�ۼ�ͳ��ʧ��")
		return
	}

	utils.SuccessResponse(c, 200, "��ȡ�ɹ�", data)
}

// GetDailyMetrics ��ȡÿ��ָ��
func (h *CumulativeStatsHandler) GetDailyMetrics(c *gin.Context) {
	// ��ȡ���ڷ�Χ��Ĭ�����30�죨�Ż�������time.Now()���ã�
	now := time.Now()
	endDate := c.DefaultQuery("end", now.Format("2006-01-02"))
	startDate := c.DefaultQuery("start", now.AddDate(0, 0, -30).Format("2006-01-02"))

	// ��ȡ��������
	trend, err := h.cumulativeRepo.GetDailyMetrics(startDate, endDate)
	if err != nil {
		h.logger.Error("��ȡÿ��ָ������ʧ��",
			"startDate", startDate,
			"endDate", endDate,
			"error", err.Error())
		utils.ErrorResponse(c, 500, "��ȡÿ��ָ��ʧ��")
		return
	}

	// ��ȡ��������
	today, err := h.cumulativeRepo.GetTodayDailyMetric()
	if err != nil {
		h.logger.Error("��ȡ����ָ��ʧ��", "error", err.Error())
		utils.ErrorResponse(c, 500, "��ȡ����ָ��ʧ��")
		return
	}

	// ���trend�ǿյģ����ٰ������������
	if len(trend) == 0 && today != nil {
		trend = []models.DailyMetrics{*today}
	}

	// �������ͳ��
	summary := calculateDailySummary(trend)

	utils.SuccessResponse(c, 200, "��ȡ�ɹ�", gin.H{
		"today":   today,
		"trend":   trend,
		"summary": summary,
	})
}

// GetRealtimeMetrics ��ȡʵʱָ��
func (h *CumulativeStatsHandler) GetRealtimeMetrics(c *gin.Context) {
	// ��ʵʱ��������ȡ��������
	realtimeMgr := services.GetRealtimeMetricsManager()

	// ʹ��WebSocket������ͳ�������û���ÿ����¼�û����Ὠ��ȫ��WS���ӣ�
	onlineUsers := GetWebSocketOnlineCount()

	currentQPS := realtimeMgr.GetCurrentQPS()
	cpuPercent, memoryPercent := realtimeMgr.GetSystemMetrics()

	data := gin.H{
		"online_users":  onlineUsers,
		"current_qps":   currentQPS,
		"system_cpu":    cpuPercent,
		"system_memory": memoryPercent,
	}

	utils.SuccessResponse(c, 200, "��ȡ�ɹ�", data)
}

// GetWebSocketOnlineCount ��ȡWebSocket������������ȷͳ�ƣ�
func GetWebSocketOnlineCount() int {
	// ��websocket_chat.go�ж����ȫ��hub��ȡ��������
	if globalHub != nil {
		return globalHub.GetOnlineCount()
	}
	// ���WebSocket hubδ��ʼ��������0
	return 0
}

// calculateDailySummary ����ÿ��ָ�����
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

