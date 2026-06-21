package mongo

import (
	"context"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TokenBudgetRepo struct {
	coll *mongo.Collection
}

func NewTokenBudgetRepo(db *mongo.Database) *TokenBudgetRepo {
	return &TokenBudgetRepo{coll: db.Collection("token_budgets")}
}

func (r *TokenBudgetRepo) Get(ctx context.Context, storyID string) (*domain.TokenBudget, error) {
	var tb domain.TokenBudget
	if err := r.coll.FindOne(ctx, bson.M{"storyId": storyID}).Decode(&tb); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &tb, nil
}

func (r *TokenBudgetRepo) Upsert(ctx context.Context, tb *domain.TokenBudget) error {
	tb.UpdatedAt = time.Now()
	opts := options.Replace().SetUpsert(true)
	_, err := r.coll.ReplaceOne(ctx, bson.M{"storyId": tb.StoryID}, tb, opts)
	return err
}

func (r *TokenBudgetRepo) DeleteByStory(ctx context.Context, storyID string) error {
	_, err := r.coll.DeleteMany(ctx, bson.M{"storyId": storyID})
	return err
}
