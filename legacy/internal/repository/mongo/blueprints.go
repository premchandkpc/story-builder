package mongo

import (
	"context"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BlueprintRepo struct {
	coll *mongo.Collection
}

func NewBlueprintRepo(db *mongo.Database) *BlueprintRepo {
	return &BlueprintRepo{coll: db.Collection("story_blueprints")}
}

func (r *BlueprintRepo) GetByStory(ctx context.Context, storyID string) (*domain.StoryBlueprint, error) {
	var bp domain.StoryBlueprint
	err := r.coll.FindOne(ctx, bson.M{"_id": storyID}).Decode(&bp)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &bp, nil
}

func (r *BlueprintRepo) Upsert(ctx context.Context, bp *domain.StoryBlueprint) error {
	now := time.Now()
	filter := bson.M{"_id": bp.Premise}
	update := bson.M{"$set": bson.M{
		"theme":        bp.Theme,
		"genre":        bp.Genre,
		"mainConflict": bp.MainConflict,
		"acts":         bp.Acts,
		"characterArcs": bp.CharacterArcs,
		"plotThreads":  bp.PlotThreads,
		"endingState":  bp.EndingState,
		"updatedAt":    now,
	}}
	_, err := r.coll.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

func (r *BlueprintRepo) DeleteByStory(ctx context.Context, storyID string) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": storyID})
	return err
}
