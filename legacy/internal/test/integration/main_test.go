//go:build integration

package integration

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	mgorepo "github.com/premchand/story-builder/internal/repository/mongo"
)

var testDB *mongo.Database

func TestMain(m *testing.M) {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://storybuilder:storybuilder@localhost:27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("mongo connect: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("mongo ping: %v (is docker compose up?)", err)
	}

	dbName := "storybuilder_test"
	testDB = client.Database(dbName)

	if err := mgorepo.EnsureIndexes(ctx, testDB); err != nil {
		log.Fatalf("ensure indexes: %v", err)
	}

	code := m.Run()

	testDB.Drop(context.Background())
	client.Disconnect(context.Background())
	os.Exit(code)
}

func cleanCollections(t *testing.T, colls ...string) {
	t.Helper()
	ctx := context.Background()
	for _, c := range colls {
		if _, err := testDB.Collection(c).DeleteMany(ctx, bson.M{}); err != nil {
			t.Fatalf("clean %s: %v", c, err)
		}
	}
}
