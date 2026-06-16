package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/premchand/story-builder/internal/domain"
)

type SceneEdgeRepo struct {
	coll *mongo.Collection
}

func NewSceneEdgeRepo(db *mongo.Database) *SceneEdgeRepo {
	return &SceneEdgeRepo{coll: db.Collection("scene_edges")}
}

func (r *SceneEdgeRepo) Create(ctx context.Context, e *domain.SceneEdge) error {
	e.CreatedAt = time.Now()
	if e.ID == "" {
		e.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.coll.InsertOne(ctx, e)
	return err
}

func (r *SceneEdgeRepo) ListByStory(ctx context.Context, storyID string) ([]*domain.SceneEdge, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"storyId": storyID})
	if err != nil {
		return nil, err
	}
	var edges []*domain.SceneEdge
	if err := cursor.All(ctx, &edges); err != nil {
		return nil, err
	}
	return edges, nil
}

func (r *SceneEdgeRepo) ListFrom(ctx context.Context, sceneID string) ([]*domain.SceneEdge, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"fromSceneId": sceneID})
	if err != nil {
		return nil, err
	}
	var edges []*domain.SceneEdge
	if err := cursor.All(ctx, &edges); err != nil {
		return nil, err
	}
	return edges, nil
}

func (r *SceneEdgeRepo) ListTo(ctx context.Context, sceneID string) ([]*domain.SceneEdge, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"toSceneId": sceneID})
	if err != nil {
		return nil, err
	}
	var edges []*domain.SceneEdge
	if err := cursor.All(ctx, &edges); err != nil {
		return nil, err
	}
	return edges, nil
}

func (r *SceneEdgeRepo) Delete(ctx context.Context, storyID, fromSceneID, toSceneID string) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{
		"storyId":     storyID,
		"fromSceneId": fromSceneID,
		"toSceneId":   toSceneID,
	})
	return err
}
