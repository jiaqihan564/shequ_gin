package utils

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

// ValidateEmail 验证邮箱格式
func ValidateEmail(email string) bool {
	if email == "" {
		return false
	}
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

//

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

// ValidateNickname 验证昵称格式
func ValidateNickname(nickname string) bool {
	// 昵称长度1-50个字符
	if len(nickname) < 1 || len(nickname) > 50 {
		return false
	}
	// 不能只包含空格
	if strings.TrimSpace(nickname) == "" {
		return false
	}
	return true
}

// ValidateBio 验证简介格式
func ValidateBio(bio string) bool {
	// 简介长度最多500个字符
	if len(bio) > 500 {
		return false
	}
	return true
}

// ValidateURL 验证URL格式
func ValidateURL(url string) bool {
	if url == "" {
		return false
	}
	// 简单的URL验证
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}
