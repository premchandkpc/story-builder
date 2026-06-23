package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/premchand/story-builder/internal/domain"
)

type CharacterViewRepo struct {
	coll *mongo.Collection
}

func NewCharacterViewRepo(db *mongo.Database) *CharacterViewRepo {
	return &CharacterViewRepo{coll: db.Collection("character_views")}
}

func (r *CharacterViewRepo) Get(ctx context.Context, charID string) (*domain.CharacterView, error) {
	var view domain.CharacterView
	if err := r.coll.FindOne(ctx, bson.M{"_id": charID}).Decode(&view); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &view, nil
}

func (r *CharacterViewRepo) Upsert(ctx context.Context, view *domain.CharacterView) error {
	view.UpdatedAt = time.Now()
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"_id": view.CharacterID},
		bson.M{"$set": view},
		options.Update().SetUpsert(true),
	)
	return err
}

func (r *CharacterViewRepo) ListByStory(ctx context.Context, storyID string) ([]*domain.CharacterView, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"storyId": storyID})
	if err != nil {
		return nil, err
	}
	var views []*domain.CharacterView
	if err := cursor.All(ctx, &views); err != nil {
		return nil, err
	}
	return views, nil
}

func (r *CharacterViewRepo) DeleteByStory(ctx context.Context, storyID string) error {
	_, err := r.coll.DeleteMany(ctx, bson.M{"storyId": storyID})
	return err
}
