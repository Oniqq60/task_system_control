package document

import (
	"context"
	"errors"
	"strings"

	pb "github.com/Oniqq60/task_system_control/gen/proto/document"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type GrpcHandler struct {
	pb.UnimplementedDocumentServiceServer
	service Service
	maxSize int64
	auth    Authorizer
}

var (
	errUnauthorized      = errors.New("unauthorized")
	tokenBlacklistPrefix = "auth:token:blacklist:"
)

type authClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type Authorizer interface {
	Authorize(ctx context.Context, token string) (Requester, error)
}

func NewGrpcHandler(service Service, maxSize int64, auth Authorizer) *GrpcHandler {
	return &GrpcHandler{
		service: service,
		maxSize: maxSize,
		auth:    auth,
	}
}

func (h *GrpcHandler) AddDocument(ctx context.Context, req *pb.AddDocumentRequest) (*pb.AddDocumentResponse, error) {
	requester, err := h.authorize(ctx)
	if err != nil {
		return nil, err
	}

	if requester.Role != RoleAdmin && requester.UserID != req.OwnerId {
		return nil, status.Error(codes.PermissionDenied, "forbidden")
	}

	content := req.GetFileContent()
	if len(content) == 0 {
		return nil, status.Error(codes.InvalidArgument, "empty file content")
	}

	input := AddDocumentInput{
		TaskID:      req.TaskId,
		OwnerID:     req.OwnerId,
		Filename:    req.Filename,
		ContentType: req.ContentType,
		Tags:        req.Tags,
		Content:     content,
		MaxSize:     h.maxSize,
	}

	metadata, _, err := h.service.AddDocument(ctx, input)
	if err != nil {
		return nil, handleServiceErr(err)
	}

	return &pb.AddDocumentResponse{
		Id:          metadata.ID.Hex(),
		Filename:    metadata.Filename,
		ContentType: metadata.ContentType,
		Size:        metadata.Size,
		TaskId:      metadata.TaskID,
		OwnerId:     metadata.OwnerID,
		Tags:        metadata.Tags,
		UploadedAt:  metadata.UploadedAt.Unix(),
	}, nil
}

func (h *GrpcHandler) DeleteDocument(ctx context.Context, req *pb.DeleteDocumentRequest) (*pb.DeleteDocumentResponse, error) {
	requester, err := h.authorize(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.service.DeleteDocument(ctx, req.GetId(), requester); err != nil {
		return nil, handleServiceErr(err)
	}
	return &pb.DeleteDocumentResponse{Success: true}, nil
}

func (h *GrpcHandler) GetDocument(ctx context.Context, req *pb.GetDocumentRequest) (*pb.GetDocumentResponse, error) {
	requester, err := h.authorize(ctx)
	if err != nil {
		return nil, err
	}

	metadata, content, err := h.service.GetDocument(ctx, req.GetId(), requester)
	if err != nil {
		return nil, handleServiceErr(err)
	}

	return &pb.GetDocumentResponse{
		Id:          metadata.ID.Hex(),
		Filename:    metadata.Filename,
		ContentType: metadata.ContentType,
		Size:        metadata.Size,
		FileContent: content,
		TaskId:      metadata.TaskID,
		OwnerId:     metadata.OwnerID,
		Tags:        metadata.Tags,
		UploadedAt:  metadata.UploadedAt.Unix(),
	}, nil
}

func (h *GrpcHandler) GetDocumentsByTask(ctx context.Context, req *pb.GetDocumentsByTaskRequest) (*pb.GetDocumentsByTaskResponse, error) {
	requester, err := h.authorize(ctx)
	if err != nil {
		return nil, err
	}

	docs, err := h.service.GetDocumentsByTask(ctx, req.GetTaskId(), requester)
	if err != nil {
		return nil, handleServiceErr(err)
	}

	return &pb.GetDocumentsByTaskResponse{
		Documents: mapDocs(docs),
	}, nil
}

func (h *GrpcHandler) GetDocumentsByOwner(ctx context.Context, req *pb.GetDocumentsByOwnerRequest) (*pb.GetDocumentsByOwnerResponse, error) {
	requester, err := h.authorize(ctx)
	if err != nil {
		return nil, err
	}

	docs, err := h.service.GetDocumentsByOwner(ctx, req.GetOwnerId(), requester)
	if err != nil {
		return nil, handleServiceErr(err)
	}

	return &pb.GetDocumentsByOwnerResponse{
		Documents: mapDocs(docs),
	}, nil
}

func NewAuthorizer(jwtSecret []byte, redis *redis.Client) Authorizer {
	return &metadataAuthorizer{
		jwtSecret: jwtSecret,
		redis:     redis,
	}
}

func (a *metadataAuthorizer) Authorize(ctx context.Context, token string) (Requester, error) {
	claims := &authClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return a.jwtSecret, nil
	})
	if err != nil || !parsed.Valid {
		return Requester{}, errUnauthorized
	}

	if claims.ID == "" || claims.UserID == "" {
		return Requester{}, errUnauthorized
	}

	if a.redis != nil {
		key := tokenBlacklistPrefix + claims.ID
		exists, redisErr := a.redis.Exists(ctx, key).Result()
		if redisErr != nil {
			return Requester{}, redisErr
		}
		if exists > 0 {
			return Requester{}, errUnauthorized
		}
	}

	return Requester{
		UserID: claims.UserID,
		Role:   Role(strings.ToLower(claims.Role)),
	}, nil
}

type metadataAuthorizer struct {
	jwtSecret []byte
	redis     *redis.Client
}

func (h *GrpcHandler) authorize(ctx context.Context) (Requester, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return Requester{}, errUnauthorized
	}

	tokens := md.Get("authorization")
	if len(tokens) == 0 {
		return Requester{}, errUnauthorized
	}

	token := strings.TrimPrefix(tokens[0], "Bearer ")
	return h.auth.Authorize(ctx, token)
}

func mapDocs(metas []Metadata) []*pb.Document {
	pbDocs := make([]*pb.Document, 0, len(metas))
	for _, m := range metas {
		pbDocs = append(pbDocs, &pb.Document{
			Id:          m.ID.Hex(),
			Filename:    m.Filename,
			ContentType: m.ContentType,
			Size:        m.Size,
			TaskId:      m.TaskID,
			OwnerId:     m.OwnerID,
			Tags:        m.Tags,
			UploadedAt:  m.UploadedAt.Unix(),
		})
	}
	return pbDocs
}

func handleServiceErr(err error) error {
	if errors.Is(err, ErrForbidden) {
		return status.Error(codes.PermissionDenied, err.Error())
	}
	if errors.Is(err, ErrFileTooLarge) {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if errors.Is(err, ErrInvalidContentType) {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if errors.Is(err, ErrInvalidDocumentID) {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if errors.Is(err, ErrInvalidTaskID) {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if errors.Is(err, ErrInvalidOwnerID) {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if errors.Is(err, ErrEmptyFilename) {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if errors.Is(err, ErrEmptyContent) {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	return status.Error(codes.Internal, err.Error())
}
