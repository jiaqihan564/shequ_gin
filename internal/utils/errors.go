package utils

import (
	"errors"
	"fmt"
)

// 定义常用错误
var (
	// 认证相关错误
	ErrUserNotAuthenticated = errors.New("用户未认证")
	ErrInvalidUserID        = errors.New("无效的用户ID")
	ErrInvalidToken         = errors.New("无效的token")
	ErrTokenExpired         = errors.New("token已过期")
	ErrInvalidCredentials   = errors.New("用户名或密码错误")
	ErrAccountDisabled      = errors.New("账户已被禁用")
	ErrTooManyLoginAttempts = errors.New("登录尝试次数过多，请稍后再试")

	// 用户相关错误
	ErrUserNotFound       = errors.New("用户不存在")
	ErrUserAlreadyExists  = errors.New("用户已存在")
	ErrEmailAlreadyExists = errors.New("邮箱已被注册")
	ErrInvalidEmail       = errors.New("无效的邮箱格式")
	ErrInvalidUsername    = errors.New("无效的用户名格式")
	ErrInvalidPassword    = errors.New("无效的密码格式")

	// 数据库相关错误
	ErrDatabaseConnection = errors.New("数据库连接失败")
	ErrDatabaseQuery      = errors.New("数据库查询失败")
	ErrDatabaseInsert     = errors.New("数据库插入失败")
	ErrDatabaseUpdate     = errors.New("数据库更新失败")
	ErrDatabaseDelete     = errors.New("数据库删除失败")

	// 请求相关错误
	ErrInvalidRequest       = errors.New("无效的请求")
	ErrMissingParameter     = errors.New("缺少必要参数")
	ErrInvalidParameter     = errors.New("无效的参数")
	ErrValidationFailed     = errors.New("参数验证失败")
	ErrRequestTooLarge      = errors.New("请求体过大")
	ErrUnsupportedMediaType = errors.New("不支持的媒体类型")

	// 权限相关错误
	ErrInsufficientPermissions = errors.New("权限不足")
	ErrAccessDenied            = errors.New("访问被拒绝")
	ErrUnauthorized            = errors.New("未授权操作")
	ErrResourceNotFound        = errors.New("资源不存在")

	// 系统相关错误
	ErrInternalServerError = errors.New("内部服务器错误")
	ErrServiceUnavailable  = errors.New("服务不可用")
	ErrRateLimitExceeded   = errors.New("请求频率过高")
	ErrMaintenanceMode     = errors.New("系统维护中")

	// 配置相关错误
	ErrInvalidConfig  = errors.New("无效的配置")
	ErrConfigNotFound = errors.New("配置文件不存在")
)

// Standard Error Codes - 标准错误码（用于API响应）
const (
	// Authentication & Authorization - 认证和授权
	ErrCodeAuthRequired       = "AUTH_REQUIRED"
	ErrCodeInvalidToken       = "INVALID_TOKEN"
	ErrCodeTokenExpired       = "TOKEN_EXPIRED"
	ErrCodeInvalidCredentials = "INVALID_CREDENTIALS"
	ErrCodePermissionDenied   = "PERMISSION_DENIED"

	// User Management - 用户管理
	ErrCodeUserNotFound = "USER_NOT_FOUND"
	ErrCodeUserExists   = "USER_EXISTS"
	ErrCodeEmailExists  = "EMAIL_EXISTS"

	// Validation - 数据验证
	ErrCodeInvalidInput     = "INVALID_INPUT"
	ErrCodeMissingParam     = "MISSING_PARAMETER"
	ErrCodeValidationFailed = "VALIDATION_FAILED"

	// Upload - 文件上传
	ErrCodeUploadInvalidType = "UPLOAD_INVALID_TYPE"
	ErrCodeUploadTooLarge    = "UPLOAD_TOO_LARGE"
	ErrCodeUploadFailed      = "UPLOAD_FAILED"

	// Database - 数据库
	ErrCodeDatabaseError  = "DATABASE_ERROR"
	ErrCodeRecordNotFound = "RECORD_NOT_FOUND"
	ErrCodeDuplicateEntry = "DUPLICATE_ENTRY"

	// Rate Limiting - 限流
	ErrCodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"

	// System - 系统
	ErrCodeInternalError      = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

// AppError 应用错误（包含上下文信息）
type AppError struct {
	Err     error                  // 原始错误
	Message string                 // 用户友好的错误消息
	Code    int                    // HTTP状态码
	Context map[string]interface{} // 上下文信息
}

// Error 实现error接口
func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "未知错误"
}

