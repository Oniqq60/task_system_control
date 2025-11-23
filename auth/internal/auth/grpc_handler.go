package auth

import (
	"context"
	"errors"
	"time"

	pb "github.com/Oniqq60/task_system_control/gen/proto/auth/v1"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GrpcHandler struct {
	pb.UnimplementedAuthServiceServer
	service   UserService
	jwtSecret []byte
	rdb       *redis.Client
}

// NewGrpcHandler создаёт новый gRPC handler
func NewGrpcHandler(service UserService, jwtSecret []byte, rdb *redis.Client) *GrpcHandler {
	return &GrpcHandler{
		service:   service,
		jwtSecret: jwtSecret,
		rdb:       rdb,
	}
}

// Register создаёт нового пользователя
func (h *GrpcHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password required")
	}

	role := RoleEmployee
	if req.Role != "" {
		switch Role(req.Role) {
		case RoleAdmin:
			role = RoleAdmin
		case RoleEmployee:
			role = RoleEmployee
		default:
			return nil, status.Error(codes.InvalidArgument, "invalid role")
		}
	}

	user := Users{
		ID:            uuid.New(),
		Name:          req.Name,
		Email:         req.Email,
		Password_hash: req.Password,
		Role:          role,
		Created_at:    time.Now(),
	}
	if req.ManagerId != "" {
		managerID, err := uuid.Parse(req.ManagerId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid manager_id")
		}
		if role == RoleAdmin {
			return nil, status.Error(codes.InvalidArgument, "admin cannot have manager")
		}
		user.ManagerID = &managerID
	}

	if err := h.service.Register(ctx, user); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.RegisterResponse{
		Id:    user.ID.String(),
		Email: user.Email,
		Name:  user.Name,
		Role:  string(user.Role),
	}, nil
}

// Login аутентифицирует пользователя и возвращает JWT токен
func (h *GrpcHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password required")
	}

	key := "auth:login:attempts:grpc:" + req.Email
	if cnt, err := h.rdb.Get(ctx, key).Int64(); err == nil && cnt >= 5 {
		return nil, status.Error(codes.ResourceExhausted, "too many attempts")
	}

	user, claims, err := h.service.Login(ctx, req.Email, req.Password)
	if err != nil {

		val, _ := h.rdb.Incr(ctx, key).Result()
		if val == 1 {
			_ = h.rdb.Expire(ctx, key, 10*time.Minute).Err()
		}
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	_ = h.rdb.Del(ctx, key).Err()

	token, err := SignToken(claims, h.jwtSecret)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to sign token")
	}

	expiresIn := int64(0)
	if claims.ExpiresAt != nil {
		expiresIn = claims.ExpiresAt.Time.Unix() - time.Now().Unix()
	}

	return &pb.LoginResponse{
		AccessToken: token,
		ExpiresIn:   expiresIn,
		UserId:      user.ID.String(),
		Role:        string(user.Role),
	}, nil
}

// ValidateToken проверяет валидность JWT токена
func (h *GrpcHandler) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	if req.Token == "" {
		return &pb.ValidateTokenResponse{
			Valid: false,
			Error: "token is required",
		}, nil
	}

	claims, err := ParseToken(req.Token, h.jwtSecret)
	if err != nil {
		return &pb.ValidateTokenResponse{
			Valid: false,
			Error: err.Error(),
		}, nil
	}

	if claims.ID == "" {
		return &pb.ValidateTokenResponse{
			Valid: false,
			Error: "token missing id",
		}, nil
	}

	if h.rdb != nil {
		key := tokenBlacklistPrefix + claims.ID
		exists, err := h.rdb.Exists(ctx, key).Result()
		if err != nil {
			return &pb.ValidateTokenResponse{
				Valid: false,
				Error: "token validation failed",
			}, nil
		}
		if exists == 1 {
			return &pb.ValidateTokenResponse{
				Valid: false,
				Error: "token revoked",
			}, nil
		}
	}

	return &pb.ValidateTokenResponse{
		Valid:  true,
		UserId: claims.UserID.String(),
		Role:   string(claims.Role),
	}, nil
}

// GetManager возвращает manager_id для сотрудника, если он назначен.
func (h *GrpcHandler) GetManager(ctx context.Context, req *pb.GetManagerRequest) (*pb.GetManagerResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	managerID, found, err := h.service.GetManager(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "failed to fetch manager")
	}

	resp := &pb.GetManagerResponse{
		Found: found,
	}
	if found {
		resp.ManagerId = managerID.String()
	}
	return resp, nil
}
