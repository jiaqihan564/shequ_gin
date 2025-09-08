package handlers

import (
	"net/http"

	"gin/internal/models"

	"github.com/gin-gonic/gin"
)

// HealthHandler 健康检查处理器
type HealthHandler struct{}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Check 健康检查
func (h *HealthHandler) Check(c *gin.Context) {
	c.JSON(http.StatusOK, models.CommonResponse{
		Code:    200,
		Message: "服务运行正常",
	})
}
