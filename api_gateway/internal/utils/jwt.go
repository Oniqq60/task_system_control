package utils

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrEmptyToken    = errors.New("authorization token is required")
	ErrInvalidToken  = errors.New("token is invalid")
	ErrExpiredToken  = errors.New("token is expired")
	ErrWeakSecretKey = errors.New("jwt secret must be at least 32 bytes")
)

type Claims struct {
	UserID    string
	Role      string
	ExpiresAt time.Time
	IssuedAt  time.Time
}

type Verifier struct {
	secret []byte
}

func NewVerifier(secret string) (*Verifier, error) {
	if len(secret) < 32 {
		return nil, ErrWeakSecretKey
	}
	return &Verifier{secret: []byte(secret)}, nil
}

func (v *Verifier) ParseToken(ctx context.Context, token string) (Claims, error) {
	raw := strings.TrimSpace(token)
	if raw == "" {
		return Claims{}, ErrEmptyToken
	}

	type jwtClaims struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
		jwt.RegisteredClaims
	}

	parsedClaims := &jwtClaims{}

	parsedToken, err := jwt.ParseWithClaims(raw, parsedClaims, func(token *jwt.Token) (interface{}, error) {
		return v.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !parsedToken.Valid {
		return Claims{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims := Claims{
		UserID: parsedClaims.UserID,
		Role:   parsedClaims.Role,
	}
	if parsedClaims.ExpiresAt != nil {
		claims.ExpiresAt = parsedClaims.ExpiresAt.Time
	}
	if parsedClaims.IssuedAt != nil {
		claims.IssuedAt = parsedClaims.IssuedAt.Time
	}

	return claims, v.ValidateClaims(ctx, claims)
}

func (v *Verifier) ValidateClaims(_ context.Context, claims Claims) error {
	if claims.UserID == "" {
		return fmt.Errorf("%w: user_id missing", ErrInvalidToken)
	}
	if claims.ExpiresAt.IsZero() {
		return fmt.Errorf("%w: exp missing", ErrInvalidToken)
	}
	if time.Now().After(claims.ExpiresAt) {
		return ErrExpiredToken
	}
	return nil
}
