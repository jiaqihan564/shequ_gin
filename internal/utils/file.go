package utils

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
)

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

// validateMagicNumber 验证文件魔数
func (fv *FileValidator) validateMagicNumber(fileHeader *multipart.FileHeader) error {
	file, err := fileHeader.Open()
	if err != nil {
		return WrapError(err, "无法打开文件")
	}
	defer file.Close()

	// 读取前16字节用于魔数验证
	buf := make([]byte, 16)
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

// IsPNG 检查是否是PNG文件（通过魔数）
func IsPNG(file *multipart.FileHeader) bool {
	f, err := file.Open()
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 8)
	n, err := io.ReadFull(f, buf)
	if err != nil || n != 8 {
		return false
	}

	// PNG magic number: 89 50 4E 47 0D 0A 1A 0A
	return buf[0] == 0x89 && buf[1] == 0x50 && buf[2] == 0x4E && buf[3] == 0x47 &&
		buf[4] == 0x0D && buf[5] == 0x0A && buf[6] == 0x1A && buf[7] == 0x0A
}

// IsJPEG 检查是否是JPEG文件（通过魔数）
func IsJPEG(file *multipart.FileHeader) bool {
	f, err := file.Open()
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 3)
	n, err := io.ReadFull(f, buf)
	if err != nil || n != 3 {
		return false
	}

	// JPEG magic number: FF D8 FF
	return buf[0] == 0xFF && buf[1] == 0xD8 && buf[2] == 0xFF
}

// GetFileExtension 根据魔数获取文件扩展名
func GetFileExtension(file *multipart.FileHeader) string {
	if IsPNG(file) {
		return ".png"
	}
	if IsJPEG(file) {
		return ".jpg"
	}
	return ""
}
