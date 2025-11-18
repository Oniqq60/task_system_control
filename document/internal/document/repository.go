package document

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var ErrNotFound = errors.New("document not found")

type Repository interface {
	Insert(ctx context.Context, metadata Metadata) (primitive.ObjectID, error)
	Delete(ctx context.Context, id primitive.ObjectID) error
	FindByID(ctx context.Context, id primitive.ObjectID) (Metadata, error)
	FindByTask(ctx context.Context, taskID string) ([]Metadata, error)
	FindByOwner(ctx context.Context, ownerID string) ([]Metadata, error)
}

type mongoRepository struct {
	collection *mongo.Collection
}

func NewRepository(collection *mongo.Collection) Repository {
	return &mongoRepository{
		collection: collection,
	}
}

func (r *mongoRepository) Insert(ctx context.Context, metadata Metadata) (primitive.ObjectID, error) {
	now := time.Now()
	metadata.UploadedAt = now
	metadata.LastModified = now

	res, err := r.collection.InsertOne(ctx, metadata)
	if err != nil {
		return primitive.NilObjectID, err
	}

	id, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, errors.New("unexpected insert id type")
	}
	return id, nil
}

func (r *mongoRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	res, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *mongoRepository) FindByID(ctx context.Context, id primitive.ObjectID) (Metadata, error) {
	var metadata Metadata
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&metadata)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Metadata{}, ErrNotFound
		}
		return Metadata{}, err
	}
	return metadata, nil
}

func (r *mongoRepository) FindByTask(ctx context.Context, taskID string) ([]Metadata, error) {
	return r.findMany(ctx, bson.M{"task_id": taskID})
}

func (r *mongoRepository) FindByOwner(ctx context.Context, ownerID string) ([]Metadata, error) {
	return r.findMany(ctx, bson.M{"owner_id": ownerID})
}

func (r *mongoRepository) findMany(ctx context.Context, filter bson.M) ([]Metadata, error) {
	cur, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var docs []Metadata
	for cur.Next(ctx) {
		var doc Metadata
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}

	return docs, nil
}
