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
