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

type SceneTurnRepo struct {
	coll *mongo.Collection
}

func NewSceneTurnRepo(db *mongo.Database) *SceneTurnRepo {
	return &SceneTurnRepo{coll: db.Collection("scene_turns")}
}

func (r *SceneTurnRepo) Create(ctx context.Context, t *domain.SceneTurn) error {
	t.CreatedAt = time.Now()
	if t.ID == "" {
		t.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.coll.InsertOne(ctx, t)
	return err
}

func (r *SceneTurnRepo) Get(ctx context.Context, id string) (*domain.SceneTurn, error) {
	var t domain.SceneTurn
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&t)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *SceneTurnRepo) Update(ctx context.Context, t *domain.SceneTurn) error {
	t.UpdatedAt = time.Now()
	_, err := r.coll.ReplaceOne(ctx, bson.M{"_id": t.ID}, t)
	return err
}

func (r *SceneTurnRepo) ListByScene(ctx context.Context, sceneID string) ([]*domain.SceneTurn, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"sceneId": sceneID}, options.Find().SetSort(bson.D{{Key: "number", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var turns []*domain.SceneTurn
	if err := cursor.All(ctx, &turns); err != nil {
		return nil, err
	}
	return turns, nil
}

func (r *SceneTurnRepo) ListByRole(ctx context.Context, sceneID, role string) ([]*domain.SceneTurn, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"sceneId": sceneID, "role": role}, options.Find().SetSort(bson.D{{Key: "number", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var turns []*domain.SceneTurn
	if err := cursor.All(ctx, &turns); err != nil {
		return nil, err
	}
	return turns, nil
}

func (r *SceneTurnRepo) DeleteByScene(ctx context.Context, sceneID string) error {
	_, err := r.coll.DeleteMany(ctx, bson.M{"sceneId": sceneID})
	return err
}

func (r *SceneTurnRepo) DeleteByStory(ctx context.Context, storyID string) error {
	_, err := r.coll.DeleteMany(ctx, bson.M{"storyId": storyID})
	return err
}
