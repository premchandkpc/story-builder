package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/premchand/story-builder/internal/domain"
)

type LocationRepo struct {
	coll *mongo.Collection
}

func NewLocationRepo(db *mongo.Database) *LocationRepo {
	return &LocationRepo{coll: db.Collection("locations")}
}

func (r *LocationRepo) Create(ctx context.Context, l *domain.Location) error {
	l.CreatedAt = time.Now()
	if l.ID == "" {
		l.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.coll.InsertOne(ctx, l)
	return err
}

func (r *LocationRepo) GetByName(ctx context.Context, storyID, name string) (*domain.Location, error) {
	var l domain.Location
	err := r.coll.FindOne(ctx, bson.M{"storyId": storyID, "name": name}).Decode(&l)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &l, err
}

func (r *LocationRepo) Get(ctx context.Context, id string) (*domain.Location, error) {
	var l domain.Location
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&l)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &l, err
}

func (r *LocationRepo) ListByStory(ctx context.Context, storyID string) ([]*domain.Location, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"storyId": storyID})
	if err != nil {
		return nil, err
	}
	var locs []*domain.Location
	if err := cursor.All(ctx, &locs); err != nil {
		return nil, err
	}
	return locs, nil
}

func (r *LocationRepo) Update(ctx context.Context, l *domain.Location) error {
	_, err := r.coll.ReplaceOne(ctx, bson.M{"_id": l.ID}, l)
	return err
}

func (r *LocationRepo) DeleteByStory(ctx context.Context, storyID string) error {
	_, err := r.coll.DeleteMany(ctx, bson.M{"storyId": storyID})
	return err
}
