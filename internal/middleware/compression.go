// Package middleware 提供Gin框架的HTTP中间件组件
package middleware

import (
	"compress/gzip"
	"io"
	"strings"
	"sync"
	"sync/atomic"

	"gin/internal/utils"

	"github.com/gin-gonic/gin"
)

const (
	// 默认压缩配置
	DefaultCompressionLevel   = gzip.DefaultCompression // 6
	DefaultMinCompressionSize = 1024                    // 1KB 最小压缩大小
	BestSpeedCompressionLevel = gzip.BestSpeed          // 1 - 最快速度
	BestCompressionLevel      = gzip.BestCompression    // 9 - 最高压缩率
)

// 压缩统计
var (
	compressedRequests   uint64
	uncompressedRequests uint64
	totalBytesSaved      uint64
)

// gzip writer池，减少内存分配
var gzipWriterPools = []*sync.Pool{
	// Level 1 - BestSpeed
	{New: func() interface{} { w, _ := gzip.NewWriterLevel(io.Discard, gzip.BestSpeed); return w }},
	// Level 6 - DefaultCompression
	{New: func() interface{} { w, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression); return w }},
	// Level 9 - BestCompression
	{New: func() interface{} { w, _ := gzip.NewWriterLevel(io.Discard, gzip.BestCompression); return w }},
}

type gzipWriter struct {
	gin.ResponseWriter
	writer         *gzip.Writer
	originalSize   int
	compressedSize int
	shouldCompress bool
	minSize        int
}

func (g *gzipWriter) Write(data []byte) (int, error) {
	g.originalSize += len(data)

	// 如果数据小于最小压缩大小，不压缩
	if !g.shouldCompress && g.originalSize < g.minSize {
		return g.ResponseWriter.Write(data)
	}

	// 第一次达到最小大小时，设置压缩头
	if !g.shouldCompress && g.originalSize >= g.minSize {
		g.shouldCompress = true
		g.Header().Set("Content-Encoding", "gzip")
		g.Header().Set("Vary", "Accept-Encoding")
		g.Header().Del("Content-Length") // 删除原始Content-Length
	}

	if g.shouldCompress {
		n, err := g.writer.Write(data)
		g.compressedSize += n
		return n, err
	}

	return g.ResponseWriter.Write(data)
}

func (g *gzipWriter) WriteString(s string) (int, error) {
	return g.Write([]byte(s))
}

// WriteHeader 重写WriteHeader以支持条件压缩
func (g *gzipWriter) WriteHeader(code int) {
	// 检查Content-Type是否适合压缩
	contentType := g.Header().Get("Content-Type")
	if !shouldCompressContentType(contentType) {
		g.shouldCompress = false
	}
	g.ResponseWriter.WriteHeader(code)
}

// shouldCompressContentType 判断Content-Type是否适合压缩
func shouldCompressContentType(contentType string) bool {
	// 只压缩文本类型的响应
	compressibleTypes := []string{
		"text/",
		"application/json",
		"application/javascript",
		"application/xml",
		"application/xhtml+xml",
		"image/svg+xml",
	}

	for _, ct := range compressibleTypes {
		if strings.Contains(contentType, ct) {
			return true
		}
	}
	return false
}

// getGzipWriterPool 根据压缩级别获取对应的writer池
func getGzipWriterPool(level int) *sync.Pool {
	switch level {
	case gzip.BestSpeed:
		return gzipWriterPools[0]
	case gzip.BestCompression:
		return gzipWriterPools[2]
	default:
		return gzipWriterPools[1]
	}
}

// CompressionMiddleware 增强的压缩中间件
// 参数：
//   - level: 压缩级别 (1-9, 默认6)
//   - minSize: 最小压缩大小（字节，默认1024）
func CompressionMiddleware(level int, minSize int) gin.HandlerFunc {
	if level < gzip.BestSpeed || level > gzip.BestCompression {
		level = DefaultCompressionLevel
	}
	if minSize <= 0 {
		minSize = DefaultMinCompressionSize
	}

	pool := getGzipWriterPool(level)
	logger := utils.GetLogger()

	logger.Info("压缩中间件已初始化",
		"level", level,
		"minSize", minSize,
		"levelName", getCompressionLevelName(level))

	return func(c *gin.Context) {
		// 检查客户端是否支持gzip
		acceptEncoding := c.GetHeader("Accept-Encoding")
		if !strings.Contains(acceptEncoding, "gzip") {
			atomic.AddUint64(&uncompressedRequests, 1)
			c.Next()
			return
		}

		// 跳过不适合压缩的请求（如WebSocket、流式响应）
		if c.GetHeader("Upgrade") == "websocket" {
			c.Next()
			return
		}

		// 从池中获取gzip writer
		gz := pool.Get().(*gzip.Writer)
		defer pool.Put(gz)

		gz.Reset(c.Writer)

		gw := &gzipWriter{
			ResponseWriter: c.Writer,
			writer:         gz,
			shouldCompress: false,
			minSize:        minSize,
		}
		c.Writer = gw

		c.Next()

		// 关闭writer并记录统计
		if gw.shouldCompress {
			gz.Close()
			atomic.AddUint64(&compressedRequests, 1)
			saved := gw.originalSize - gw.compressedSize
			if saved > 0 {
				atomic.AddUint64(&totalBytesSaved, uint64(saved))
			}

			// 记录压缩效果（仅用于调试）
			if gw.originalSize > 0 {
				ratio := float64(gw.compressedSize) / float64(gw.originalSize) * 100
				logger.Debug("响应已压缩",
					"path", c.Request.URL.Path,
					"original", gw.originalSize,
					"compressed", gw.compressedSize,
					"saved", saved,
					"ratio", ratio)
			}
		} else {
			atomic.AddUint64(&uncompressedRequests, 1)
		}
	}
}

// DefaultCompressionMiddleware 使用默认配置的压缩中间件
func DefaultCompressionMiddleware() gin.HandlerFunc {
	return CompressionMiddleware(DefaultCompressionLevel, DefaultMinCompressionSize)
}

// FastCompressionMiddleware 快速压缩中间件（速度优先）
func FastCompressionMiddleware() gin.HandlerFunc {
	return CompressionMiddleware(BestSpeedCompressionLevel, DefaultMinCompressionSize)
}

// BestCompressionMiddleware 最佳压缩中间件（压缩率优先）
func BestCompressionMiddleware() gin.HandlerFunc {
	return CompressionMiddleware(BestCompressionLevel, DefaultMinCompressionSize)
}

// GetCompressionStats 获取压缩统计信息
func GetCompressionStats() map[string]interface{} {
	compressed := atomic.LoadUint64(&compressedRequests)
	uncompressed := atomic.LoadUint64(&uncompressedRequests)
	saved := atomic.LoadUint64(&totalBytesSaved)
	total := compressed + uncompressed

	var compressionRate float64
	if total > 0 {
		compressionRate = float64(compressed) / float64(total) * 100
	}

	return map[string]interface{}{
		"compressed_requests":   compressed,
		"uncompressed_requests": uncompressed,
		"total_requests":        total,
		"compression_rate":      compressionRate,
		"total_bytes_saved":     saved,
		"avg_bytes_saved": func() uint64 {
			if compressed > 0 {
				return saved / compressed
			}
			return 0
		}(),
	}
}

// getCompressionLevelName 获取压缩级别名称
func getCompressionLevelName(level int) string {
	switch level {
	case gzip.BestSpeed:
		return "BestSpeed"
	case gzip.BestCompression:
		return "BestCompression"
	case gzip.DefaultCompression:
		return "Default"
	default:
		return "Custom"
	}
}
