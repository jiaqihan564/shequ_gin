package utils

import (
	"net/http"

	"gin/internal/models"

	"github.com/gin-gonic/gin"
)

// SuccessResponse 成功响应
func SuccessResponse(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(code, models.CommonResponse{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// ErrorResponse 错误响应
func ErrorResponse(c *gin.Context, code int, message string) {
	c.JSON(code, models.CommonResponse{
		Code:    code,
		Message: message,
	})
}

// BadRequestResponse 400错误响应
func BadRequestResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusBadRequest, message)
}

// UnauthorizedResponse 401错误响应
func UnauthorizedResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusUnauthorized, message)
}

// NotFoundResponse 404错误响应
func NotFoundResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusNotFound, message)
}

// ConflictResponse 409错误响应
func ConflictResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusConflict, message)
}

// InternalServerErrorResponse 500错误响应
func InternalServerErrorResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusInternalServerError, message)
}
