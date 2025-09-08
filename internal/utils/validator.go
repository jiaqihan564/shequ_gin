package utils

import (
	"regexp"
	"strings"
)

// ValidateEmail 验证邮箱格式
func ValidateEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// ValidateUsername 验证用户名格式
func ValidateUsername(username string) bool {
	// 用户名长度3-20位，只能包含字母、数字、下划线
	if len(username) < 3 || len(username) > 20 {
		return false
	}
	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	return usernameRegex.MatchString(username)
}

// ValidatePassword 验证密码强度
func ValidatePassword(password string) bool {
	// 密码长度至少6位
	if len(password) < 6 {
		return false
	}
	// 可以添加更多密码强度验证规则
	return true
}

// SanitizeString 清理字符串
func SanitizeString(input string) string {
	// 去除首尾空格
	input = strings.TrimSpace(input)
	// 可以添加更多清理规则
	return input
}
