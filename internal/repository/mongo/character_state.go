package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/premchand/story-builder/internal/domain"
)

type CharacterStateRepo struct {
	coll *mongo.Collection
}

func NewCharacterStateRepo(db *mongo.Database) *CharacterStateRepo {
	return &CharacterStateRepo{coll: db.Collection("character_state")}
}

func (r *CharacterStateRepo) Append(ctx context.Context, s *domain.CharacterState) error {
	s.CreatedAt = time.Now()
	_, err := r.coll.InsertOne(ctx, s)
	return err
}

func (r *CharacterStateRepo) Get(ctx context.Context, characterID, sceneID string) (*domain.CharacterState, error) {
	var s domain.CharacterState
	err := r.coll.FindOne(ctx, bson.M{"characterId": characterID, "sceneId": sceneID}).Decode(&s)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &s, err
}

func (r *CharacterStateRepo) ListByCharacter(ctx context.Context, characterID string) ([]*domain.CharacterState, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"characterId": characterID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var states []*domain.CharacterState
	if err := cursor.All(ctx, &states); err != nil {
		return nil, err
	}
	return states, nil
}

func (r *CharacterStateRepo) ListByScene(ctx context.Context, sceneID string) ([]*domain.CharacterState, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"sceneId": sceneID})
	if err != nil {
		return nil, err
	}
	var states []*domain.CharacterState
	if err := cursor.All(ctx, &states); err != nil {
		return nil, err
	}
	return states, nil
}
