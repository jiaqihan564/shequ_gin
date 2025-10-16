package utils

import (
	"crypto/subtle"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword 生成密码哈希
// 使用 bcrypt cost 14 提供更强的安全性
func HashPassword(password string) (string, error) {
	// 验证密码长度，防止过长密码导致DoS
	if len(password) > 72 {
		return "", ErrInvalidPassword
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// CheckPasswordHash 验证密码哈希
// bcrypt.CompareHashAndPassword 内部已使用常量时间比较，无需额外处理
func CheckPasswordHash(password, hash string) bool {
	// 验证密码长度，防止过长密码导致DoS
	if len(password) > 72 || len(hash) == 0 {
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
