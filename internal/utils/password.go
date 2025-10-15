package utils

import "golang.org/x/crypto/bcrypt"

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
