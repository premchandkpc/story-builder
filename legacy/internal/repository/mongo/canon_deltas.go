package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/premchand/story-builder/internal/domain"
)

type CanonDeltaRepo struct {
	coll *mongo.Collection
}

func NewCanonDeltaRepo(db *mongo.Database) *CanonDeltaRepo {
	return &CanonDeltaRepo{coll: db.Collection("canon_deltas")}
}

func (r *CanonDeltaRepo) Create(ctx context.Context, d *domain.CanonDelta) error {
	d.CreatedAt = time.Now()
	if d.ID == "" {
		d.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.coll.InsertOne(ctx, d)
	return err
}

func (r *CanonDeltaRepo) ListByScene(ctx context.Context, sceneID string) ([]*domain.CanonDelta, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"sceneId": sceneID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	var deltas []*domain.CanonDelta
	if err := cursor.All(ctx, &deltas); err != nil {
		return nil, err
	}
	return deltas, nil
}

func (r *CanonDeltaRepo) ListByStory(ctx context.Context, storyID string) ([]*domain.CanonDelta, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"storyId": storyID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	var deltas []*domain.CanonDelta
	if err := cursor.All(ctx, &deltas); err != nil {
		return nil, err
	}
	return deltas, nil
}

func (r *CanonDeltaRepo) DeleteByStory(ctx context.Context, storyID string) error {
	_, err := r.coll.DeleteMany(ctx, bson.M{"storyId": storyID})
	return err
}
