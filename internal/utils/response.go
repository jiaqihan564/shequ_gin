package utils

import (
	"net/http"
	"strconv"

	"gin/internal/models"

	"github.com/gin-gonic/gin"
)

// ResponseHandler 响应处理器
type ResponseHandler struct{}

// SuccessResponse 成功响应
func (rh *ResponseHandler) SuccessResponse(c *gin.Context, code int, message string, data interface{}) {
	response := models.CommonResponse{
		Code:    code,
		Message: message,
		Data:    data,
	}
	c.JSON(code, response)
}

// ErrorResponse 错误响应
func (rh *ResponseHandler) ErrorResponse(c *gin.Context, code int, message string) {
	response := models.CommonResponse{
		Code:    code,
		Message: message,
	}
	c.JSON(code, response)
}

// BadRequestResponse 400错误响应
func (rh *ResponseHandler) BadRequestResponse(c *gin.Context, message string) {
	rh.ErrorResponse(c, http.StatusBadRequest, message)
}

// UnauthorizedResponse 401错误响应
func (rh *ResponseHandler) UnauthorizedResponse(c *gin.Context, message string) {
	rh.ErrorResponse(c, http.StatusUnauthorized, message)
}

// ForbiddenResponse 403错误响应
func (rh *ResponseHandler) ForbiddenResponse(c *gin.Context, message string) {
	rh.ErrorResponse(c, http.StatusForbidden, message)
}

// NotFoundResponse 404错误响应
func (rh *ResponseHandler) NotFoundResponse(c *gin.Context, message string) {
	rh.ErrorResponse(c, http.StatusNotFound, message)
}

// ConflictResponse 409错误响应
func (rh *ResponseHandler) ConflictResponse(c *gin.Context, message string) {
	rh.ErrorResponse(c, http.StatusConflict, message)
}

// InternalServerErrorResponse 500错误响应
func (rh *ResponseHandler) InternalServerErrorResponse(c *gin.Context, message string) {
	rh.ErrorResponse(c, http.StatusInternalServerError, message)
}

// ValidationErrorResponse 验证错误响应
func (rh *ResponseHandler) ValidationErrorResponse(c *gin.Context, message string) {
	rh.ErrorResponse(c, http.StatusUnprocessableEntity, message)
}

// TooManyRequestsResponse 429错误响应
func (rh *ResponseHandler) TooManyRequestsResponse(c *gin.Context, message string) {
	rh.ErrorResponse(c, http.StatusTooManyRequests, message)
}

// 全局响应处理器实例
var globalResponseHandler *ResponseHandler

// InitResponseHandler 初始化全局响应处理器
func InitResponseHandler() {
	globalResponseHandler = &ResponseHandler{}
}

// GetResponseHandler 获取全局响应处理器
func GetResponseHandler() *ResponseHandler {
	if globalResponseHandler == nil {
		globalResponseHandler = &ResponseHandler{}
	}
	return globalResponseHandler
}

// 便捷函数
func SuccessResponse(c *gin.Context, code int, message string, data interface{}) {
	GetResponseHandler().SuccessResponse(c, code, message, data)
}

func ErrorResponse(c *gin.Context, code int, message string) {
	GetResponseHandler().ErrorResponse(c, code, message)
}

func BadRequestResponse(c *gin.Context, message string) {
	GetResponseHandler().BadRequestResponse(c, message)
}

func UnauthorizedResponse(c *gin.Context, message string) {
	GetResponseHandler().UnauthorizedResponse(c, message)
}

func ForbiddenResponse(c *gin.Context, message string) {
	GetResponseHandler().ForbiddenResponse(c, message)
}

func NotFoundResponse(c *gin.Context, message string) {
	GetResponseHandler().NotFoundResponse(c, message)
}

func ConflictResponse(c *gin.Context, message string) {
	GetResponseHandler().ConflictResponse(c, message)
}

func InternalServerErrorResponse(c *gin.Context, message string) {
	GetResponseHandler().InternalServerErrorResponse(c, message)
}

func ValidationErrorResponse(c *gin.Context, message string) {
	GetResponseHandler().ValidationErrorResponse(c, message)
}

func TooManyRequestsResponse(c *gin.Context, message string) {
	GetResponseHandler().TooManyRequestsResponse(c, message)
}

// ParseUintParam 解析URL参数中的uint类型
func ParseUintParam(c *gin.Context, param string) (uint, error) {
	idStr := c.Param(param)
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

// GetUserIDFromContext 从上下文中获取用户ID
func GetUserIDFromContext(c *gin.Context) (uint, error) {
	userID, exists := c.Get("userID")
	if !exists {
		return 0, ErrUserNotAuthenticated
	}

	// 处理字符串类型的用户ID
	if idStr, ok := userID.(string); ok {
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			return 0, ErrInvalidUserID
		}
		return uint(id), nil
	}

	// 处理uint类型的用户ID
	if id, ok := userID.(uint); ok {
		return id, nil
	}

	return 0, ErrInvalidUserID
}
