package document

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type minioStorage struct {
	client     *minio.Client
	bucketName string
}

func hashSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

type ObjectStorage interface {
	Save(ctx context.Context, filename string, contentType string, data []byte) (objectKey string, checksum string, size int64, err error)
	Delete(ctx context.Context, objectKey string) error
	Get(ctx context.Context, objectKey string) (io.ReadCloser, int64, error)
	Bucket() string
}

func NewMinioStorage(endpoint, accessKey, secretKey string, useSSL bool, bucket string) (ObjectStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, errBucket := client.BucketExists(ctx, bucket)
	if errBucket != nil {
		return nil, errBucket
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}

	return &minioStorage{
		client:     client,
		bucketName: bucket,
	}, nil
}

func (s *minioStorage) Save(ctx context.Context, filename string, contentType string, data []byte) (string, string, int64, error) {
	objectID := uuid.New().String()
	ext := strings.ToLower(filepath.Ext(filename))
	objectKey := fmt.Sprintf("documents/%s%s", objectID, ext)
	reader := bytes.NewReader(data)
	size := int64(len(data))

	checksum := hashSHA256(data)

	_, err := s.client.PutObject(ctx, s.bucketName, objectKey, reader, size, minio.PutObjectOptions{
		ContentType:  contentType,
		UserMetadata: map[string]string{"filename": filename},
	})
	if err != nil {
		return "", "", 0, err
	}

	return objectKey, checksum, size, nil
}

func (s *minioStorage) Delete(ctx context.Context, objectKey string) error {
	return s.client.RemoveObject(ctx, s.bucketName, objectKey, minio.RemoveObjectOptions{})
}

func (s *minioStorage) Get(ctx context.Context, objectKey string) (io.ReadCloser, int64, error) {
	obj, err := s.client.GetObject(ctx, s.bucketName, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, err
	}
	info, err := obj.Stat()
	if err != nil {
		_ = obj.Close()
		return nil, 0, err
	}
	return obj, info.Size, nil
}

func (s *minioStorage) Bucket() string {
	return s.bucketName
}
