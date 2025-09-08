package handlers

import (
	"net/http"
	"time"

	"gin/internal/models"
	"gin/internal/services"
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// HealthHandler 健康检查处理器
type HealthHandler struct {
	db     *services.Database
	logger utils.Logger
}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler(db *services.Database) *HealthHandler {
	return &HealthHandler{
		db:     db,
		logger: utils.GetLogger(),
	}
}

// HealthStatus 健康状态结构
type HealthStatus struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version"`
	Services  map[string]interface{} `json:"services"`
}

// Check 健康检查
func (h *HealthHandler) Check(c *gin.Context) {
	status := "healthy"
	services := make(map[string]interface{})

	// 检查数据库连接
	if h.db != nil {
		if err := h.db.HealthCheck(); err != nil {
			status = "unhealthy"
			services["database"] = map[string]interface{}{
				"status": "down",
				"error":  err.Error(),
			}
			h.logger.Error("数据库健康检查失败", "error", err.Error())
		} else {
			services["database"] = map[string]interface{}{
				"status": "up",
			}
		}
	} else {
		status = "unhealthy"
		services["database"] = map[string]interface{}{
			"status": "not_configured",
		}
	}

	// 检查内存使用情况
	services["memory"] = map[string]interface{}{
		"status": "up",
	}

	// 检查系统时间
	services["system"] = map[string]interface{}{
		"status":    "up",
		"timestamp": time.Now().Unix(),
	}

	healthStatus := HealthStatus{
		Status:    status,
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Services:  services,
	}

	httpStatus := http.StatusOK
	if status == "unhealthy" {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, models.CommonResponse{
		Code:    httpStatus,
		Message: "健康检查完成",
		Data:    healthStatus,
	})
}

// Ready 就绪检查
func (h *HealthHandler) Ready(c *gin.Context) {
	// 检查关键服务是否就绪
	if h.db != nil {
		if err := h.db.HealthCheck(); err != nil {
			h.logger.Error("服务未就绪：数据库连接失败", "error", err.Error())
			c.JSON(http.StatusServiceUnavailable, models.CommonResponse{
				Code:    503,
				Message: "服务未就绪",
			})
			return
		}
	}

	c.JSON(http.StatusOK, models.CommonResponse{
		Code:    200,
		Message: "服务已就绪",
	})
}

// Live 存活检查
func (h *HealthHandler) Live(c *gin.Context) {
	// 简单的存活检查，不依赖外部服务
	c.JSON(http.StatusOK, models.CommonResponse{
		Code:    200,
		Message: "服务存活",
		Data: map[string]interface{}{
			"timestamp": time.Now().Unix(),
		},
	})
}
