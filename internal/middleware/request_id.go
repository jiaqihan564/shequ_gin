package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// genRequestID 生成请求ID
func genRequestID() string {
	timestamp := time.Now().UnixNano()
	randomBytes := make([]byte, 8)
	_, _ = rand.Read(randomBytes)
	randomStr := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("%d-%s", timestamp, randomStr)
}

// RequestIDMiddleware 请求ID中间件
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试从请求头获取请求ID
		requestID := c.GetHeader("X-Request-ID")

		// 如果没有请求ID，生成一个新的
		if requestID == "" {
			requestID = genRequestID()
		}

		// 将请求ID设置到上下文中
		c.Set("requestID", requestID)

		// 将请求ID添加到响应头
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}
