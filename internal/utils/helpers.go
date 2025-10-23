// Package utils 提供通用工具函数
package utils

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// TruncateString 截断字符串到指定最大长度
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}

// TruncateText 按字符数截断字符串（支持Unicode）
func TruncateText(input string, maxLength int) string {
	runes := []rune(input)
	if len(runes) <= maxLength {
		return input
	}
	return string(runes[:maxLength])
}

// NormalizeWhitespace 规范化字符串中的空格
func NormalizeWhitespace(input string) string {
	words := strings.Fields(input)
	return strings.Join(words, " ")
}

// FormatBytes 格式化字节为可读格式
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	if exp >= len(units) {
		exp = len(units) - 1
	}

	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// FormatSQLParams 格式化SQL参数用于日志记录
func FormatSQLParams(params []interface{}) string {
	if len(params) == 0 {
		return "[]"
	}

	parts := make([]string, 0, len(params))
	for i, p := range params {
		var str string
		switch v := p.(type) {
		case string:
			if len(v) > 50 {
				str = fmt.Sprintf("\"%s...\"", v[:47])
			} else {
				str = fmt.Sprintf("\"%s\"", v)
			}
		case []byte:
			if len(v) > 50 {
				str = fmt.Sprintf("[]byte(%d bytes)", len(v))
			} else {
				str = fmt.Sprintf("[]byte(%s)", string(v))
			}
		case nil:
			str = "null"
		default:
			str = fmt.Sprintf("%v", v)
		}
		parts = append(parts, fmt.Sprintf("[%d]=%s", i, str))
	}

	return "[" + strings.Join(parts, ", ") + "]"
}

// ValidateContentLength 验证内容长度
func ValidateContentLength(content string, minLength, maxLength int) error {
	length := utf8.RuneCountInString(content)

	if length < minLength {
		return fmt.Errorf("content too short: minimum %d characters required", minLength)
	}

	if length > maxLength {
		return fmt.Errorf("content too long: maximum %d characters allowed", maxLength)
	}

	return nil
}

// IsEmpty 检查字符串是否为空
func IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

// SanitizeToken 清理token用于日志记录
func SanitizeToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// SanitizeAuthHeader 清理授权头用于日志记录
func SanitizeAuthHeader(header string) string {
	if header == "" {
		return ""
	}
	if len(header) > 20 {
		return header[:7] + "..." + header[len(header)-4:]
	}
	return "Bearer ***"
}

// SanitizeEmail 清理邮箱地址用于日志记录
func SanitizeEmail(email string) string {
	if email == "" {
		return ""
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***@***"
	}

	localPart := parts[0]
	if len(localPart) <= 2 {
		return "**@" + parts[1]
	}

	masked := localPart[:2] + strings.Repeat("*", len(localPart)-2)
	return masked + "@" + parts[1]
}

// ContainsString 检查字符串切片是否包含指定字符串
func ContainsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// UniqueStrings 移除字符串切片中的重复项
func UniqueStrings(slice []string) []string {
	seen := make(map[string]bool, len(slice))
	result := make([]string, 0, len(slice))

	for _, str := range slice {
		if !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}

	return result
}

// UniqueUints 移除uint切片中的重复项
func UniqueUints(slice []uint) []uint {
	seen := make(map[uint]bool, len(slice))
	result := make([]uint, 0, len(slice))

	for _, num := range slice {
		if !seen[num] {
			seen[num] = true
			result = append(result, num)
		}
	}

	return result
}
