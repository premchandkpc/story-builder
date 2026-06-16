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

type SummaryRepo struct {
	coll *mongo.Collection
}

func NewSummaryRepo(db *mongo.Database) *SummaryRepo {
	return &SummaryRepo{coll: db.Collection("summaries")}
}

func (r *SummaryRepo) Upsert(ctx context.Context, s *domain.Summary) error {
	s.CreatedAt = time.Now()
	if s.ID == "" {
		s.ID = primitive.NewObjectID().Hex()
	}
	filter := bson.M{"storyId": s.StoryID}
	if s.SceneID != "" {
		filter["sceneId"] = s.SceneID
	}
	filter["level"] = s.Level

	_, err := r.coll.ReplaceOne(ctx, filter, s, options.Replace().SetUpsert(true))
	return err
}

func (r *SummaryRepo) GetByLevel(ctx context.Context, storyID, level string) (*domain.Summary, error) {
	var s domain.Summary
	err := r.coll.FindOne(ctx, bson.M{"storyId": storyID, "level": level}).Decode(&s)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &s, err
}

func (r *SummaryRepo) GetSceneSummary(ctx context.Context, storyID, sceneID string) (*domain.Summary, error) {
	var s domain.Summary
	err := r.coll.FindOne(ctx, bson.M{"storyId": storyID, "sceneId": sceneID, "level": "scene"}).Decode(&s)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &s, err
}

func (r *SummaryRepo) ListByLevel(ctx context.Context, storyID, level string) ([]*domain.Summary, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"storyId": storyID, "level": level})
	if err != nil {
		return nil, err
	}
	var summaries []*domain.Summary
	if err := cursor.All(ctx, &summaries); err != nil {
		return nil, err
	}
	return summaries, nil
}
