package document

import (
	"errors"
	"mime"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

var (
	ErrInvalidFilename = errors.New("invalid filename")
	ErrInvalidFileType = errors.New("file type not allowed")
	ErrPathTraversal   = errors.New("path traversal detected")
	// ErrInvalidContentType определен в service.go
)

// Разрешенные MIME типы для загрузки файлов
var allowedMimeTypes = map[string]bool{
	"application/pdf":    true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
	"application/vnd.ms-powerpoint":                                             true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	"text/plain":                   true,
	"text/csv":                     true,
	"image/jpeg":                   true,
	"image/png":                    true,
	"image/gif":                    true,
	"image/webp":                   true,
	"application/zip":              true,
	"application/x-zip-compressed": true,
	"application/json":             true,
	"application/xml":              true,
	"text/xml":                     true,
}

// Разрешенные расширения файлов
var allowedExtensions = map[string]bool{
	".pdf":  true,
	".doc":  true,
	".docx": true,
	".xls":  true,
	".xlsx": true,
	".ppt":  true,
	".pptx": true,
	".txt":  true,
	".csv":  true,
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
	".zip":  true,
	".json": true,
	".xml":  true,
}

// ValidateFilename проверяет имя файла на безопасность
func ValidateFilename(filename string) error {
	if filename == "" {
		return ErrInvalidFilename
	}

	// Проверка на path traversal
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return ErrPathTraversal
	}

	// Проверка длины
	if len(filename) > 255 {
		return ErrInvalidFilename
	}

	// Проверка на наличие только допустимых символов
	if !utf8.ValidString(filename) {
		return ErrInvalidFilename
	}

	// Проверка расширения
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return ErrInvalidFilename
	}
	if !allowedExtensions[ext] {
		return ErrInvalidFileType
	}

	// Базовое имя файла (без расширения)
	baseName := strings.TrimSuffix(filename, ext)
	if len(baseName) == 0 {
		return ErrInvalidFilename
	}

	return nil
}

// ValidateContentType проверяет MIME тип файла
func ValidateContentType(contentType string) error {
	if contentType == "" {
		return ErrInvalidContentType // Используем ошибку из service.go
	}

	// Парсим MIME тип (может содержать параметры, например "text/plain; charset=utf-8")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ErrInvalidContentType // Используем ошибку из service.go
	}

	// Проверяем, разрешен ли этот тип
	if !allowedMimeTypes[mediaType] {
		return ErrInvalidContentType // Используем ошибку из service.go
	}

	return nil
}

// SanitizeFilename очищает имя файла от опасных символов
func SanitizeFilename(filename string) string {
	// Удаляем path traversal попытки
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
	// Экранируем кавычки и обратные слеши для Content-Disposition
	filename = strings.ReplaceAll(filename, `"`, `\"`)
	filename = strings.ReplaceAll(filename, `\`, `\\`)
	return filename
}
