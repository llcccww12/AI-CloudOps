/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package utils

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

var allowedCommentAttachmentExt = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".pdf":  "application/pdf",
	".txt":  "text/plain",
	".log":  "text/plain",
	".md":   "text/markdown",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".zip":  "application/zip",
}

func GetWorkorderAttachmentDir() string {
	dir := strings.TrimSpace(viper.GetString("workorder.attachment_dir"))
	if dir == "" {
		return "./data/workorder/attachments"
	}
	return dir
}

func GetWorkorderAttachmentMaxSizeBytes() int64 {
	mb := viper.GetInt("workorder.attachment_max_size_mb")
	if mb <= 0 {
		mb = 10
	}
	return int64(mb) * 1024 * 1024
}

func GetWorkorderAttachmentMaxCount() int {
	count := viper.GetInt("workorder.attachment_max_count")
	if count <= 0 {
		return 5
	}
	return count
}

func ValidateCommentAttachmentFileName(fileName string) (ext string, contentType string, err error) {
	base := filepath.Base(fileName)
	if base == "." || base == ".." || base == "" {
		return "", "", fmt.Errorf("文件名无效")
	}
	ext = strings.ToLower(filepath.Ext(base))
	contentType, ok := allowedCommentAttachmentExt[ext]
	if !ok {
		return "", "", fmt.Errorf("不支持的文件类型: %s", ext)
	}
	return ext, contentType, nil
}

func SanitizeAttachmentFileName(fileName string) string {
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

func IsImageContentType(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}
