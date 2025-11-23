package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleEmployee Role = "employee"
)

type Users struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	Email           string     `json:"email"`
	Password_hash   string     `json:"password_hash"`
	Role            Role       `json:"role"`
	ManagerID       *uuid.UUID `json:"manager_id" gorm:"type:uuid"`
	Completed_tasks int64      `json:"completed_tasks"`
	Created_at      time.Time  `json:"created_at"`
}

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Role   Role      `json:"role"`
	jwt.RegisteredClaims
}
