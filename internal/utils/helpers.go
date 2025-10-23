// Package utils 提供通用工具函数
package utils

import (
	"fmt"
	"strings"
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
