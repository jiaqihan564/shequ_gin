package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// 随机字节池（性能优化）
var randomBytesPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 8)
		return &b
	},
}

// genRequestID 生成请求ID（优化：使用对象池）
func genRequestID() string {
	timestamp := time.Now().UnixNano()

	// 从池中获取字节切片指针
	randomBytesPtr := randomBytesPool.Get().(*[]byte)
	defer randomBytesPool.Put(randomBytesPtr)

	randomBytes := *randomBytesPtr
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
