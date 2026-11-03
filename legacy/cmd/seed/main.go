package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	mgorepo "github.com/premchand/story-builder/internal/repository/mongo"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SeedConfig struct {
	MongoURI    string
	MongoDB     string
	StoryCount  int
	SceneCount  int
	CharCount   int
}

func main() {
	cfg := SeedConfig{
		MongoURI:   getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		MongoDB:    getEnv("MONGODB_DB", "story_builder"),
		StoryCount: 2,
		SceneCount: 5,
		CharCount:  3,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		slog.Error("mongo connect failed", "error", err)
		os.Exit(1)
	}
	defer client.Disconnect(ctx)

	db := client.Database(cfg.MongoDB)

	storyRepo := mgorepo.NewStoryRepo(db)
	sceneRepo := mgorepo.NewSceneRepo(db)
	charRepo := mgorepo.NewCharacterRepo(db)
	bibleRepo := mgorepo.NewBibleRepo(db)
	edgeRepo := mgorepo.NewSceneEdgeRepo(db)
	locRepo := mgorepo.NewLocationRepo(db)
	tlRepo := mgorepo.NewTimelineRepo(db)

	for i := 0; i < cfg.StoryCount; i++ {
		story := &domain.Story{
			ID:        primitive.NewObjectID().Hex(),
			Title:     fmt.Sprintf("Demo Story %d", i+1),
			Status:    domain.StoryStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := storyRepo.Create(ctx, story); err != nil {
			slog.Error("create story failed", "error", err)
			continue
		}

		bible := &domain.StoryBible{
			ID:     primitive.NewObjectID().Hex(),
			StoryID: story.ID,
			World:  fmt.Sprintf("A demo world for story %d", i+1),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := bibleRepo.Create(ctx, bible); err != nil {
			slog.Warn("create bible failed", "error", err)
		}

		charIDs := make([]string, cfg.CharCount)
		for j := 0; j < cfg.CharCount; j++ {
			char := &domain.Character{
				ID:          primitive.NewObjectID().Hex(),
				StoryID:     story.ID,
				Name:        fmt.Sprintf("Character %c", 'A'+j),
				Personality: map[string]any{"trait": "curious and determined"},
				MoralAlignment: "neutral",
				CreatedAt:   time.Now(),
			}
			if err := charRepo.Create(ctx, char); err != nil {
				slog.Error("create char failed", "error", err)
				continue
			}
			charIDs[j] = char.ID
		}

		sceneIDs := make([]string, cfg.SceneCount)
		for j := 0; j < cfg.SceneCount; j++ {
			scene := &domain.Scene{
				ID:        primitive.NewObjectID().Hex(),
				StoryID:   story.ID,
				Title:     fmt.Sprintf("Scene %d", j+1),
				Status:    domain.SceneStatusDraft,
				FlowType:  "dialogue",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := sceneRepo.Create(ctx, scene); err != nil {
				slog.Error("create scene failed", "error", err)
				continue
			}
			sceneIDs[j] = scene.ID

			edgeTypes := []string{"seq", "fork", "seq"}
			if j > 0 {
				et := edgeTypes[(j-1)%len(edgeTypes)]
				edge := &domain.SceneEdge{
					ID:          primitive.NewObjectID().Hex(),
					StoryID:     story.ID,
					FromSceneID: sceneIDs[j-1],
					ToSceneID:   scene.ID,
					Type:        et,
				}
				if err := edgeRepo.Create(ctx, edge); err != nil {
					slog.Warn("create edge failed", "error", err)
				}
			}

			loc := &domain.Location{
				ID:      primitive.NewObjectID().Hex(),
				StoryID: story.ID,
				Name:    fmt.Sprintf("Location %d", j+1),
			}
			if err := locRepo.Create(ctx, loc); err != nil {
				slog.Warn("create location failed", "error", err)
			}

			tl := &domain.TimelineEvent{
				ID:        primitive.NewObjectID().Hex(),
				StoryID:   story.ID,
				SceneID:   scene.ID,
				Title:     fmt.Sprintf("Event: %s", scene.Title),
				Order:     j + 1,
				CreatedAt: time.Now(),
			}
			if err := tlRepo.Create(ctx, tl); err != nil {
				slog.Warn("create timeline event failed", "error", err)
			}
		}

		slog.Info("seeded story", "id", story.ID, "title", story.Title, "scenes", len(sceneIDs))
	}

	fmt.Println("Seed complete.")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
