package handlers

import (
	"net/http"
	"strconv"

	"gin/internal/models"
	"gin/internal/services"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户处理器
type UserHandler struct {
	userService *services.UserService
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetProfile 获取用户信息
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.CommonResponse{
			Code:    401,
			Message: "用户未认证",
		})
		return
	}

	// 调用服务层获取用户信息
	user, err := h.userService.GetUserByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, models.CommonResponse{
			Code:    404,
			Message: "用户不存在",
		})
		return
	}

	c.JSON(http.StatusOK, models.CommonResponse{
		Code:    200,
		Message: "获取用户信息成功",
		Data:    user,
	})
}

// UpdateProfile 更新用户信息
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.CommonResponse{
			Code:    401,
			Message: "用户未认证",
		})
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.CommonResponse{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 调用服务层更新用户信息
	user, err := h.userService.UpdateUser(userID.(uint), req.Email)
	if err != nil {
		c.JSON(http.StatusNotFound, models.CommonResponse{
			Code:    404,
			Message: "用户不存在",
		})
		return
	}

	c.JSON(http.StatusOK, models.CommonResponse{
		Code:    200,
		Message: "用户信息更新成功",
		Data:    user,
	})
}

// GetUserByID 根据ID获取用户信息（管理员功能）
func (h *UserHandler) GetUserByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.CommonResponse{
			Code:    400,
			Message: "无效的用户ID",
		})
		return
	}

	user, err := h.userService.GetUserByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, models.CommonResponse{
			Code:    404,
			Message: "用户不存在",
		})
		return
	}

	c.JSON(http.StatusOK, models.CommonResponse{
		Code:    200,
		Message: "获取用户信息成功",
		Data:    user,
	})
}
