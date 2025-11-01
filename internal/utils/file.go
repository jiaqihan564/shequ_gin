package utils

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"strings"
	"sync"
	"unicode"
)

// 魔数验证buffer池（性能优化）
// 注意：存储 *[]byte 而非 []byte，避免 interface{} 装箱时的堆分配
// 默认buffer大小16字节，可通过配置调整
var magicNumberBufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 16) // 默认16字节
		return &buf // 返回指针，避免 Put 时的内存分配
	},
}

// FileValidator 文件验证器
type FileValidator struct {
	MaxSize      int64
	AllowedTypes []string
	MagicNumbers map[string][]byte // 文件魔数（文件签名）
}

// NewFileValidator 创建文件验证器
func NewFileValidator(maxSize int64, allowedTypes []string) *FileValidator {
	return &FileValidator{
		MaxSize:      maxSize,
		AllowedTypes: allowedTypes,
		MagicNumbers: map[string][]byte{
			"image/png":  {0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			"image/jpeg": {0xFF, 0xD8, 0xFF},
			"image/gif":  {0x47, 0x49, 0x46, 0x38},
			"image/webp": {0x52, 0x49, 0x46, 0x46}, // "RIFF"
		},
	}
}

// Validate 验证文件
func (fv *FileValidator) Validate(file *multipart.FileHeader) error {
	// 验证文件大小
	if fv.MaxSize > 0 && file.Size > fv.MaxSize {
		return ErrRequestTooLarge
	}

	// 验证文件类型（通过魔数）
	if len(fv.AllowedTypes) > 0 {
		if err := fv.validateMagicNumber(file); err != nil {
			return err
		}
	}

	return nil
}

// validateMagicNumber 验证文件魔数（优化：使用对象池）
func (fv *FileValidator) validateMagicNumber(fileHeader *multipart.FileHeader) error {
	file, err := fileHeader.Open()
	if err != nil {
		return WrapError(err, "无法打开文件")
	}
	defer file.Close()

	// 从对象池获取buffer（性能优化）
	// 获取指针类型，避免 interface{} 转换时的堆分配
	bufPtr := magicNumberBufferPool.Get().(*[]byte)
	defer magicNumberBufferPool.Put(bufPtr) // 直接传递指针，避免 SA6002 警告
	buf := *bufPtr                          // 解引用获取实际的 []byte

	n, err := io.ReadFull(file, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return WrapError(err, "读取文件失败")
	}
	buf = buf[:n]

	// 检查是否匹配任何允许的类型
	for _, allowedType := range fv.AllowedTypes {
		magicNumber, exists := fv.MagicNumbers[allowedType]
		if !exists {
			continue
		}

		if bytes.HasPrefix(buf, magicNumber) {
			return nil
		}

		// WebP特殊处理：需要检查WEBP标识
		if allowedType == "image/webp" && len(buf) >= 12 {
			if bytes.HasPrefix(buf, []byte{0x52, 0x49, 0x46, 0x46}) &&
				bytes.HasPrefix(buf[8:], []byte{0x57, 0x45, 0x42, 0x50}) {
				return nil
			}
		}
	}

	return errors.New("不支持的文件类型")
}

// EncodeFileName 编码文件名以符合RFC 5987规范
// 用于Content-Disposition响应头，支持中文和特殊字符
// 性能优化：使用strings.Builder减少内存分配
func EncodeFileName(filename string) string {
	// 快速检查是否只包含ASCII字符（性能优化：早期退出）
	hasNonASCII := false
	for i := 0; i < len(filename); i++ {
		if filename[i] > unicode.MaxASCII {
			hasNonASCII = true
			break
		}
	}

	var sb strings.Builder
	sb.WriteString("attachment; filename=\"")

	// 如果是纯ASCII，使用简单的格式
	if !hasNonASCII {
		// 转义引号和反斜杠（性能优化：逐字符处理，避免多次ReplaceAll）
		for i := 0; i < len(filename); i++ {
			c := filename[i]
			if c == '\\' || c == '"' {
				sb.WriteByte('\\')
			}
			sb.WriteByte(c)
		}
		sb.WriteByte('"')
		return sb.String()
	}

	// 包含非ASCII字符，使用RFC 5987格式
	// 生成fallback文件名（提取扩展名）
	if idx := strings.LastIndex(filename, "."); idx != -1 {
		sb.WriteString("download")
		sb.WriteString(filename[idx:])
	} else {
		sb.WriteString("download")
	}
	sb.WriteString("\"; filename*=UTF-8''")

	// URL编码文件名（性能优化：直接写入builder）
	for _, r := range filename {
		if r <= unicode.MaxASCII && isUnreserved(byte(r)) {
			sb.WriteRune(r)
		} else {
			// 对非ASCII字符进行URL编码
			for _, b := range []byte(string(r)) {
				sb.WriteByte('%')
				sb.WriteByte(upperhex[b>>4])
				sb.WriteByte(upperhex[b&15])
			}
		}
	}

	return sb.String()
}

// isUnreserved 检查字符是否是URL unreserved字符（性能优化：避免使用url.QueryEscape）
func isUnreserved(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
		c == '-' || c == '_' || c == '.' || c == '~'
}

const upperhex = "0123456789ABCDEF"
