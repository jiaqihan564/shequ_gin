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

// ErrorCode 错误代码类型
type ErrorCode int

const (
	// 认证相关错误代码
	CodeUserNotAuthenticated ErrorCode = 1001
	CodeInvalidUserID        ErrorCode = 1002
	CodeInvalidToken         ErrorCode = 1003
	CodeTokenExpired         ErrorCode = 1004
	CodeInvalidCredentials   ErrorCode = 1005
	CodeAccountDisabled      ErrorCode = 1006
	CodeTooManyLoginAttempts ErrorCode = 1007

	// 用户相关错误代码
	CodeUserNotFound       ErrorCode = 2001
	CodeUserAlreadyExists  ErrorCode = 2002
	CodeEmailAlreadyExists ErrorCode = 2003
	CodeInvalidEmail       ErrorCode = 2004
	CodeInvalidUsername    ErrorCode = 2005
	CodeInvalidPassword    ErrorCode = 2006

	// 数据库相关错误代码
	CodeDatabaseConnection ErrorCode = 3001
	CodeDatabaseQuery      ErrorCode = 3002
	CodeDatabaseInsert     ErrorCode = 3003
	CodeDatabaseUpdate     ErrorCode = 3004
	CodeDatabaseDelete     ErrorCode = 3005

	// 请求相关错误代码
	CodeInvalidRequest       ErrorCode = 4001
	CodeMissingParameter     ErrorCode = 4002
	CodeInvalidParameter     ErrorCode = 4003
	CodeValidationFailed     ErrorCode = 4004
	CodeRequestTooLarge      ErrorCode = 4005
	CodeUnsupportedMediaType ErrorCode = 4006

	// 权限相关错误代码
	CodeInsufficientPermissions ErrorCode = 5001
	CodeAccessDenied            ErrorCode = 5002
	CodeResourceNotFound        ErrorCode = 5003

	// 系统相关错误代码
	CodeInternalServerError ErrorCode = 6001
	CodeServiceUnavailable  ErrorCode = 6002
	CodeRateLimitExceeded   ErrorCode = 6003
	CodeMaintenanceMode     ErrorCode = 6004

	// 配置相关错误代码
	CodeInvalidConfig  ErrorCode = 7001
	CodeConfigNotFound ErrorCode = 7002
)

// GetErrorCode 获取错误对应的错误代码
func GetErrorCode(err error) ErrorCode {
	switch err {
	case ErrUserNotAuthenticated:
		return CodeUserNotAuthenticated
	case ErrInvalidUserID:
		return CodeInvalidUserID
	case ErrInvalidToken:
		return CodeInvalidToken
	case ErrTokenExpired:
		return CodeTokenExpired
	case ErrInvalidCredentials:
		return CodeInvalidCredentials
	case ErrAccountDisabled:
		return CodeAccountDisabled
	case ErrTooManyLoginAttempts:
		return CodeTooManyLoginAttempts
	case ErrUserNotFound:
		return CodeUserNotFound
	case ErrUserAlreadyExists:
		return CodeUserAlreadyExists
	case ErrEmailAlreadyExists:
		return CodeEmailAlreadyExists
	case ErrInvalidEmail:
		return CodeInvalidEmail
	case ErrInvalidUsername:
		return CodeInvalidUsername
	case ErrInvalidPassword:
		return CodeInvalidPassword
	case ErrDatabaseConnection:
		return CodeDatabaseConnection
	case ErrDatabaseQuery:
		return CodeDatabaseQuery
	case ErrDatabaseInsert:
		return CodeDatabaseInsert
	case ErrDatabaseUpdate:
		return CodeDatabaseUpdate
	case ErrDatabaseDelete:
		return CodeDatabaseDelete
	case ErrInvalidRequest:
		return CodeInvalidRequest
	case ErrMissingParameter:
		return CodeMissingParameter
	case ErrInvalidParameter:
		return CodeInvalidParameter
	case ErrValidationFailed:
		return CodeValidationFailed
	case ErrRequestTooLarge:
		return CodeRequestTooLarge
	case ErrUnsupportedMediaType:
		return CodeUnsupportedMediaType
	case ErrInsufficientPermissions:
		return CodeInsufficientPermissions
	case ErrAccessDenied:
		return CodeAccessDenied
	case ErrResourceNotFound:
		return CodeResourceNotFound
	case ErrInternalServerError:
		return CodeInternalServerError
	case ErrServiceUnavailable:
		return CodeServiceUnavailable
	case ErrRateLimitExceeded:
		return CodeRateLimitExceeded
	case ErrMaintenanceMode:
		return CodeMaintenanceMode
	case ErrInvalidConfig:
		return CodeInvalidConfig
	case ErrConfigNotFound:
		return CodeConfigNotFound
	default:
		return CodeInternalServerError
	}
}

// GetHTTPStatusCode 获取错误对应的HTTP状态码
func GetHTTPStatusCode(err error) int {
	switch err {
	case ErrUserNotAuthenticated, ErrInvalidToken, ErrTokenExpired:
		return 401
	case ErrInvalidCredentials, ErrAccountDisabled, ErrTooManyLoginAttempts:
		return 401
	case ErrInsufficientPermissions, ErrAccessDenied:
		return 403
	case ErrUserNotFound, ErrResourceNotFound:
		return 404
	case ErrUserAlreadyExists, ErrEmailAlreadyExists:
		return 409
	case ErrInvalidRequest, ErrMissingParameter, ErrInvalidParameter, ErrValidationFailed:
		return 400
	case ErrRequestTooLarge:
		return 413
	case ErrUnsupportedMediaType:
		return 415
	case ErrRateLimitExceeded:
		return 429
	case ErrServiceUnavailable, ErrMaintenanceMode:
		return 503
	default:
		return 500
	}
}
