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

type AgentRunRepo struct {
	coll *mongo.Collection
}

func NewAgentRunRepo(db *mongo.Database) *AgentRunRepo {
	return &AgentRunRepo{coll: db.Collection("agent_runs")}
}

func (r *AgentRunRepo) Create(ctx context.Context, a *domain.AgentRun) error {
	a.CreatedAt = time.Now()
	if a.ID == "" {
		a.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.coll.InsertOne(ctx, a)
	return err
}

func (r *AgentRunRepo) List(ctx context.Context, filter domain.AgentRunFilter) ([]*domain.AgentRun, error) {
	q := bson.M{}
	if filter.StoryID != "" {
		q["storyId"] = filter.StoryID
	}
	if filter.SceneID != "" {
		q["sceneId"] = filter.SceneID
	}
	if filter.AgentType != "" {
		q["agentType"] = filter.AgentType
	}
	if filter.Status != "" {
		q["status"] = filter.Status
	}
	opt := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	if filter.Limit > 0 {
		opt.SetLimit(int64(filter.Limit))
	}
	cursor, err := r.coll.Find(ctx, q, opt)
	if err != nil {
		return nil, err
	}
	var runs []*domain.AgentRun
	if err := cursor.All(ctx, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}

func (r *AgentRunRepo) DeleteByStory(ctx context.Context, storyID string) error {
	_, err := r.coll.DeleteMany(ctx, bson.M{"storyId": storyID})
	return err
}
