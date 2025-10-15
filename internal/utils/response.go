package utils

import (
	"net/http"
	"strconv"

	"gin/internal/models"

	"github.com/gin-gonic/gin"
)

// SuccessResponse 成功响应
func SuccessResponse(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(code, models.CommonResponse{
		Code:      code,
		Message:   message,
		RequestID: getRequestID(c),
		Data:      data,
	})
}

// ErrorResponse 错误响应
func ErrorResponse(c *gin.Context, code int, message string) {
	c.JSON(code, models.CommonResponse{
		Code:      code,
		Message:   message,
		RequestID: getRequestID(c),
	})
}

// CodeErrorResponse 带错误码的错误响应
func CodeErrorResponse(c *gin.Context, code int, errorCode string, message string) {
	c.JSON(code, models.CommonResponse{
		Code:      code,
		Message:   message,
		ErrorCode: errorCode,
		RequestID: getRequestID(c),
	})
}

// 快捷错误响应函数
func BadRequestResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusBadRequest, message)
}

func UnauthorizedResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusUnauthorized, message)
}

func NotFoundResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusNotFound, message)
}

func InternalServerErrorResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusInternalServerError, message)
}

func ValidationErrorResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusUnprocessableEntity, message)
}

func TooManyRequestsResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusTooManyRequests, message)
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

// getRequestID 从gin上下文获取request_id
func getRequestID(c *gin.Context) string {
	if v, ok := c.Get("requestID"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
