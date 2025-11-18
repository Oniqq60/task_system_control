package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func BuildJWTClaims(user Users, ttlSeconds int64) Claims {
	now := time.Now()
	rc := jwt.RegisteredClaims{
		Subject:   user.ID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(ttlSeconds) * time.Second)),
		ID:        uuid.NewString(),
	}
	return Claims{
		UserID:           user.ID,
		Role:             user.Role,
		RegisteredClaims: rc,
	}
}

func SignToken(claims Claims, secret []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ParseToken парсит и валидирует JWT токен
func ParseToken(tokenString string, secret []byte) (Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return secret, nil
	})

	if err != nil {
		return Claims{}, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// Если UserID не заполнен, но Subject есть, извлекаем UserID из Subject
		if claims.UserID == uuid.Nil && claims.Subject != "" {
			userID, err := uuid.Parse(claims.Subject)
			if err == nil {
				claims.UserID = userID
			}
		}
		return *claims, nil
	}

	return Claims{}, jwt.ErrSignatureInvalid
}
