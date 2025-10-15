package middleware

import (
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

var requestCounter uint64

// RequestIDMiddleware 请求ID中间件（使用原子计数器，性能最优）
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成请求ID：时间戳 + 原子计数器
		timestamp := time.Now().UnixNano()
		count := atomic.AddUint64(&requestCounter, 1)
		requestID := strconv.FormatInt(timestamp, 10) + "-" + strconv.FormatUint(count, 10)

		c.Set("requestID", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}
