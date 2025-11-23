package services

import (
	"context"
	"fmt"
	"log"
	"time"

	authpb "github.com/Oniqq60/task_system_control/gen/proto/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AuthService struct {
	conn   *grpc.ClientConn
	client authpb.AuthServiceClient
}

func NewAuthService(target string) (*AuthService, error) {
	if target == "" {
		return nil, fmt.Errorf("auth grpc target is required")
	}

	var conn *grpc.ClientConn
	var err error

	// Пытаемся подключиться с повторами
	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		conn, err = grpc.DialContext(ctx, target,
			grpc.WithBlock(),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		cancel()

		if err == nil {
			break
		}

		log.Printf("Failed to connect to auth service: %v. Retrying in 3s...", err)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth service after retries: %w", err)
	}

	return &AuthService{
		conn:   conn,
		client: authpb.NewAuthServiceClient(conn),
	}, nil
}

func (s *AuthService) Close() error {
	if s == nil || s.conn == nil {
		return nil
	}
	return s.conn.Close()
}

func (s *AuthService) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.RegisterResponse, error) {
	return s.client.Register(ctx, req)
}

func (s *AuthService) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	return s.client.Login(ctx, req)
}

func (s *AuthService) ValidateToken(ctx context.Context, req *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
	return s.client.ValidateToken(ctx, req)
}
