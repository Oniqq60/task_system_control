package document

import (
	"errors"
	"mime"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

var allowedExtensions = map[string]bool{
	".pdf":  true,
	".doc":  true,
	".docx": true,
	".txt":  true,
	".xls":  true,
	".xlsx": true,
	".png":  true,
	".jpg":  true,
	".jpeg": true,
}

var allowedMimeTypes = map[string]bool{
	"application/pdf":    true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"text/plain":               true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
	"image/png":  true,
	"image/jpeg": true,
}

var (
	ErrInvalidFilename = errors.New("invalid filename")
	ErrInvalidFileType = errors.New("file type not allowed")
	ErrPathTraversal   = errors.New("path traversal detected")
)

// ValidateFilename проверяет имя файла на безопасность
func ValidateFilename(filename string) error {
	if filename == "" {
		return ErrInvalidFilename
	}

	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return ErrPathTraversal
	}

	if len(filename) > 255 {
		return ErrInvalidFilename
	}

	if !utf8.ValidString(filename) {
		return ErrInvalidFilename
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return ErrInvalidFilename
	}
	if !allowedExtensions[ext] {
		return ErrInvalidFileType
	}

	baseName := strings.TrimSuffix(filename, ext)
	if len(baseName) == 0 {
		return ErrInvalidFilename
	}

	return nil
}

// ValidateContentType проверяет MIME тип файла
func ValidateContentType(contentType string) error {
	if contentType == "" {
		return ErrInvalidContentType
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ErrInvalidContentType
	}

	if !allowedMimeTypes[mediaType] {
		return ErrInvalidContentType
	}

	return nil
}

// SanitizeFilename очищает имя файла от опасных символов
func SanitizeFilename(filename string) string {

	filename = strings.ReplaceAll(filename, "..", "")
	filename = strings.ReplaceAll(filename, "/", "")
	filename = strings.ReplaceAll(filename, "\\", "")

	// Удаляем управляющие символы
	var builder strings.Builder
	for _, r := range filename {
		if r >= 32 && r != 127 {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

// EscapeFilename экранирует имя файла для использования в HTTP заголовках
func EscapeFilename(filename string) string {

	filename = strings.ReplaceAll(filename, `"`, `\"`)
	filename = strings.ReplaceAll(filename, `\`, `\\`)
	return filename
}
