package mongo

import (
	"context"

	"github.com/premchand/story-builder/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ChapterRepo struct {
	coll *mongo.Collection
}

func NewChapterRepo(db *mongo.Database) *ChapterRepo {
	return &ChapterRepo{coll: db.Collection("chapters")}
}

func (r *ChapterRepo) Create(ctx context.Context, c *domain.Chapter) error {
	_, err := r.coll.InsertOne(ctx, c)
	return err
}

func (r *ChapterRepo) Get(ctx context.Context, id string) (*domain.Chapter, error) {
	var c domain.Chapter
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&c)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ChapterRepo) ListByStory(ctx context.Context, storyID string) ([]*domain.Chapter, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"storyId": storyID})
	if err != nil {
		return nil, err
	}
	var chapters []*domain.Chapter
	if err := cursor.All(ctx, &chapters); err != nil {
		return nil, err
	}
	return chapters, nil
}

func (r *ChapterRepo) ListByAct(ctx context.Context, storyID string, actNumber int) ([]*domain.Chapter, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"storyId": storyID, "actNumber": actNumber})
	if err != nil {
		return nil, err
	}
	var chapters []*domain.Chapter
	if err := cursor.All(ctx, &chapters); err != nil {
		return nil, err
	}
	return chapters, nil
}

func (r *ChapterRepo) Update(ctx context.Context, c *domain.Chapter) error {
	_, err := r.coll.ReplaceOne(ctx, bson.M{"_id": c.ID}, c)
	return err
}

func (r *ChapterRepo) Delete(ctx context.Context, id string) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *ChapterRepo) DeleteByStory(ctx context.Context, storyID string) error {
	_, err := r.coll.DeleteMany(ctx, bson.M{"storyId": storyID})
	return err
}
