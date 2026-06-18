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

type CharacterRepo struct {
	coll *mongo.Collection
}

func NewCharacterRepo(db *mongo.Database) *CharacterRepo {
	return &CharacterRepo{coll: db.Collection("characters")}
}

func (r *CharacterRepo) Create(ctx context.Context, c *domain.Character) error {
	c.CreatedAt = time.Now()
	c.Version = 1
	if c.ID == "" {
		c.ID = primitive.NewObjectID().Hex()
	}
	if c.CharID == "" {
		c.CharID = c.ID
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

func (r *CharacterRepo) GetLatest(ctx context.Context, charID string) (*domain.Character, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "version", Value: -1}})
	var c domain.Character
	err := r.coll.FindOne(ctx, bson.M{"charId": charID}, opts).Decode(&c)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &c, err
}

func (r *CharacterRepo) ListByStory(ctx context.Context, storyID string) ([]*domain.Character, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"storyId": storyID}}},
		{{Key: "$sort", Value: bson.D{{Key: "version", Value: -1}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$charId"},
			{Key: "doc", Value: bson.D{{Key: "$first", Value: "$$ROOT"}}},
		}}},
		{{Key: "$replaceWith", Value: "$doc"}},
	}
	cursor, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var chars []*domain.Character
	if err := cursor.All(ctx, &chars); err != nil {
		return nil, err
	}
	return chars, nil
}

// Update creates a new versioned document (immutable append-log).
func (r *CharacterRepo) Update(ctx context.Context, c *domain.Character) error {
	doc := *c
	doc.ID = primitive.NewObjectID().Hex()
	doc.Version = c.Version + 1
	doc.CreatedAt = time.Now()
	_, err := r.coll.InsertOne(ctx, &doc)
	return err
}

func (r *CharacterRepo) DeleteByStory(ctx context.Context, storyID string) error {
	_, err := r.coll.DeleteMany(ctx, bson.M{"storyId": storyID})
	return err
}
