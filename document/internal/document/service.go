package document

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleEmployee Role = "employee"
)

type Service interface {
	AddDocument(ctx context.Context, input AddDocumentInput) (Metadata, []byte, error)
	DeleteDocument(ctx context.Context, id string, requester Requester) error
	GetDocument(ctx context.Context, id string, requester Requester) (Metadata, []byte, error)
	GetDocumentsByTask(ctx context.Context, taskID string, requester Requester) ([]Metadata, error)
	GetDocumentsByOwner(ctx context.Context, ownerID string, requester Requester) ([]Metadata, error)
}

type AddDocumentInput struct {
	TaskID      string
	OwnerID     string
	Filename    string
	ContentType string
	Tags        []string
	Content     []byte
	MaxSize     int64
}

type Requester struct {
	UserID string
	Role   Role
}

var (
	ErrForbidden           = errors.New("forbidden")
	ErrFileTooLarge        = errors.New("file too large")
	ErrInvalidContentType  = errors.New("content type required")
	ErrInvalidDocumentID   = errors.New("invalid document id")
	ErrInvalidTaskID       = errors.New("invalid task id")
	ErrInvalidOwnerID      = errors.New("invalid owner id")
	ErrEmptyFilename       = errors.New("filename required")
	ErrEmptyContent        = errors.New("file content required")
	errMaxSizeNotSpecified = errors.New("max file size not specified")
)

type service struct {
	repo    Repository
	storage ObjectStorage
}

func NewService(repo Repository, storage ObjectStorage) Service {
	return &service{
		repo:    repo,
		storage: storage,
	}
}

func (s *service) AddDocument(ctx context.Context, input AddDocumentInput) (Metadata, []byte, error) {
	if input.MaxSize <= 0 {
		return Metadata{}, nil, errMaxSizeNotSpecified
	}
	if input.Filename == "" {
		return Metadata{}, nil, ErrEmptyFilename
	}
	if input.ContentType == "" {
		return Metadata{}, nil, ErrInvalidContentType
	}
	if len(input.Content) == 0 {
		return Metadata{}, nil, ErrEmptyContent
	}
	if int64(len(input.Content)) > input.MaxSize {
		return Metadata{}, nil, ErrFileTooLarge
	}
	if _, err := uuid.Parse(input.TaskID); err != nil {
		return Metadata{}, nil, ErrInvalidTaskID
	}
	if _, err := uuid.Parse(input.OwnerID); err != nil {
		return Metadata{}, nil, ErrInvalidOwnerID
	}

	saveCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	objectKey, checksum, size, err := s.storage.Save(saveCtx, input.Filename, input.ContentType, input.Content)
	if err != nil {
		return Metadata{}, nil, err
	}

	metadata := Metadata{
		TaskID:      input.TaskID,
		OwnerID:     input.OwnerID,
		Filename:    input.Filename,
		ContentType: input.ContentType,
		Size:        size,
		Tags:        input.Tags,
		MinioObject: objectKey,
		MinioBucket: s.storage.Bucket(),
		Checksum:    checksum,
	}

	insertCtx, cancelInsert := context.WithTimeout(ctx, 10*time.Second)
	defer cancelInsert()

	insertID, err := s.repo.Insert(insertCtx, metadata)
	if err != nil {
		_ = s.storage.Delete(context.Background(), objectKey)
		return Metadata{}, nil, err
	}
	metadata.ID = insertID

	return metadata, input.Content, nil
}

func (s *service) DeleteDocument(ctx context.Context, id string, requester Requester) error {
	objectID, err := parseObjectID(id)
	if err != nil {
		return ErrInvalidDocumentID
	}

	doc, err := s.repo.FindByID(ctx, objectID)
	if err != nil {
		return err
	}

	if !requester.CanManageDocument(doc) {
		return ErrForbidden
	}

	if err := s.repo.Delete(ctx, objectID); err != nil {
		return err
	}

	go func(objectKey string) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.storage.Delete(ctx, objectKey)
	}(doc.MinioObject)

	return nil
}

func (s *service) GetDocument(ctx context.Context, id string, requester Requester) (Metadata, []byte, error) {
	objectID, err := parseObjectID(id)
	if err != nil {
		return Metadata{}, nil, ErrInvalidDocumentID
	}

	doc, err := s.repo.FindByID(ctx, objectID)
	if err != nil {
		return Metadata{}, nil, err
	}

	if !requester.CanAccessDocument(doc) {
		return Metadata{}, nil, ErrForbidden
	}

	readCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	reader, _, err := s.storage.Get(readCtx, doc.MinioObject)
	if err != nil {
		return Metadata{}, nil, err
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return Metadata{}, nil, err
	}

	return doc, content, nil
}

func (s *service) GetDocumentsByTask(ctx context.Context, taskID string, requester Requester) ([]Metadata, error) {
	if _, err := uuid.Parse(taskID); err != nil {
		return nil, ErrInvalidTaskID
	}
	if requester.Role != RoleAdmin {
		return nil, ErrForbidden
	}
	return s.repo.FindByTask(ctx, taskID)
}

func (s *service) GetDocumentsByOwner(ctx context.Context, ownerID string, requester Requester) ([]Metadata, error) {
	if _, err := uuid.Parse(ownerID); err != nil {
		return nil, ErrInvalidOwnerID
	}
	if requester.Role != RoleAdmin && requester.UserID != ownerID {
		return nil, ErrForbidden
	}
	return s.repo.FindByOwner(ctx, ownerID)
}

func parseObjectID(id string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(id)
}

func (r Requester) CanManageDocument(doc Metadata) bool {
	if r.Role == RoleAdmin {
		return true
	}
	return r.UserID == doc.OwnerID
}

func (r Requester) CanAccessDocument(doc Metadata) bool {
	if r.Role == RoleAdmin {
		return true
	}
	return r.UserID == doc.OwnerID
}
