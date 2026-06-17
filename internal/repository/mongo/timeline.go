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

type TimelineRepo struct {
	coll *mongo.Collection
}

func NewTimelineRepo(db *mongo.Database) *TimelineRepo {
	return &TimelineRepo{coll: db.Collection("timeline_events")}
}

func (r *TimelineRepo) Create(ctx context.Context, e *domain.TimelineEvent) error {
	e.CreatedAt = time.Now()
	if e.ID == "" {
		e.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.coll.InsertOne(ctx, e)
	return err
}

func (r *TimelineRepo) DeleteByStory(ctx context.Context, storyID string) error {
	_, err := r.coll.DeleteMany(ctx, bson.M{"storyId": storyID})
	return err
}

func (r *TimelineRepo) ListByStory(ctx context.Context, storyID string) ([]*domain.TimelineEvent, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"storyId": storyID}, options.Find().SetSort(bson.D{{Key: "order", Value: 1}, {Key: "createdAt", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var events []*domain.TimelineEvent
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}
