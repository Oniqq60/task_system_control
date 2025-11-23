package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserService interface {
	Register(ctx context.Context, u Users) error
	Login(ctx context.Context, email string, password string) (Users, Claims, error)
	Profile(ctx context.Context, userID uuid.UUID) (Users, error)
	GetManager(ctx context.Context, userID uuid.UUID) (uuid.UUID, bool, error)
}

var ErrUserNotFound = errors.New("user not found")

type userService struct {
	repo          UserRepository
	jwtSecret     []byte
	jwtTTLSeconds int64
}

func NewUserService(repo UserRepository, jwtSecret []byte, jwtTTLSeconds int64) UserService {
	return &userService{
		repo:          repo,
		jwtSecret:     jwtSecret,
		jwtTTLSeconds: jwtTTLSeconds,
	}
}

func (r *userService) Register(ctx context.Context, u Users) error {

	hashed, err := HashPassword(u.Password_hash)
	if err != nil {
		return err
	}
	u.Password_hash = hashed
	return r.repo.RegisterUser(ctx, u)
}

func (r *userService) Login(ctx context.Context, email string, password string) (Users, Claims, error) {
	user, _, err := r.repo.LoginUser(ctx, email, password)
	if err != nil {
		return Users{}, Claims{}, err
	}
	claims := BuildJWTClaims(user, r.jwtTTLSeconds)
	return user, claims, nil
}

func (r *userService) Profile(ctx context.Context, userID uuid.UUID) (Users, error) {
	user, err := r.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Users{}, ErrUserNotFound
		}
		return Users{}, err
	}
	user.Password_hash = ""
	return user, nil
}

func (r *userService) GetManager(ctx context.Context, userID uuid.UUID) (uuid.UUID, bool, error) {
	user, err := r.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.UUID{}, false, ErrUserNotFound
		}
		return uuid.UUID{}, false, err
	}

	if user.ManagerID == nil {
		return uuid.UUID{}, false, nil
	}

	return *user.ManagerID, true, nil
}
