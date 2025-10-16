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

// 标准错误码字符串（与文档一致）
const (
	ErrCodeAuthRequired      = "AUTH_REQUIRED"
	ErrCodeUploadInvalidType = "UPLOAD_INVALID_TYPE"
	ErrCodeUploadTooLarge    = "UPLOAD_TOO_LARGE"
	ErrCodeUploadFailed      = "UPLOAD_FAILED"
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

// GetHTTPStatusCode 获取错误对应的HTTP状态码
func GetHTTPStatusCode(err error) int {
	// 检查是否是AppError
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}

	// 标准错误映射
	switch {
	case errors.Is(err, ErrUserNotAuthenticated) || errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrTokenExpired):
		return 401
	case errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrAccountDisabled) || errors.Is(err, ErrTooManyLoginAttempts):
		return 401
	case errors.Is(err, ErrInsufficientPermissions) || errors.Is(err, ErrAccessDenied):
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
