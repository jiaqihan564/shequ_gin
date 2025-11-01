package utils

import (
	"crypto/subtle"

	"gin/internal/config"

	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultBcryptCost 默认bcrypt成本（推荐10-12之间）
	// cost=10: ~100ms, cost=12: ~400ms, cost=14: ~1.6s
	DefaultBcryptCost = 12
)

// HashPassword 生成密码哈希
// 使用可配置的 bcrypt cost 提供安全性与性能的平衡
func HashPassword(password string) (string, error) {
	return HashPasswordWithCost(password, DefaultBcryptCost)
}

// HashPasswordWithCost 使用指定成本生成密码哈希
func HashPasswordWithCost(password string, cost int) (string, error) {
	// 使用默认配置验证（向后兼容）
	return HashPasswordWithConfig(password, cost, nil)
}

// HashPasswordWithConfig 使用配置生成密码哈希
func HashPasswordWithConfig(password string, cost int, cfg *config.SecurityPasswordConfig) (string, error) {
	// 设置默认值
	maxBytes := 72
	minCost := 10
	maxCost := 14

	// 如果提供了配置，使用配置值
	if cfg != nil {
		maxBytes = cfg.PasswordMaxBytes
		minCost = cfg.BcryptCostMin
		maxCost = cfg.BcryptCostMax
	}

	// 验证密码长度，防止过长密码导致DoS
	if len(password) > maxBytes {
		return "", ErrInvalidPassword
	}

	// 验证成本范围
	if cost < minCost {
		cost = minCost
	}
	if cost > maxCost {
		cost = maxCost
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", WrapError(err, "密码加密失败")
	}
	return string(bytes), nil
}

// CheckPasswordHash 验证密码哈希
// bcrypt.CompareHashAndPassword 内部已使用常量时间比较，无需额外处理
func CheckPasswordHash(password, hash string) bool {
	return CheckPasswordHashWithConfig(password, hash, nil)
}

// CheckPasswordHashWithConfig 验证密码哈希（使用配置）
func CheckPasswordHashWithConfig(password, hash string, cfg *config.SecurityPasswordConfig) bool {
	// 设置默认值
	maxBytes := 72

	// 如果提供了配置，使用配置值
	if cfg != nil {
		maxBytes = cfg.PasswordMaxBytes
	}

	// 验证密码长度，防止过长密码导致DoS
	if len(password) > maxBytes || len(hash) == 0 {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// SecureCompare 常量时间字符串比较，防止时序攻击
// 用于比较token、签名等敏感数据
func SecureCompare(a, b string) bool {
	// 先比较长度，如果长度不同直接返回false
	// 但仍然执行subtle.ConstantTimeCompare以保持常量时间
	if len(a) != len(b) {
		// 使用假数据进行比较以保持常量时间
		subtle.ConstantTimeCompare([]byte(a), []byte(a))
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
