package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/premchand/story-builder/internal/domain"
)

type SceneRepo struct {
	coll *mongo.Collection
}

func NewSceneRepo(db *mongo.Database) *SceneRepo {
	return &SceneRepo{coll: db.Collection("scenes")}
}

func (r *SceneRepo) Create(ctx context.Context, s *domain.Scene) error {
	s.CreatedAt = time.Now()
	s.UpdatedAt = s.CreatedAt
	if s.ID == "" {
		s.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.coll.InsertOne(ctx, s)
	return err
}

func (r *SceneRepo) Get(ctx context.Context, id string) (*domain.Scene, error) {
	var s domain.Scene
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&s)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &s, err
}

func (r *SceneRepo) Update(ctx context.Context, s *domain.Scene) error {
	s.UpdatedAt = time.Now()
	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": s.ID}, bson.M{"$set": s})
	return err
}

func (r *SceneRepo) ListByStory(ctx context.Context, storyID string) ([]*domain.Scene, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"storyId": storyID})
	if err != nil {
		return nil, err
	}
	var scenes []*domain.Scene
	if err := cursor.All(ctx, &scenes); err != nil {
		return nil, err
	}
	return scenes, nil
}

func (r *SceneRepo) Delete(ctx context.Context, id string) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *SceneRepo) DeleteByStory(ctx context.Context, storyID string) error {
	_, err := r.coll.DeleteMany(ctx, bson.M{"storyId": storyID})
	return err
}
