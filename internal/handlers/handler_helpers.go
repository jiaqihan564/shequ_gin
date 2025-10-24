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

// extractRequestContext 提取请求上下文信息
func extractRequestContext(c *gin.Context) RequestContext {
	return RequestContext{
		ClientIP:  c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		StartTime: time.Now(),
	}
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
