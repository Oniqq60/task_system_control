package authclient

import (
	"context"
	"fmt"
	"time"

	pb "github.com/Oniqq60/task_system_control/gen/proto/auth/v1"
	"google.golang.org/grpc"
)

type Client struct {
	rpc     pb.AuthServiceClient
	timeout time.Duration
}

// New создаёт клиента. timeout задаёт максимальное время ожидания gRPC вызова.
func New(conn *grpc.ClientConn, timeout time.Duration) *Client {
	return &Client{
		rpc:     pb.NewAuthServiceClient(conn),
		timeout: timeout,
	}
}

// ResolveManager возвращает manager_id для userID или пустую строку, если менеджер не назначен.
func (c *Client) ResolveManager(ctx context.Context, userID string) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("userID is required")
	}

	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.rpc.GetManager(callCtx, &pb.GetManagerRequest{UserId: userID})
	if err != nil {
		return "", fmt.Errorf("auth.GetManager RPC failed: %w", err)
	}

	if !resp.GetFound() || resp.GetManagerId() == "" {
		return "", nil
	}

	return resp.GetManagerId(), nil
}
