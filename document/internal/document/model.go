package document

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Metadata представляет документ, хранящийся в MongoDB.
type Metadata struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TaskID       string             `bson:"task_id" json:"task_id"`
	OwnerID      string             `bson:"owner_id" json:"owner_id"`
	Filename     string             `bson:"filename" json:"filename"`
	ContentType  string             `bson:"content_type" json:"content_type"`
	Size         int64              `bson:"size" json:"size"`
	Tags         []string           `bson:"tags" json:"tags"`
	UploadedAt   time.Time          `bson:"uploaded_at" json:"uploaded_at"`
	MinioObject  string             `bson:"minio_object" json:"minio_object"`
	MinioBucket  string             `bson:"minio_bucket" json:"minio_bucket"`
	Checksum     string             `bson:"checksum,omitempty" json:"checksum,omitempty"`
	LastModified time.Time          `bson:"last_modified" json:"last_modified"`
}
