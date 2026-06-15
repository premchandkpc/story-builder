package mongo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type EventDoc struct {
	ID          string         `bson:"_id" json:"id"`
	Type        string         `bson:"type" json:"type"`
	AggregateID string         `bson:"aggregate_id" json:"aggregate_id"`
	StoryID     string         `bson:"story_id,omitempty" json:"story_id,omitempty"`
	SceneID     string         `bson:"scene_id,omitempty" json:"scene_id,omitempty"`
	CharID      string         `bson:"character_id,omitempty" json:"character_id,omitempty"`
	GenID       string         `bson:"generation_id,omitempty" json:"generation_id,omitempty"`
	Payload     map[string]any `bson:"payload,omitempty" json:"payload,omitempty"`
	TraceID     string         `bson:"trace_id,omitempty" json:"trace_id,omitempty"`
	Timestamp   time.Time      `bson:"timestamp" json:"timestamp"`
}

type EventStoreRepo struct {
	coll *mongo.Collection
}

func NewEventStoreRepo(client *Client) *EventStoreRepo {
	return &EventStoreRepo{coll: client.Coll(CollEventStore)}
}

func (r *EventStoreRepo) Append(ctx context.Context, doc *EventDoc) error {
	doc.ID = uuid.New().String()
	doc.Timestamp = time.Now()
	_, err := r.coll.InsertOne(ctx, doc)
	return err
}

func (r *EventStoreRepo) GetByAggregate(ctx context.Context, aggregateID string) ([]EventDoc, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"aggregate_id": aggregateID})
	if err != nil {
		return nil, err
	}
	var docs []EventDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

func (r *EventStoreRepo) GetByStory(ctx context.Context, storyID string, limit int64) ([]EventDoc, error) {
	opts := options.Find().SetSort(bson.M{"timestamp": -1})
	if limit > 0 {
		opts.SetLimit(limit)
	}
	cursor, err := r.coll.Find(ctx, bson.M{"story_id": storyID}, opts)
	if err != nil {
		return nil, err
	}
	var docs []EventDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

func (r *EventStoreRepo) Replay(ctx context.Context, aggregateID string) ([]EventDoc, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"aggregate_id": aggregateID})
	if err != nil {
		return nil, err
	}
	var docs []EventDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}
