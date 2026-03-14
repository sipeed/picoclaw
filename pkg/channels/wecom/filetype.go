package wecom

import (
	"bytes"
	"strings"
)

// FileType 文件类型
type FileType string

const (
	FileTypeJPEG     FileType = "image/jpeg"
	FileTypePNG      FileType = "image/png"
	FileTypeGIF      FileType = "image/gif"
	FileTypeWebP     FileType = "image/webp"
	FileTypePDF      FileType = "application/pdf"
	FileTypeDOC      FileType = "application/msword"
	FileTypeDOCX     FileType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	FileTypeUnknown  FileType = "unknown"
)

// FileTypeInfo 文件类型信息
type FileTypeInfo struct {
	Type     FileType
	Ext      string
	MIMEType string
}

// fileTypeSignatures 文件类型签名（魔数）
var fileTypeSignatures = []struct {
	Signature []byte
	Type      FileType
	Ext       string
	MIMEType  string
}{
	// JPEG: FF D8 FF
	{[]byte{0xFF, 0xD8, 0xFF}, FileTypeJPEG, ".jpg", "image/jpeg"},
	// PNG: 89 50 4E 47
	{[]byte{0x89, 0x50, 0x4E, 0x47}, FileTypePNG, ".png", "image/png"},
	// GIF: 47 49 46 38
	{[]byte{0x47, 0x49, 0x46, 0x38}, FileTypeGIF, ".gif", "image/gif"},
	// WebP: 52 49 46 46 ... 57 45 42 50
	{[]byte{0x52, 0x49, 0x46, 0x46}, FileTypeWebP, ".webp", "image/webp"},
	// PDF: 25 50 44 46
	{[]byte{0x25, 0x50, 0x44, 0x46}, FileTypePDF, ".pdf", "application/pdf"},
	// DOC: D0 CF 11 E0 (OLE Compound Document)
	{[]byte{0xD0, 0xCF, 0x11, 0xE0}, FileTypeDOC, ".doc", "application/msword"},
	// DOCX: 50 4B 03 04 (ZIP格式，需要进一步检查)
	{[]byte{0x50, 0x4B, 0x03, 0x04}, FileTypeDOCX, ".docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
}

// DetectFileType 检测文件类型
func DetectFileType(data []byte) FileTypeInfo {
	if len(data) < 4 {
		return FileTypeInfo{Type: FileTypeUnknown, Ext: "", MIMEType: "application/octet-stream"}
	}

	for _, sig := range fileTypeSignatures {
		if len(data) >= len(sig.Signature) && bytes.HasPrefix(data, sig.Signature) {
			// 对于 WebP 需要额外检查
			if sig.Type == FileTypeWebP && len(data) >= 12 {
				// WebP 的签名在 8-11 字节位置
				if !bytes.Equal(data[8:12], []byte{0x57, 0x45, 0x42, 0x50}) {
					continue
				}
			}
			return FileTypeInfo{
				Type:     sig.Type,
				Ext:      sig.Ext,
				MIMEType: sig.MIMEType,
			}
		}
	}

	return FileTypeInfo{Type: FileTypeUnknown, Ext: "", MIMEType: "application/octet-stream"}
}

// IsImage 检查是否为图片
func IsImage(fileType FileType) bool {
	switch fileType {
	case FileTypeJPEG, FileTypePNG, FileTypeGIF, FileTypeWebP:
		return true
	default:
		return false
	}
}

// IsDocument 检查是否为文档
func IsDocument(fileType FileType) bool {
	switch fileType {
	case FileTypePDF, FileTypeDOC, FileTypeDOCX:
		return true
	default:
		return false
	}
}

// GetFileTypeByExt 根据扩展名获取文件类型
func GetFileTypeByExt(ext string) FileTypeInfo {
	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	switch ext {
	case ".jpg", ".jpeg":
		return FileTypeInfo{Type: FileTypeJPEG, Ext: ".jpg", MIMEType: "image/jpeg"}
	case ".png":
		return FileTypeInfo{Type: FileTypePNG, Ext: ".png", MIMEType: "image/png"}
	case ".gif":
		return FileTypeInfo{Type: FileTypeGIF, Ext: ".gif", MIMEType: "image/gif"}
	case ".webp":
		return FileTypeInfo{Type: FileTypeWebP, Ext: ".webp", MIMEType: "image/webp"}
	case ".pdf":
		return FileTypeInfo{Type: FileTypePDF, Ext: ".pdf", MIMEType: "application/pdf"}
	case ".doc":
		return FileTypeInfo{Type: FileTypeDOC, Ext: ".doc", MIMEType: "application/msword"}
	case ".docx":
		return FileTypeInfo{Type: FileTypeDOCX, Ext: ".docx", MIMEType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document"}
	default:
		return FileTypeInfo{Type: FileTypeUnknown, Ext: ext, MIMEType: "application/octet-stream"}
	}
}
