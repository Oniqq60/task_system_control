package auth

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

var (
	ErrInvalidEmail    = errors.New("invalid email format")
	ErrInvalidPassword = errors.New("password must be at least 8 characters long")
	ErrInvalidName     = errors.New("name must be between 1 and 100 characters")
)

// ValidateEmail проверяет формат email
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return ErrInvalidEmail
	}
	if len(email) > 254 {
		return ErrInvalidEmail
	}
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}
	return nil
}

// ValidatePassword проверяет требования к паролю
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrInvalidPassword
	}
	if len(password) > 128 {
		return ErrInvalidPassword
	}

	hasLetter := false
	hasDigit := false
	for _, r := range password {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
		if hasLetter && hasDigit {
			break
		}
	}
	if !hasLetter || !hasDigit {
		return ErrInvalidPassword
	}
	return nil
}

// ValidateName проверяет имя пользователя
func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if len(name) == 0 || len(name) > 100 {
		return ErrInvalidName
	}

	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != ' ' && r != '-' && r != '_' {
			return ErrInvalidName
		}
	}
	return nil
}

// SanitizeString удаляет опасные символы из строки
func SanitizeString(s string) string {
	s = strings.TrimSpace(s)
	// Удаляем управляющие символы
	var builder strings.Builder
	for _, r := range s {
		if r >= 32 && r != 127 {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}
