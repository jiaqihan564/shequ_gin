package handlers

import (
	"time"

	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

// RequestContext 请求上下文信息
type RequestContext struct {
	ClientIP  string
	UserAgent string
	StartTime time.Time
}

// extractRequestContext 提取请求上下文信息（优化：缓存多次调用的值）
func extractRequestContext(c *gin.Context) RequestContext {
	// 尝试从context获取已缓存的值，避免重复计算
	if ctx, exists := c.Get("_request_context"); exists {
		if reqCtx, ok := ctx.(RequestContext); ok {
			return reqCtx
		}
	}

	// 创建新的上下文（只执行一次）
	reqCtx := RequestContext{
		ClientIP:  c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		StartTime: time.Now(),
	}

	// 缓存到context中
	c.Set("_request_context", reqCtx)
	return reqCtx
}

// getUserIDOrFail 获取用户ID，失败时自动返回错误响应
// 返回值：userID, isOK
func getUserIDOrFail(c *gin.Context) (uint, bool) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.UnauthorizedResponse(c, err.Error())
		return 0, false
	}
	return userID, true
}

// bindJSONOrFail 绑定JSON请求体，失败时自动返回错误响应
// 返回值：isOK
func bindJSONOrFail(c *gin.Context, req interface{}, logger utils.Logger, funcName string) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		if logger != nil && funcName != "" {
			logger.Warn(funcName+"请求参数错误", "error", err.Error())
		}
		utils.ValidationErrorResponse(c, "请求参数错误: "+err.Error())
		return false
	}
	return true
}

// parseUintParam 解析URL参数为uint，失败时自动返回错误响应
// 返回值：value, isOK
func parseUintParam(c *gin.Context, paramName string, errorMsg string) (uint, bool) {
	value, err := utils.ParseUintParam(c, paramName)
	if err != nil {
		utils.BadRequestResponse(c, errorMsg)
		return 0, false
	}
	return value, true
}
