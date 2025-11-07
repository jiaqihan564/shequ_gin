package utils

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"

	"github.com/nfnt/resize"
)

// ImageProcessor 图片处理器
type ImageProcessor struct {
	MaxWidth       uint   // 最大宽度
	MaxHeight      uint   // 最大高度
	JpegQuality    int    // JPEG质量（1-100）
	OutputFormat   string // 输出格式：png, jpeg
	EnableAutoSize bool   // 是否自动调整尺寸
}

// ImageProcessResult 图片处理结果
type ImageProcessResult struct {
	Data           *bytes.Buffer // 处理后的图片数据
	Width          int           // 图片宽度
	Height         int           // 图片高度
	OriginalWidth  int           // 原始宽度
	OriginalHeight int           // 原始高度
	Format         string        // 格式
	Size           int64         // 文件大小（字节）
	Resized        bool          // 是否被缩放
}

// NewImageProcessor 创建图片处理器
func NewImageProcessor(maxWidth, maxHeight uint, jpegQuality int, outputFormat string) *ImageProcessor {
	if jpegQuality < 1 || jpegQuality > 100 {
		jpegQuality = 85 // 默认质量
	}
	if outputFormat != "png" && outputFormat != "jpeg" {
		outputFormat = "png" // 默认PNG
	}
	return &ImageProcessor{
		MaxWidth:       maxWidth,
		MaxHeight:      maxHeight,
		JpegQuality:    jpegQuality,
		OutputFormat:   outputFormat,
		EnableAutoSize: true,
	}
}

// ProcessAvatar 处理头像图片（缩放+压缩）
func (p *ImageProcessor) ProcessAvatar(fileHeader *multipart.FileHeader) (*ImageProcessResult, error) {
	// 打开文件
	file, err := fileHeader.Open()
	if err != nil {
		return nil, WrapError(err, "无法打开图片文件")
	}
	defer file.Close()

	// 解码图片
	img, format, err := image.Decode(file)
	if err != nil {
		return nil, WrapError(err, "无法解析图片格式")
	}

	// 获取原始尺寸
	bounds := img.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	result := &ImageProcessResult{
		OriginalWidth:  origWidth,
		OriginalHeight: origHeight,
		Format:         format,
		Resized:        false,
	}

	// 检查是否需要缩放
	needResize := p.EnableAutoSize && (origWidth > int(p.MaxWidth) || origHeight > int(p.MaxHeight))

	var processedImg image.Image
	if needResize {
		// 缩放图片（保持宽高比）
		processedImg = resize.Thumbnail(p.MaxWidth, p.MaxHeight, img, resize.Lanczos3)
		result.Resized = true
	} else {
		processedImg = img
	}

	// 获取处理后的尺寸
	processedBounds := processedImg.Bounds()
	result.Width = processedBounds.Dx()
	result.Height = processedBounds.Dy()

	// 编码输出
	buf := new(bytes.Buffer)
	switch p.OutputFormat {
	case "jpeg":
		err = jpeg.Encode(buf, processedImg, &jpeg.Options{Quality: p.JpegQuality})
		result.Format = "jpeg"
	case "png":
		// PNG编码（使用默认压缩）
		encoder := png.Encoder{CompressionLevel: png.DefaultCompression}
		err = encoder.Encode(buf, processedImg)
		result.Format = "png"
	default:
		return nil, errors.New("不支持的输出格式")
	}

	if err != nil {
		return nil, WrapError(err, "图片编码失败")
	}

	result.Data = buf
	result.Size = int64(buf.Len())

	return result, nil
}

// ValidateImageDimensions 验证图片尺寸（不做处理，只检查）
func ValidateImageDimensions(file io.Reader, maxWidth, maxHeight int) (width, height int, err error) {
	// 解码图片配置（不加载完整图片数据，更快）
	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, WrapError(err, "无法读取图片尺寸")
	}

	width = config.Width
	height = config.Height

	if maxWidth > 0 && width > maxWidth {
		return width, height, fmt.Errorf("图片宽度超过限制（最大：%dpx，当前：%dpx）", maxWidth, width)
	}

	if maxHeight > 0 && height > maxHeight {
		return width, height, fmt.Errorf("图片高度超过限制（最大：%dpx，当前：%dpx）", maxHeight, height)
	}

	return width, height, nil
}

// GetImageDimensions 获取图片尺寸（从multipart.FileHeader）
func GetImageDimensions(fileHeader *multipart.FileHeader) (width, height int, err error) {
	file, err := fileHeader.Open()
	if err != nil {
		return 0, 0, WrapError(err, "无法打开文件")
	}
	defer file.Close()

	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, WrapError(err, "无法读取图片尺寸")
	}

	return config.Width, config.Height, nil
}