// Unwrap 支持errors.Unwrap
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError 创建应用错误
func NewAppError(err error, message string, code int) *AppError {
	return &AppError{
		Err:     err,
		Message: message,
		Code:    code,
		Context: make(map[string]interface{}),
	}
}

// WithContext 添加上下文信息
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	e.Context[key] = value
	return e
}

// WrapError 包装错误并添加上下文
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// GetHTTPStatusCode returns the HTTP status code for an error.
// It checks if the error is an AppError first, then falls back to standard error mapping.
func GetHTTPStatusCode(err error) int {
	if err == nil {
		return 200
	}

	// Check if it's an AppError
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}

	// Standard error mapping
	switch {
	case errors.Is(err, ErrUserNotAuthenticated) || errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrTokenExpired):
		return 401
	case errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrAccountDisabled) || errors.Is(err, ErrTooManyLoginAttempts):
		return 401
	case errors.Is(err, ErrInsufficientPermissions) || errors.Is(err, ErrAccessDenied) || errors.Is(err, ErrUnauthorized):
		return 403
	case errors.Is(err, ErrUserNotFound) || errors.Is(err, ErrResourceNotFound):
		return 404
	case errors.Is(err, ErrUserAlreadyExists) || errors.Is(err, ErrEmailAlreadyExists):
		return 409
	case errors.Is(err, ErrInvalidRequest) || errors.Is(err, ErrMissingParameter) ||
		errors.Is(err, ErrInvalidParameter) || errors.Is(err, ErrValidationFailed):
		return 400
	case errors.Is(err, ErrInvalidUsername) || errors.Is(err, ErrInvalidEmail) || errors.Is(err, ErrInvalidPassword):
		return 400
	case errors.Is(err, ErrRequestTooLarge):
		return 413
	case errors.Is(err, ErrUnsupportedMediaType):
		return 415
	case errors.Is(err, ErrRateLimitExceeded):
		return 429
	case errors.Is(err, ErrServiceUnavailable) || errors.Is(err, ErrMaintenanceMode):
		return 503
	default:
		return 500
	}
}

// GetErrorCode returns the error code string for API responses.
func GetErrorCode(err error) string {
	if err == nil {
		return ""
	}

	// Check if it's an AppError with custom error code
	var appErr *AppError
	if errors.As(err, &appErr) {
		// Use the error type as code if available
		if errCode, ok := appErr.Context["error_code"].(string); ok {
			return errCode
		}
	}

	// Map standard errors to error codes
	switch {
	case errors.Is(err, ErrUserNotAuthenticated) || errors.Is(err, ErrInvalidToken):
		return ErrCodeAuthRequired
	case errors.Is(err, ErrTokenExpired):
		return ErrCodeTokenExpired
	case errors.Is(err, ErrInvalidCredentials):
		return ErrCodeInvalidCredentials
	case errors.Is(err, ErrUserNotFound):
		return ErrCodeUserNotFound
	case errors.Is(err, ErrUserAlreadyExists):
		return ErrCodeUserExists
	case errors.Is(err, ErrEmailAlreadyExists):
		return ErrCodeEmailExists
	case errors.Is(err, ErrInvalidParameter) || errors.Is(err, ErrValidationFailed):
		return ErrCodeInvalidInput
	case errors.Is(err, ErrMissingParameter):
		return ErrCodeMissingParam
	case errors.Is(err, ErrRateLimitExceeded):
		return ErrCodeRateLimitExceeded
	case errors.Is(err, ErrDatabaseQuery) || errors.Is(err, ErrDatabaseConnection):
		return ErrCodeDatabaseError
	default:
		return ErrCodeInternalError
	}
}
