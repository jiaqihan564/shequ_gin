package utils

import (
	"html"
	"regexp"
	"strings"
	"unicode/utf8"
)

// XSS 危险标签和属性
var (
	dangerousTags = []string{
		"script", "iframe", "object", "embed", "link", "style",
		"meta", "base", "form", "input", "button", "textarea",
	}

	dangerousAttrs = []string{
		"on", // 所有 on* 事件属性
		"javascript:", "vbscript:", "data:",
		"formaction", "action",
	}

	// HTML 标签正则
	htmlTagRegex = regexp.MustCompile(`<[^>]+>`)

	// SQL 注入模式（仅作为额外检测，不应依赖此作为唯一防护）
	sqlInjectionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute)`),
		regexp.MustCompile(`(?i)(--|#|/\*|\*/)`),
		regexp.MustCompile(`(?i)(or|and)\s+\d+\s*=\s*\d+`),
		regexp.MustCompile(`(?i)(or|and)\s+['"][^'"]*['"]\s*=\s*['"][^'"]*['"]`),
	}

	// XSS 攻击模式
	xssPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
		regexp.MustCompile(`(?i)javascript:`),
		regexp.MustCompile(`(?i)on\w+\s*=`),
		regexp.MustCompile(`(?i)<iframe[^>]*>`),
		regexp.MustCompile(`(?i)vbscript:`),
		regexp.MustCompile(`(?i)data:text/html`),
	}

	// 路径遍历模式
	pathTraversalRegex = regexp.MustCompile(`\.\.(/|\\)`)

	// Email 提取正则（用于脱敏）
	emailMaskRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

	// 手机号正则（中国）
	phoneRegex = regexp.MustCompile(`1[3-9]\d{9}`)

	// 身份证号正则（中国）
	idCardRegex = regexp.MustCompile(`\d{17}[\dXx]`)
)

// SanitizeHTML 清理 HTML 内容，移除危险标签和属性
func SanitizeHTML(input string) string {
	if input == "" {
		return ""
	}

	// 1. 移除危险标签
	result := input
	for _, tag := range dangerousTags {
		// 移除开标签和闭标签
		result = regexp.MustCompile(`(?i)<\s*`+tag+`[^>]*>`).ReplaceAllString(result, "")
		result = regexp.MustCompile(`(?i)</\s*`+tag+`[^>]*>`).ReplaceAllString(result, "")
	}

	// 2. 移除危险属性
	for _, attr := range dangerousAttrs {
		if strings.HasPrefix(attr, "on") {
			// 移除所有 on* 事件属性
			result = regexp.MustCompile(`(?i)\s+on\w+\s*=\s*["'][^"']*["']`).ReplaceAllString(result, "")
		} else {
			result = regexp.MustCompile(`(?i)\s+`+regexp.QuoteMeta(attr)+`\s*=\s*["'][^"']*["']`).ReplaceAllString(result, "")
		}
	}

	// 3. 转义剩余的特殊字符
	result = html.EscapeString(result)

	return result
}

// StripHTML 完全移除所有 HTML 标签
func StripHTML(input string) string {
	if input == "" {
		return ""
	}

	// 移除所有 HTML 标签
	result := htmlTagRegex.ReplaceAllString(input, "")

	// 解码 HTML 实体
	result = html.UnescapeString(result)

	// 清理多余空白
	result = strings.TrimSpace(result)
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")

	return result
}

// EscapeHTML HTML 转义
func EscapeHTML(input string) string {
	return html.EscapeString(input)
}

// DetectSQLInjection 检测潜在的 SQL 注入攻击
func DetectSQLInjection(input string) bool {
	if input == "" {
		return false
	}

	lower := strings.ToLower(input)

	for _, pattern := range sqlInjectionPatterns {
		if pattern.MatchString(lower) {
			return true
		}
	}

	return false
}

// DetectXSS 检测潜在的 XSS 攻击
func DetectXSS(input string) bool {
	if input == "" {
		return false
	}

	lower := strings.ToLower(input)

	for _, pattern := range xssPatterns {
		if pattern.MatchString(lower) {
			return true
		}
	}

	return false
}

// DetectPathTraversal 检测路径遍历攻击
func DetectPathTraversal(input string) bool {
	return pathTraversalRegex.MatchString(input)
}

// SanitizeFilename 清理文件名，移除危险字符
func SanitizeFilename(filename string) string {
	if filename == "" {
		return ""
	}

	// 移除路径分隔符
	result := strings.ReplaceAll(filename, "/", "_")
	result = strings.ReplaceAll(result, "\\", "_")
	result = strings.ReplaceAll(result, "..", "_")

	// 移除控制字符
	result = regexp.MustCompile(`[\x00-\x1f\x7f]`).ReplaceAllString(result, "")

	// 限制长度
	if utf8.RuneCountInString(result) > 255 {
		result = TruncateText(result, 255)
	}

	return result
}

// MaskEmail 邮箱脱敏
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}

	username := parts[0]
	domain := parts[1]

	if len(username) <= 2 {
		return email
	}

	// 保留前1位和后1位，中间用 * 代替
	masked := string(username[0]) + strings.Repeat("*", len(username)-2) + string(username[len(username)-1])
	return masked + "@" + domain
}

// MaskPhone 手机号脱敏
func MaskPhone(phone string) string {
	if len(phone) != 11 {
		return phone
	}

	// 保留前3位和后4位，中间用 **** 代替
	return phone[:3] + "****" + phone[7:]
}

// MaskIDCard 身份证号脱敏
func MaskIDCard(idCard string) string {
	if len(idCard) != 18 {
		return idCard
	}

	// 保留前6位和后4位，中间用 * 代替
	return idCard[:6] + "********" + idCard[14:]
}

// SanitizeLogData 日志数据脱敏
func SanitizeLogData(data string) string {
	result := data

	// 脱敏邮箱
	result = emailMaskRegex.ReplaceAllStringFunc(result, MaskEmail)

	// 脱敏手机号
	result = phoneRegex.ReplaceAllStringFunc(result, MaskPhone)

	// 脱敏身份证号
	result = idCardRegex.ReplaceAllStringFunc(result, MaskIDCard)

	return result
}

// ValidateContentLength is now in helpers.go - use that instead
// Keeping error definitions here for backward compatibility
var (
	ErrContentTooShort = NewAppError(ErrInvalidParameter, "内容太短", 400)
	ErrContentTooLong  = NewAppError(ErrInvalidParameter, "内容太长", 400)
)

// ContainsProhibitedWords 检查是否包含禁用词
func ContainsProhibitedWords(content string, prohibitedWords []string) bool {
	lower := strings.ToLower(content)

	for _, word := range prohibitedWords {
		if strings.Contains(lower, strings.ToLower(word)) {
			return true
		}
	}

	return false
}

// NormalizeWhitespace is now in helpers.go - use that instead
// TruncateText is now in helpers.go - use that instead

// DeepSanitize 深度清理（组合多种清理方法）
func DeepSanitize(input string) (string, []string) {
	warnings := []string{}

	// 检测攻击模式
	if DetectXSS(input) {
		warnings = append(warnings, "检测到潜在XSS攻击")
	}

	if DetectSQLInjection(input) {
		warnings = append(warnings, "检测到潜在SQL注入")
	}

	if DetectPathTraversal(input) {
		warnings = append(warnings, "检测到路径遍历攻击")
	}

	// 清理 HTML
	result := SanitizeHTML(input)

	// 规范化空白
	result = NormalizeWhitespace(result)

	return result, warnings
}
