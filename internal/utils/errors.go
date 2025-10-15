package utils

import "errors"

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

	// 系统相关错误
	ErrInternalServerError = errors.New("内部服务器错误")
	ErrServiceUnavailable  = errors.New("服务不可用")
	ErrRateLimitExceeded   = errors.New("请求频率过高")
)

// 标准错误码字符串（与文档一致）
const (
	ErrCodeAuthRequired      = "AUTH_REQUIRED"
	ErrCodeUploadInvalidType = "UPLOAD_INVALID_TYPE"
	ErrCodeUploadTooLarge    = "UPLOAD_TOO_LARGE"
	ErrCodeUploadFailed      = "UPLOAD_FAILED"
)

// GetHTTPStatusCode 获取错误对应的HTTP状态码
func GetHTTPStatusCode(err error) int {
	switch {
	case errors.Is(err, ErrUserNotAuthenticated), errors.Is(err, ErrInvalidToken), errors.Is(err, ErrTokenExpired),
		errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrAccountDisabled), errors.Is(err, ErrTooManyLoginAttempts):
		return 401
	case errors.Is(err, ErrUserNotFound):
		return 404
	case errors.Is(err, ErrUserAlreadyExists), errors.Is(err, ErrEmailAlreadyExists):
		return 409
	case errors.Is(err, ErrInvalidRequest), errors.Is(err, ErrMissingParameter), errors.Is(err, ErrInvalidParameter),
		errors.Is(err, ErrValidationFailed), errors.Is(err, ErrInvalidUsername), errors.Is(err, ErrInvalidEmail), errors.Is(err, ErrInvalidPassword):
		return 400
	case errors.Is(err, ErrRequestTooLarge):
		return 413
	case errors.Is(err, ErrServiceUnavailable):
		return 503
	default:
		return 500
	}
}
