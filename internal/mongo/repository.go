package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/premchand/story-builder/internal/story"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repository struct{ db *mongo.Database }

func Connect(ctx context.Context, uri, database string) (*mongo.Client, *Repository, error) {
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(connectCtx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, nil, fmt.Errorf("connect MongoDB: %w", err)
	}
	if err := client.Ping(connectCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, nil, fmt.Errorf("ping MongoDB: %w", err)
	}
	repo := &Repository{db: client.Database(database)}
	if err := repo.EnsureIndexes(ctx); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, nil, err
	}
	return client, repo, nil
}

func (r *Repository) EnsureIndexes(ctx context.Context) error {
	indexes := []struct {
		collection string
		model      mongo.IndexModel
	}{
		{"scenes", mongo.IndexModel{Keys: bson.D{{Key: "storyId", Value: 1}}}},
		{"edges", mongo.IndexModel{Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "fromSceneId", Value: 1}, {Key: "toSceneId", Value: 1}}, Options: options.Index().SetUnique(true)}},
	}
	for _, item := range indexes {
		if _, err := r.db.Collection(item.collection).Indexes().CreateOne(ctx, item.model); err != nil {
			return fmt.Errorf("ensure %s index: %w", item.collection, err)
		}
	}
	return nil
}

func (r *Repository) CreateStory(ctx context.Context, value story.Story) error {
	_, err := r.db.Collection("stories").InsertOne(ctx, value)
	return err
}
func (r *Repository) GetStory(ctx context.Context, id string) (story.Story, error) {
	var value story.Story
	err := r.db.Collection("stories").FindOne(ctx, bson.M{"_id": id}).Decode(&value)
	return value, findError(err)
}
func (r *Repository) ListStories(ctx context.Context) ([]story.Story, error) {
	return list[story.Story](ctx, r.db.Collection("stories"), bson.M{}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
}
func (r *Repository) CreateScene(ctx context.Context, value story.Scene) error {
	_, err := r.db.Collection("scenes").InsertOne(ctx, value)
	return err
}
func (r *Repository) GetScene(ctx context.Context, id string) (story.Scene, error) {
	var value story.Scene
	err := r.db.Collection("scenes").FindOne(ctx, bson.M{"_id": id}).Decode(&value)
	return value, findError(err)
}
func (r *Repository) UpdateScene(ctx context.Context, value story.Scene) error {
	result, err := r.db.Collection("scenes").ReplaceOne(ctx, bson.M{"_id": value.ID}, value)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return story.ErrNotFound
	}
	return nil
}
func (r *Repository) ListScenes(ctx context.Context, storyID string) ([]story.Scene, error) {
	return list[story.Scene](ctx, r.db.Collection("scenes"), bson.M{"storyId": storyID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
}
func (r *Repository) CreateEdge(ctx context.Context, value story.Edge) error {
	_, err := r.db.Collection("edges").InsertOne(ctx, value)
	if mongo.IsDuplicateKeyError(err) {
		return story.ErrDuplicateEdge
	}
	return err
}
func (r *Repository) ListEdges(ctx context.Context, storyID string) ([]story.Edge, error) {
	return list[story.Edge](ctx, r.db.Collection("edges"), bson.M{"storyId": storyID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
}

func findError(err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return story.ErrNotFound
	}
	return err
}
func list[T any](ctx context.Context, collection *mongo.Collection, filter any, options *options.FindOptions) ([]T, error) {
	cursor, err := collection.Find(ctx, filter, options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var values []T
	if err := cursor.All(ctx, &values); err != nil {
		return nil, err
	}
	return values, nil
}
