package mongo

import (
	"context"

	"github.com/premchand/story-builder/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type BibleRepo struct {
	coll *mongo.Collection
}

func NewBibleRepo(db *mongo.Database) *BibleRepo {
	return &BibleRepo{coll: db.Collection("bibles")}
}

func (r *BibleRepo) Create(ctx context.Context, b *domain.StoryBible) error {
	_, err := r.coll.InsertOne(ctx, b)
	return err
}

func (r *BibleRepo) Get(ctx context.Context, id string) (*domain.StoryBible, error) {
	var b domain.StoryBible
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&b)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *BibleRepo) GetByStory(ctx context.Context, storyID string) (*domain.StoryBible, error) {
	var b domain.StoryBible
	err := r.coll.FindOne(ctx, bson.M{"storyId": storyID}).Decode(&b)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *BibleRepo) Update(ctx context.Context, b *domain.StoryBible) error {
	_, err := r.coll.ReplaceOne(ctx, bson.M{"_id": b.ID}, b)
	return err
}

func (r *BibleRepo) DeleteByStory(ctx context.Context, storyID string) error {
	_, err := r.coll.DeleteMany(ctx, bson.M{"storyId": storyID})
	return err
}
