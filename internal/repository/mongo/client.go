package mongo

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Connect(ctx context.Context, uri, dbName string) (*mongo.Database, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("mongo ping: %w", err)
	}

	return client.Database(dbName), nil
}

func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	indexes := map[string][]mongo.IndexModel{
		"stories": {
			{Keys: bson.D{{Key: "title", Value: 1}}},
			{Keys: bson.D{{Key: "status", Value: 1}}},
		},
		"scenes": {
			{Keys: bson.D{{Key: "storyId", Value: 1}}},
			{Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "timelinePosition", Value: 1}}},
			{Keys: bson.D{{Key: "status", Value: 1}}},
		},
		"scene_edges": {
			{Keys: bson.D{{Key: "storyId", Value: 1}}},
			{Keys: bson.D{{Key: "fromSceneId", Value: 1}}},
			{Keys: bson.D{{Key: "toSceneId", Value: 1}}},
			{
				Keys:    bson.D{{Key: "storyId", Value: 1}, {Key: "fromSceneId", Value: 1}, {Key: "toSceneId", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
		},
		"characters": {
			{Keys: bson.D{{Key: "storyId", Value: 1}}},
			{Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "name", Value: 1}}},
			{
				Keys:    bson.D{{Key: "charId", Value: 1}, {Key: "version", Value: -1}},
				Options: options.Index().SetUnique(true),
			},
		},
		"character_state": {
			{Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "characterId", Value: 1}, {Key: "sceneId", Value: 1}, {Key: "createdAt", Value: -1}}},
			{Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "characterId", Value: 1}, {Key: "createdAt", Value: -1}}},
			{Keys: bson.D{{Key: "sceneId", Value: 1}}},
		},
		"character_memories": {
			{Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "characterId", Value: 1}, {Key: "createdAt", Value: -1}}},
			{Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "characterId", Value: 1}, {Key: "importance", Value: -1}}},
		},
		"locations": {
			{Keys: bson.D{{Key: "storyId", Value: 1}}},
			{
				Keys:    bson.D{{Key: "storyId", Value: 1}, {Key: "name", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
		},
		"generations": {
			{Keys: bson.D{{Key: "sceneId", Value: 1}, {Key: "createdAt", Value: -1}}},
			{Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "createdAt", Value: -1}}},
		},
		"summaries": {
			{Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "level", Value: 1}, {Key: "createdAt", Value: -1}}},
			{Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "sceneId", Value: 1}}},
		},
		"timeline_events": {
			{Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "order", Value: 1}}},
			{Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "sceneId", Value: 1}}},
		},
		"scene_turns": {
			{Keys: bson.D{{Key: "sceneId", Value: 1}, {Key: "number", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "sceneId", Value: 1}, {Key: "role", Value: 1}}},
			{Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "createdAt", Value: -1}}},
		},
		"agent_runs": {
			{Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "sceneId", Value: 1}, {Key: "createdAt", Value: -1}}},
			{Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "agentType", Value: 1}}},
			{Keys: bson.D{{Key: "sceneId", Value: 1}}},
		},
		"canon_deltas": {
			{Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "createdAt", Value: -1}}},
			{Keys: bson.D{{Key: "sceneId", Value: 1}}},
			{Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "category", Value: 1}}},
		},
		"jobs": {
			{Keys: bson.D{{Key: "type", Value: 1}, {Key: "status", Value: 1}}},
			{Keys: bson.D{{Key: "status", Value: 1}, {Key: "updatedAt", Value: 1}}},
		},
	}

	for collName, models := range indexes {
		coll := db.Collection(collName)
		for _, m := range models {
			_, err := coll.Indexes().CreateOne(ctx, m)
			if err != nil && strings.Contains(err.Error(), "IndexKeySpecsConflict") {
				slog.Warn("index conflict, skipping", "collection", collName, "error", err)
				continue
			}
			if err != nil {
				return fmt.Errorf("create index for %s: %w", collName, err)
			}
		}
	}

	return nil
}
