package services

import (
	"context"
	"fmt"
	"time"

	documentpb "github.com/Oniqq60/task_system_control/gen/proto/document"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type DocumentService struct {
	conn   *grpc.ClientConn
	client documentpb.DocumentServiceClient
}

func NewDocumentService(target string) (*DocumentService, error) {
	if target == "" {
		return nil, fmt.Errorf("document grpc target is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, target,
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial document service: %w", err)
	}

	return &DocumentService{
		conn:   conn,
		client: documentpb.NewDocumentServiceClient(conn),
	}, nil
}

func (s *DocumentService) Close() error {
	if s == nil || s.conn == nil {
		return nil
	}
	return s.conn.Close()
}

// addAuthMetadata добавляет JWT токен в gRPC metadata из контекста
func (s *DocumentService) addAuthMetadata(ctx context.Context) context.Context {
	// Пытаемся получить токен из контекста
	if token, ok := ctx.Value("jwt_token").(string); ok && token != "" {
		md := metadata.New(map[string]string{
			"authorization": "Bearer " + token,
		})
		return metadata.NewOutgoingContext(ctx, md)
	}
	return ctx
}

func (s *DocumentService) AddDocument(ctx context.Context, req *documentpb.AddDocumentRequest) (*documentpb.AddDocumentResponse, error) {
	ctx = s.addAuthMetadata(ctx)
	return s.client.AddDocument(ctx, req)
}

func (s *DocumentService) DeleteDocument(ctx context.Context, req *documentpb.DeleteDocumentRequest) (*documentpb.DeleteDocumentResponse, error) {
	ctx = s.addAuthMetadata(ctx)
	return s.client.DeleteDocument(ctx, req)
}

func (s *DocumentService) GetDocument(ctx context.Context, req *documentpb.GetDocumentRequest) (*documentpb.GetDocumentResponse, error) {
	ctx = s.addAuthMetadata(ctx)
	return s.client.GetDocument(ctx, req)
}

func (s *DocumentService) GetDocumentsByTask(ctx context.Context, req *documentpb.GetDocumentsByTaskRequest) (*documentpb.GetDocumentsByTaskResponse, error) {
	ctx = s.addAuthMetadata(ctx)
	return s.client.GetDocumentsByTask(ctx, req)
}

func (s *DocumentService) GetDocumentsByOwner(ctx context.Context, req *documentpb.GetDocumentsByOwnerRequest) (*documentpb.GetDocumentsByOwnerResponse, error) {
	ctx = s.addAuthMetadata(ctx)
	return s.client.GetDocumentsByOwner(ctx, req)
}
