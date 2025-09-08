package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// RequestIDGenerator 请求ID生成器
type RequestIDGenerator struct{}

// NewRequestIDGenerator 创建请求ID生成器
func NewRequestIDGenerator() *RequestIDGenerator {
	return &RequestIDGenerator{}
}

// Generate 生成请求ID
func (g *RequestIDGenerator) Generate() string {
	// 使用时间戳和随机数生成唯一ID
	timestamp := time.Now().UnixNano()
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	randomStr := hex.EncodeToString(randomBytes)

	return fmt.Sprintf("%d-%s", timestamp, randomStr)
}

// GenerateShort 生成短请求ID
func (g *RequestIDGenerator) GenerateShort() string {
	// 生成8字节的随机字符串
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	return hex.EncodeToString(randomBytes)
}

// 全局请求ID生成器
var globalRequestIDGenerator *RequestIDGenerator

// InitRequestIDGenerator 初始化全局请求ID生成器
func InitRequestIDGenerator() {
	globalRequestIDGenerator = NewRequestIDGenerator()
}

// GetRequestIDGenerator 获取全局请求ID生成器
func GetRequestIDGenerator() *RequestIDGenerator {
	if globalRequestIDGenerator == nil {
		globalRequestIDGenerator = NewRequestIDGenerator()
	}
	return globalRequestIDGenerator
}

// GenerateRequestID 生成请求ID的便捷函数
func GenerateRequestID() string {
	return GetRequestIDGenerator().Generate()
}

// GenerateShortRequestID 生成短请求ID的便捷函数
func GenerateShortRequestID() string {
	return GetRequestIDGenerator().GenerateShort()
}
