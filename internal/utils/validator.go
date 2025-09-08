package utils

import (
	"regexp"
	"strings"
	"unicode"
)

// ValidateEmail 验证邮箱格式
func ValidateEmail(email string) bool {
	if email == "" {
		return false
	}
	// 更严格的邮箱验证正则表达式
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// ValidateUsername 验证用户名格式
func ValidateUsername(username string) bool {
	// 用户名长度3-20位，只能包含字母、数字、下划线
	if len(username) < 3 || len(username) > 20 {
		return false
	}
	// 不能以数字开头
	if unicode.IsDigit(rune(username[0])) {
		return false
	}
	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	return usernameRegex.MatchString(username)
}

// ValidatePassword 验证密码强度
func ValidatePassword(password string) bool {
	// 密码长度至少6位，最多50位
	if len(password) < 6 || len(password) > 50 {
		return false
	}

	// 检查是否包含至少一个字母
	hasLetter := false
	// 检查是否包含至少一个数字
	hasDigit := false

	for _, char := range password {
		switch {
		case unicode.IsLetter(char):
			hasLetter = true
		case unicode.IsDigit(char):
			hasDigit = true
		}
	}

	// 至少包含字母和数字
	return hasLetter && hasDigit
}

// ValidatePasswordStrong 验证强密码
func ValidatePasswordStrong(password string) bool {
	// 密码长度至少12位，最多50位
	if len(password) < 12 || len(password) > 50 {
		return false
	}

	// 检查是否包含至少一个数字
	hasDigit := false
	// 检查是否包含至少一个大写字母
	hasUpper := false
	// 检查是否包含至少一个小写字母
	hasLower := false
	// 检查是否包含至少一个特殊字符
	hasSpecial := false

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	// 必须包含大写字母、小写字母、数字和特殊字符
	return hasUpper && hasLower && hasDigit && hasSpecial
}

// SanitizeString 清理字符串
func SanitizeString(input string) string {
	// 去除首尾空格
	input = strings.TrimSpace(input)
	// 去除控制字符
	input = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r' {
			return -1
		}
		return r
	}, input)
	return input
}

// ValidatePhone 验证手机号格式（中国大陆）
func ValidatePhone(phone string) bool {
	if phone == "" {
		return false
	}
	// 中国大陆手机号正则表达式
	phoneRegex := regexp.MustCompile(`^1[3-9]\d{9}$`)
	return phoneRegex.MatchString(phone)
}

// ValidateURL 验证URL格式
func ValidateURL(url string) bool {
	if url == "" {
		return false
	}
	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	return urlRegex.MatchString(url)
}
