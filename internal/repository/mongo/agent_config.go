package mongo

import (
	"context"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AgentConfigRepo struct {
	coll *mongo.Collection
}

func NewAgentConfigRepo(db *mongo.Database) *AgentConfigRepo {
	return &AgentConfigRepo{coll: db.Collection("agent_configs")}
}

func (r *AgentConfigRepo) Create(ctx context.Context, a *domain.AgentConfig) error {
	a.CreatedAt = time.Now()
	_, err := r.coll.InsertOne(ctx, a)
	return err
}

func (r *AgentConfigRepo) Get(ctx context.Context, name string) (*domain.AgentConfig, error) {
	var a domain.AgentConfig
	err := r.coll.FindOne(ctx, bson.M{"_id": name}).Decode(&a)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AgentConfigRepo) List(ctx context.Context) ([]*domain.AgentConfig, error) {
	cursor, err := r.coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	var configs []*domain.AgentConfig
	if err := cursor.All(ctx, &configs); err != nil {
		return nil, err
	}
	return configs, nil
}

func (r *AgentConfigRepo) Delete(ctx context.Context, name string) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": name})
	return err
}
