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

type GenerationRepo struct {
	coll *mongo.Collection
}

func NewGenerationRepo(db *mongo.Database) *GenerationRepo {
	return &GenerationRepo{coll: db.Collection("generations")}
}

func (r *GenerationRepo) SetStepStatus(ctx context.Context, genID, step, status string) error {
	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": genID}, bson.M{
		"$set": bson.M{"stepStatus." + step: status},
	})
	return err
}

func (r *GenerationRepo) Create(ctx context.Context, g *domain.Generation) error {
	g.CreatedAt = time.Now()
	if g.ID == "" {
		g.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.coll.InsertOne(ctx, g)
	return err
}

func (r *GenerationRepo) Get(ctx context.Context, id string) (*domain.Generation, error) {
	var g domain.Generation
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&g)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &g, err
}

func (r *GenerationRepo) Update(ctx context.Context, g *domain.Generation) error {
	_, err := r.coll.ReplaceOne(ctx, bson.M{"_id": g.ID}, g)
	return err
}

func (r *GenerationRepo) FindByContextHash(ctx context.Context, storyID, hash string) (*domain.Generation, error) {
	var g domain.Generation
	err := r.coll.FindOne(ctx, bson.M{"storyId": storyID, "contextHash": hash, "accepted": true}).Decode(&g)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &g, err
}

func (r *GenerationRepo) ListByScene(ctx context.Context, sceneID string) ([]*domain.Generation, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"sceneId": sceneID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	var gens []*domain.Generation
	if err := cursor.All(ctx, &gens); err != nil {
		return nil, err
	}
	return gens, nil
}

func (r *GenerationRepo) ListByStory(ctx context.Context, storyID string) ([]*domain.Generation, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"storyId": storyID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	var gens []*domain.Generation
	if err := cursor.All(ctx, &gens); err != nil {
		return nil, err
	}
	return gens, nil
}

func (r *GenerationRepo) DeleteByScene(ctx context.Context, sceneID string) error {
	_, err := r.coll.DeleteMany(ctx, bson.M{"sceneId": sceneID})
	return err
}

func (r *GenerationRepo) SetAccepted(ctx context.Context, sceneID, genID string) error {
	_, err := r.coll.UpdateMany(ctx,
		bson.M{"sceneId": sceneID},
		bson.M{"$set": bson.M{"accepted": false}},
	)
	if err != nil {
		return err
	}
	_, err = r.coll.UpdateOne(ctx,
		bson.M{"_id": genID, "sceneId": sceneID},
		bson.M{"$set": bson.M{"accepted": true}},
	)
	return err
}

func (r *GenerationRepo) DeleteByStory(ctx context.Context, storyID string) error {
	_, err := r.coll.DeleteMany(ctx, bson.M{"storyId": storyID})
	return err
}
