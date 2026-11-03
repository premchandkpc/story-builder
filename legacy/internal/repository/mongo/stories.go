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

type StoryRepo struct {
	coll *mongo.Collection
}

func NewStoryRepo(db *mongo.Database) *StoryRepo {
	return &StoryRepo{coll: db.Collection("stories")}
}

func (r *StoryRepo) Create(ctx context.Context, s *domain.Story) error {
	s.CreatedAt = time.Now()
	s.UpdatedAt = s.CreatedAt
	if s.ID == "" {
		s.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.coll.InsertOne(ctx, s)
	return err
}

func (r *StoryRepo) Get(ctx context.Context, id string) (*domain.Story, error) {
	var s domain.Story
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&s)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *StoryRepo) Update(ctx context.Context, s *domain.Story) error {
	s.UpdatedAt = time.Now()
	_, err := r.coll.ReplaceOne(ctx, bson.M{"_id": s.ID}, s)
	return err
}

func (r *StoryRepo) List(ctx context.Context) ([]*domain.Story, error) {
	cursor, err := r.coll.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	var stories []*domain.Story
	if err := cursor.All(ctx, &stories); err != nil {
		return nil, err
	}
	return stories, nil
}

func (r *StoryRepo) Delete(ctx context.Context, id string) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
