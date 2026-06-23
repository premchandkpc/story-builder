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

type NarrativeEventRepo struct {
	coll *mongo.Collection
}

func NewNarrativeEventRepo(db *mongo.Database) *NarrativeEventRepo {
	return &NarrativeEventRepo{coll: db.Collection("narrative_events")}
}

func (r *NarrativeEventRepo) Append(ctx context.Context, e *domain.NarrativeEvent) error {
	e.CreatedAt = time.Now()
	if e.ID == "" {
		e.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.coll.InsertOne(ctx, e)
	return err
}

func (r *NarrativeEventRepo) ListByStory(ctx context.Context, storyID string, limit int) ([]*domain.NarrativeEvent, error) {
	opt := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	if limit > 0 {
		opt.SetLimit(int64(limit))
	}
	cursor, err := r.coll.Find(ctx, bson.M{"storyId": storyID}, opt)
	if err != nil {
		return nil, err
	}
	var events []*domain.NarrativeEvent
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (r *NarrativeEventRepo) ListByScene(ctx context.Context, sceneID string, limit int) ([]*domain.NarrativeEvent, error) {
	opt := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	if limit > 0 {
		opt.SetLimit(int64(limit))
	}
	cursor, err := r.coll.Find(ctx, bson.M{"sceneId": sceneID}, opt)
	if err != nil {
		return nil, err
	}
	var events []*domain.NarrativeEvent
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (r *NarrativeEventRepo) AppendMany(ctx context.Context, events []*domain.NarrativeEvent) error {
	if len(events) == 0 {
		return nil
	}
	docs := make([]any, len(events))
	now := time.Now()
	for i, e := range events {
		e.CreatedAt = now
		if e.ID == "" {
			e.ID = primitive.NewObjectID().Hex()
		}
		docs[i] = e
	}
	_, err := r.coll.InsertMany(ctx, docs)
	return err
}

func (r *NarrativeEventRepo) ListBySubject(ctx context.Context, storyID, subjectID string, limit int) ([]*domain.NarrativeEvent, error) {
	opt := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	if limit > 0 {
		opt.SetLimit(int64(limit))
	}
	cursor, err := r.coll.Find(ctx, bson.M{
		"storyId":   storyID,
		"subjectId": subjectID,
	}, opt)
	if err != nil {
		return nil, err
	}
	var events []*domain.NarrativeEvent
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (r *NarrativeEventRepo) LatestVersion(ctx context.Context, storyID string) (int64, error) {
	opt := options.FindOne().SetSort(bson.D{{Key: "version", Value: -1}})
	var result struct {
		Version int64 `bson:"version"`
	}
	if err := r.coll.FindOne(ctx, bson.M{"storyId": storyID}, opt).Decode(&result); err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, nil
		}
		return 0, err
	}
	return result.Version, nil
}

func (r *NarrativeEventRepo) DeleteByStory(ctx context.Context, storyID string) error {
	_, err := r.coll.DeleteMany(ctx, bson.M{"storyId": storyID})
	return err
}
