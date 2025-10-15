package handlers

import (
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
	if err := h.db.Ping(); err != nil {
		c.JSON(503, gin.H{"status": "unhealthy", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "healthy"})
}

// Ready 就绪检查
func (h *HealthHandler) Ready(c *gin.Context) {
	if err := h.db.Ping(); err != nil {
		c.JSON(503, gin.H{"status": "not ready", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "ready"})
}

// Live 存活检查
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(200, gin.H{"status": "alive"})
}
