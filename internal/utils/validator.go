package utils

import (
	"regexp"
	"strings"
	"unicode"

	"gin/internal/config"
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

// ValidateUsername 验证用户名格式（使用默认配置）
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

// ValidateUsernameWithConfig 验证用户名格式（使用配置）
func ValidateUsernameWithConfig(username string, cfg *config.ValidationUsernameConfig) bool {
	// 用户名长度检查
	if len(username) < cfg.MinLength || len(username) > cfg.MaxLength {
		return false
	}
	// 不能以数字开头
	if unicode.IsDigit(rune(username[0])) {
		return false
	}
	return usernameRegex.MatchString(username)
}

// ValidatePassword 验证密码强度（使用默认配置）
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

// ValidatePasswordWithConfig 验证密码强度（使用配置）
func ValidatePasswordWithConfig(password string, cfg *config.ValidationPasswordConfig, isLogin bool) bool {
	// 根据是登录还是注册选择不同的最大长度
	maxLength := cfg.MaxLength
	if isLogin {
		maxLength = cfg.MaxLengthLogin
	}

	// 密码长度检查
	if len(password) < cfg.MinLength || len(password) > maxLength {
		return false
	}

	// 登录时不检查密码复杂度，只检查长度
	if isLogin {
		return true
	}

	// 注册时检查是否包含至少一个字母和一个数字
	hasLetter := false
	hasDigit := false

	for _, char := range password {
		switch {
		case unicode.IsLetter(char):
			hasLetter = true
		case unicode.IsDigit(char):
			hasDigit = true
		}
	}

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

// ValidateNickname 验证昵称格式（使用默认配置）
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

// ValidateNicknameWithConfig 验证昵称格式（使用配置）
func ValidateNicknameWithConfig(nickname string, cfg *config.ValidationNicknameConfig) bool {
	// 昵称长度检查
	if len(nickname) < cfg.MinLength || len(nickname) > cfg.MaxLength {
		return false
	}
	// 不能只包含空格
	if strings.TrimSpace(nickname) == "" {
		return false
	}
	return true
}

// ValidateBio 验证简介格式（使用默认配置）
func ValidateBio(bio string) bool {
	// 简介长度最多500个字符
	if len(bio) > 500 {
		return false
	}
	return true
}

// ValidateBioWithConfig 验证简介格式（使用配置）
func ValidateBioWithConfig(bio string, cfg *config.ValidationBioConfig) bool {
	// 简介长度检查
	if len(bio) > cfg.MaxLength {
		return false
	}
	return true
}

// ValidateURL 验证URL格式（优化：提前返回）
func ValidateURL(url string) bool {
	if url == "" {
		return false
	}
	// 简单的URL验证（优化：使用len检查避免多次调用HasPrefix）
	if len(url) < 7 { // "http://" 最短7个字符
		return false
	}
	// 检查前缀
	return (len(url) >= 7 && url[:7] == "http://") || (len(url) >= 8 && url[:8] == "https://")
}

// ValidatePhoneNumber 验证手机号码（中国）（使用默认配置）
func ValidatePhoneNumber(phone string) bool {
	if len(phone) != 11 {
		return false
	}
	// 简单验证：以1开头，第二位是3-9，后面9位是数字
	if phone[0] != '1' {
		return false
	}
	secondDigit := phone[1]
	if secondDigit < '3' || secondDigit > '9' {
		return false
	}
	for _, c := range phone[2:] {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

// ValidatePhoneNumberWithConfig 验证手机号码（使用配置）
func ValidatePhoneNumberWithConfig(phone string, cfg *config.ValidationPhoneConfig) bool {
	if len(phone) != cfg.Length {
		return false
	}
	// 简单验证：以1开头，第二位是3-9，后面9位是数字
	if phone[0] != '1' {
		return false
	}
	secondDigit := phone[1]
	if secondDigit < '3' || secondDigit > '9' {
		return false
	}
	for _, c := range phone[2:] {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

// ValidatePositiveInt 验证正整数
func ValidatePositiveInt(n int) bool {
	return n > 0
}

// ValidateRange 验证数字范围
func ValidateRange(n, min, max int) bool {
	return n >= min && n <= max
}

// ContainsSQLKeywords 检查是否包含SQL关键字（简单防护，使用默认关键词）
func ContainsSQLKeywords(input string) bool {
	// 使用默认关键词列表
	keywords := []string{
		"select", "insert", "update", "delete", "drop",
		"union", "exec", "script", "javascript",
	}
	return ContainsSQLKeywordsWithConfig(input, keywords)
}

// ContainsSQLKeywordsWithConfig 检查是否包含SQL关键字（使用配置的关键词列表）
func ContainsSQLKeywordsWithConfig(input string, keywords []string) bool {
	lowerInput := strings.ToLower(input)
	for _, keyword := range keywords {
		if strings.Contains(lowerInput, keyword) {
			return true
		}
	}
	return false
}
