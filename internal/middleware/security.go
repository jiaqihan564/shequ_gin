package middleware

import (
	"io"

	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware 添加安全头中间件
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防止XSS攻击
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")

		// 严格的传输安全（仅在HTTPS下有效）
		// c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// 内容安全策略
		c.Header("Content-Security-Policy", "default-src 'self'")

		// 控制引用信息
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// 权限策略
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}

// RequestSizeLimitMiddleware 请求大小限制中间件
func RequestSizeLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = &limitedReader{
			reader:   c.Request.Body,
			maxBytes: maxBytes,
		}
		c.Next()
	}
}

type limitedReader struct {
	reader   io.ReadCloser
	maxBytes int64
	read     int64
}

func (lr *limitedReader) Read(p []byte) (n int, err error) {
	if lr.read >= lr.maxBytes {
		return 0, io.EOF
	}

	if int64(len(p)) > lr.maxBytes-lr.read {
		p = p[:lr.maxBytes-lr.read]
	}

	n, err = lr.reader.Read(p)
	lr.read += int64(n)
	return
}

func (lr *limitedReader) Close() error {
	return lr.reader.Close()
}
