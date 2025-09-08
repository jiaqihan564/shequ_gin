package handlers

import (
	"net/http"

	"gin/internal/models"
	"gin/internal/services"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService *services.AuthService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Login 处理登录请求
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.CommonResponse{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 调用服务层进行登录验证
	response, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.CommonResponse{
			Code:    401,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// Register 处理用户注册请求
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.CommonResponse{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 调用服务层进行用户注册
	response, err := h.authService.Register(req.Username, req.Password, req.Email)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "用户名已存在" || err.Error() == "邮箱已被注册" {
			statusCode = http.StatusConflict
		}
		c.JSON(statusCode, models.CommonResponse{
			Code:    statusCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, response)
}
