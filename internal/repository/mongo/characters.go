package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/premchand/story-builder/internal/domain"
)

type CharacterRepo struct {
	coll *mongo.Collection
}

func NewCharacterRepo(db *mongo.Database) *CharacterRepo {
	return &CharacterRepo{coll: db.Collection("characters")}
}

func (r *CharacterRepo) Create(ctx context.Context, c *domain.Character) error {
	c.CreatedAt = time.Now()
	if c.ID == "" {
		c.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.coll.InsertOne(ctx, c)
	return err
}

func (r *CharacterRepo) Get(ctx context.Context, id string) (*domain.Character, error) {
	var c domain.Character
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&c)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &c, err
}

func (r *CharacterRepo) ListByStory(ctx context.Context, storyID string) ([]*domain.Character, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"storyId": storyID})
	if err != nil {
		return nil, err
	}
	var chars []*domain.Character
	if err := cursor.All(ctx, &chars); err != nil {
		return nil, err
	}
	return chars, nil
}

func (r *CharacterRepo) Update(ctx context.Context, c *domain.Character) error {
	_, err := r.coll.ReplaceOne(ctx, bson.M{"_id": c.ID}, c)
	return err
}
