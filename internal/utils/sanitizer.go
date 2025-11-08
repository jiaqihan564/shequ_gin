package utils

import (
	"html"
	"regexp"
	"strings"
	"sync"
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

	// 预编译的正则表达式（性能优化，避免在函数中重复编译）
	controlCharsRegex = regexp.MustCompile(`[\x00-\x1f\x7f]`)
	whitespaceRegex   = regexp.MustCompile(`\s+`)
	onEventAttrRegex  = regexp.MustCompile(`(?i)\s+on\w+\s*=\s*["'][^"']*["']`)
)

// 预编译的危险标签正则（性能优化）
var dangerousTagRegexes map[string]*regexp.Regexp
var dangerousAttrRegexes map[string]*regexp.Regexp
var compiledOnce sync.Once

// initRegexes 初始化正则表达式（只执行一次）
func initRegexes() {
	dangerousTagRegexes = make(map[string]*regexp.Regexp, len(dangerousTags))
	for _, tag := range dangerousTags {
		dangerousTagRegexes[tag+"_open"] = regexp.MustCompile(`(?i)<\s*` + tag + `[^>]*>`)
		dangerousTagRegexes[tag+"_close"] = regexp.MustCompile(`(?i)</\s*` + tag + `[^>]*>`)
	}

	dangerousAttrRegexes = make(map[string]*regexp.Regexp, len(dangerousAttrs))
	for _, attr := range dangerousAttrs {
		if !strings.HasPrefix(attr, "on") {
			dangerousAttrRegexes[attr] = regexp.MustCompile(`(?i)\s+` + regexp.QuoteMeta(attr) + `\s*=\s*["'][^"']*["']`)
		}
	}
}

// SanitizeHTML 清理 HTML 内容，移除危险标签和属性（性能优化）
func SanitizeHTML(input string) string {
	if input == "" {
		return ""
	}

	// 确保正则表达式已编译
	compiledOnce.Do(initRegexes)

	// 1. 移除危险标签（使用预编译的正则）
	result := input
	for _, tag := range dangerousTags {
		if openRe, ok := dangerousTagRegexes[tag+"_open"]; ok {
			result = openRe.ReplaceAllString(result, "")
		}
		if closeRe, ok := dangerousTagRegexes[tag+"_close"]; ok {
			result = closeRe.ReplaceAllString(result, "")
		}
	}

	// 2. 移除危险属性（使用预编译的正则）
	// 移除所有 on* 事件属性
	result = onEventAttrRegex.ReplaceAllString(result, "")

	// 移除其他危险属性
	for _, attr := range dangerousAttrs {
		if !strings.HasPrefix(attr, "on") {
			if re, ok := dangerousAttrRegexes[attr]; ok {
				result = re.ReplaceAllString(result, "")
			}
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

	// 清理多余空白（使用预编译的正则）
	result = strings.TrimSpace(result)
	result = whitespaceRegex.ReplaceAllString(result, " ")

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

	// 移除控制字符（使用预编译的正则）
	result = controlCharsRegex.ReplaceAllString(result, "")

	// 限制长度
	if utf8.RuneCountInString(result) > 255 {
		result = TruncateText(result, 255)
	}

	return result
}

// GenerateURLSafeFilename 生成URL安全的文件名
// 将空格、中文和特殊字符替换为安全字符，确保在URL中可以正常使用
// 优化：移除路径分隔符，清理连续下划线，限制长度
func GenerateURLSafeFilename(filename string) string {
	if filename == "" {
		return "file"
	}

	// 先清理路径分隔符（安全检查）
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	filename = strings.ReplaceAll(filename, "..", "_")

	// 提取文件扩展名
	ext := ""
	if idx := strings.LastIndex(filename, "."); idx != -1 && idx > 0 {
		ext = filename[idx:]
		filename = filename[:idx]
	}

	// 使用strings.Builder提高性能
	var sb strings.Builder
	sb.Grow(len(filename) + len(ext))

	// 处理每个字符
	for _, r := range filename {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			// 保留字母和数字
			sb.WriteRune(r)
		case r == '-' || r == '_':
			// 保留连字符和下划线
			sb.WriteRune(r)
		case r == ' ':
			// 空格替换为下划线
			sb.WriteRune('_')
		default:
			// 所有其他字符（包括中文、特殊符号等）替换为下划线
			sb.WriteRune('_')
		}
	}

	result := sb.String()

	// 清理连续的下划线（优化：使用更高效的方式）
	for strings.Contains(result, "__") {
		result = strings.ReplaceAll(result, "__", "_")
	}

	// 移除首尾下划线
	result = strings.Trim(result, "_")

	// 如果清理后为空，使用默认名称
	if result == "" {
		result = "file"
	}

	// 限制文件名长度（不包括扩展名），避免URL过长
	if len(result) > 200 {
		result = result[:200]
		result = strings.TrimRight(result, "_")
	}

	// 添加扩展名
	return result + ext
}

// ValidateContentLength 已移至 helpers.go
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

// NormalizeWhitespace 和 TruncateText 已移至 helpers.go

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
