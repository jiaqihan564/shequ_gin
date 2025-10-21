// Package utils provides common utility functions used across the application.
// It includes string manipulation, formatting, validation, and sanitization utilities.
package utils

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// =============================================================================
// String Utilities - 字符串工具函数
// =============================================================================

// TruncateString truncates a string to the specified maximum length.
// If the string is longer than maxLen, it appends "..." at the end.
//
// Example:
//
//	TruncateString("Hello, World!", 8) // Returns: "Hello..."
//	TruncateString("Short", 10)        // Returns: "Short"
//
// Parameters:
//   - s: The input string to truncate
//   - maxLen: Maximum length (including the "..." if truncated)
//
// Returns:
//
//	The truncated string
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}

// TruncateText truncates a string by rune count (Unicode-aware).
// This is useful for handling multi-byte characters correctly.
func TruncateText(input string, maxLength int) string {
	runes := []rune(input)
	if len(runes) <= maxLength {
		return input
	}
	return string(runes[:maxLength])
}

// NormalizeWhitespace normalizes whitespace in a string.
// It replaces all consecutive whitespace characters with a single space.
func NormalizeWhitespace(input string) string {
	// Split by whitespace and rejoin
	words := strings.Fields(input)
	return strings.Join(words, " ")
}

// =============================================================================
// Formatting Utilities - 格式化工具函数
// =============================================================================

// FormatBytes formats bytes into human-readable format (KB, MB, GB, etc.)
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

// FormatSQLParams formats SQL parameters for logging.
// It handles various types and truncates long values.
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

// =============================================================================
// Validation Utilities - 验证工具函数
// =============================================================================

// ValidateContentLength validates content length in runes (Unicode-aware).
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

// IsEmpty checks if a string is empty or contains only whitespace.
func IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

// =============================================================================
// Sanitization Utilities - 数据清理工具函数
// =============================================================================

// SanitizeToken sanitizes a token for logging by keeping only first and last 4 chars.
func SanitizeToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// SanitizeAuthHeader sanitizes Authorization header for logging.
func SanitizeAuthHeader(header string) string {
	if header == "" {
		return ""
	}
	if len(header) > 20 {
		return header[:7] + "..." + header[len(header)-4:]
	}
	return "Bearer ***"
}

// SanitizeEmail sanitizes email address for logging.
// Example: user@example.com -> us***@example.com
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

// =============================================================================
// Slice Utilities - 切片工具函数
// =============================================================================

// ContainsString checks if a string slice contains a specific string.
func ContainsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// UniqueStrings removes duplicate strings from a slice.
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

// UniqueUints removes duplicate uints from a slice.
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
