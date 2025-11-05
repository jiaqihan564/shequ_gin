package handlers

import (
	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// UserFriendlyError 用户友好的错误类型
//
// 设计目的：
//   - 将内部技术错误转换为用户可理解的消息
//   - 防止敏感信息泄露（如数据库结构、文件路径等）
//   - 保持日志记录的完整性（详细错误记录到日志）
//
// 使用场景：
//   - 数据库操作失败
//   - 外部服务调用失败
//   - 文件操作失败
//   - 任何可能暴露系统内部细节的错误
type UserFriendlyError struct {
	userMessage     string // 返回给用户的消息（隐藏技术细节）
	internalMessage string // 记录到日志的内部消息（可选）
	statusCode      int    // HTTP状态码
}

// 预定义的用户友好错误消息
var (
	ErrExecutionFailed      = UserFriendlyError{"代码执行失败，请稍后重试", "", 500}
	ErrCreateArticleFailed  = UserFriendlyError{"创建文章失败，请稍后重试", "", 500}
	ErrGetUserFailed        = UserFriendlyError{"获取用户信息失败", "", 500}
	ErrSendMessageFailed    = UserFriendlyError{"发送消息失败，请稍后重试", "", 500}
	ErrUploadFailed         = UserFriendlyError{"文件上传失败，请稍后重试", "", 500}
	ErrCreateResourceFailed = UserFriendlyError{"创建资源失败，请稍后重试", "", 500}
)

// handleInternalError 处理内部错误，记录详细信息到日志，返回用户友好消息
//
// 参数:
//   - c: Gin上下文
//   - friendlyError: 用户友好错误
//   - actualErr: 实际的错误对象（用于日志）
//   - logger: 日志记录器
//   - logFields: 额外的日志字段（可变参数，成对出现：key, value）
//
// 示例:
//
//	handleInternalError(c, ErrExecutionFailed, err, h.logger, "userID", userID, "language", req.Language)
func handleInternalError(c *gin.Context, friendlyError UserFriendlyError, actualErr error, logger utils.Logger, logFields ...interface{}) {
	// 准备日志字段
	fields := []interface{}{
		"error", actualErr.Error(),
		"endpoint", c.Request.URL.Path,
		"method", c.Request.Method,
	}

	// 添加用户提供的额外字段
	fields = append(fields, logFields...)

	// 记录内部错误详情到日志
	logger.Error(friendlyError.userMessage+" - 内部错误", fields...)

	// 返回用户友好的错误消息
	utils.ErrorResponse(c, friendlyError.statusCode, friendlyError.userMessage)
}

// logNonBlockingError 记录非阻塞错误（不影响主流程的错误）
// 这类错误只记录日志，不返回给用户
//
// 示例: 保存执行记录失败，但不影响代码执行结果返回
func logNonBlockingError(logger utils.Logger, operation string, err error, logFields ...interface{}) {
	fields := []interface{}{
		"operation", operation,
		"error", err.Error(),
	}
	fields = append(fields, logFields...)
	logger.Warn("非关键操作失败", fields...)
}
