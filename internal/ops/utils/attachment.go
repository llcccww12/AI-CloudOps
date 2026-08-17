package utils

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

var allowedOpsAttachmentExt = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".pdf":  "application/pdf",
	".txt":  "text/plain",
	".md":   "text/markdown",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".ppt":  "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".zip":  "application/zip",
}

func GetOpsAttachmentDir() string {
	dir := strings.TrimSpace(viper.GetString("ops.attachment_dir"))
	if dir == "" {
		return "./data/ops/attachments"
	}
	return dir
}

func GetOpsAttachmentMaxSizeBytes() int64 {
	mb := viper.GetInt("ops.attachment_max_size_mb")
	if mb <= 0 {
		mb = 20
	}
	return int64(mb) * 1024 * 1024
}

func GetOpsAttachmentMaxCount() int {
	count := viper.GetInt("ops.attachment_max_count")
	if count <= 0 {
		return 20
	}
	return count
}

func ValidateOpsAttachmentFileName(fileName string) (ext string, contentType string, err error) {
	base := filepath.Base(fileName)
	if base == "." || base == ".." || base == "" {
		return "", "", fmt.Errorf("文件名无效")
	}
	ext = strings.ToLower(filepath.Ext(base))
	contentType, ok := allowedOpsAttachmentExt[ext]
	if !ok {
		return "", "", fmt.Errorf("不支持的文件类型: %s", ext)
	}
	return ext, contentType, nil
}

func SanitizeOpsAttachmentFileName(fileName string) string {
	base := filepath.Base(fileName)
	base = strings.ReplaceAll(base, "..", "")
	base = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '.' || r == '-' || r == '_' || r == ' ':
			return r
		default:
			return '_'
		}
	}, base)
	if strings.TrimSpace(base) == "" {
		return "file"
	}
	return base
}
