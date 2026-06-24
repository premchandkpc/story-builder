package test

import (
	"context"
	"os"
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func SetupTestDB(t *testing.T) *mongo.Database {
	t.Helper()
	ctx := context.Background()
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("mongo connect: %v", err)
	}
	t.Cleanup(func() { client.Disconnect(ctx) })
	db := client.Database("test_" + randomString(8))
	t.Cleanup(func() { db.Drop(ctx) })
	return db
}
