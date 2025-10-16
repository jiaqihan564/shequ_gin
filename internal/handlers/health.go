package handlers

import (
	"net/http"

	"gin/internal/services"

	"github.com/gin-gonic/gin"
)

// HealthHandler 健康检查处理器
type HealthHandler struct {
	db *services.Database
}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler(db *services.Database) *HealthHandler {
	return &HealthHandler{db: db}
}

// Check 健康检查
func (h *HealthHandler) Check(c *gin.Context) {
	if err := h.db.HealthCheck(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// Ready 就绪检查
func (h *HealthHandler) Ready(c *gin.Context) {
	if err := h.db.HealthCheck(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// Live 存活检查
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "live"})
}
