package auth

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository interface {
	RegisterUser(ctx context.Context, u Users) error
	LoginUser(ctx context.Context, email string, password string) (Users, Claims, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (Users, error)
}

type userRepository struct {
	db gorm.DB
}

func NewRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: *db}
}

func (s *userRepository) RegisterUser(ctx context.Context, u Users) error {

	return s.db.WithContext(ctx).Create(&u).Error
}

func (s *userRepository) LoginUser(ctx context.Context, email string, password string) (Users, Claims, error) {
	var user Users
	err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return Users{}, Claims{}, err
	}
	if err := CheckPassword(user.Password_hash, password); err != nil {
		return Users{}, Claims{}, err
	}

	return user, Claims{}, nil
}

func (s *userRepository) GetUserByID(ctx context.Context, id uuid.UUID) (Users, error) {
	var user Users
	err := s.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if err != nil {
		return Users{}, err
	}
	return user, nil
}
