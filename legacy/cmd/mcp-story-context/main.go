package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/premchand/story-builder/internal/config"
	mgorepo "github.com/premchand/story-builder/internal/repository/mongo"
)

func main() {
	cfg := config.FromEnv()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	db, err := mgorepo.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		slog.Error("mongo connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Client().Disconnect(ctx)

	mcpServer := server.NewMCPServer(
		"story-context",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	mcpServer.AddTool(registerGetCanon(), getCanonHandler(db))
	mcpServer.AddTool(registerGetCharacterState(), getCharacterStateHandler(db))
	mcpServer.AddTool(registerGetSceneHistory(), getSceneHistoryHandler(db))

	if err := server.ServeStdio(mcpServer); err != nil {
		slog.Error("mcp server error", "error", err)
		os.Exit(1)
	}
}

func registerGetCanon() mcp.Tool {
	return mcp.NewTool("get_canon",
		mcp.WithDescription("Query the canon for a story. Returns current canon state including pinned facts, world rules, and established narrative constraints."),
		mcp.WithString("story_id",
			mcp.Required(),
			mcp.Description("Story ID to query canon for"),
		),
	)
}

func getCanonHandler(db *mongo.Database) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		storyID, _ := args["story_id"].(string)
		if storyID == "" {
			return mcp.NewToolResultText("error: story_id is required"), nil
		}

		var story bson.M
		if err := db.Collection("stories").FindOne(ctx, bson.M{"id": storyID}).Decode(&story); err != nil {
			if err == mongo.ErrNoDocuments {
				return mcp.NewToolResultText(fmt.Sprintf("story %s not found", storyID)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("error: %v", err)), nil
		}

		canonPins, _ := story["canon_pins"].(bson.M)

		var deltas []bson.M
		cursor, err := db.Collection("canon_deltas").Find(ctx, bson.M{"story_id": storyID})
		if err == nil {
			cursor.All(ctx, &deltas)
		}

		result := map[string]any{
			"story_id":     storyID,
			"canon_pins":   canonPins,
			"canon_deltas": deltas,
		}
		return mcp.NewToolResultJSON(result)
	}
}

func registerGetCharacterState() mcp.Tool {
	return mcp.NewTool("get_character_state",
		mcp.WithDescription("Get the state of a character for a given scene — emotion, health, knowledge, relationships, and active goals."),
		mcp.WithString("story_id",
			mcp.Required(),
			mcp.Description("Story ID"),
		),
		mcp.WithString("character_id",
			mcp.Required(),
			mcp.Description("Character ID"),
		),
		mcp.WithString("scene_id",
			mcp.Description("Scene ID to get state for (returns latest if empty)"),
		),
	)
}

func getCharacterStateHandler(db *mongo.Database) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		storyID, _ := args["story_id"].(string)
		charID, _ := args["character_id"].(string)
		sceneID, _ := args["scene_id"].(string)
		if storyID == "" || charID == "" {
			return mcp.NewToolResultText("error: story_id and character_id are required"), nil
		}

		filter := bson.M{"story_id": storyID, "character_id": charID}
		if sceneID != "" {
			filter["scene_id"] = sceneID
		}

		var states []bson.M
		cursor, err := db.Collection("character_state").Find(ctx, filter)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("error: %v", err)), nil
		}
		cursor.All(ctx, &states)

		return mcp.NewToolResultJSON(map[string]any{
			"story_id":     storyID,
			"character_id": charID,
			"scene_id":     sceneID,
			"states":       states,
			"state_count":  len(states),
		})
	}
}

func registerGetSceneHistory() mcp.Tool {
	return mcp.NewTool("get_scene_history",
		mcp.WithDescription("Get generation and turn history for a scene — all previous LLM outputs, critic scores, and per-turn agent activity."),
		mcp.WithString("story_id",
			mcp.Required(),
			mcp.Description("Story ID"),
		),
		mcp.WithString("scene_id",
			mcp.Required(),
			mcp.Description("Scene/Node ID"),
		),
	)
}

func getSceneHistoryHandler(db *mongo.Database) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		storyID, _ := args["story_id"].(string)
		sceneID, _ := args["scene_id"].(string)
		if storyID == "" || sceneID == "" {
			return mcp.NewToolResultText("error: story_id and scene_id are required"), nil
		}

		var generations []bson.M
		genCursor, _ := db.Collection("generations").Find(ctx, bson.M{"story_id": storyID, "node_id": sceneID})
		genCursor.All(ctx, &generations)

		var turns []bson.M
		turnCursor, _ := db.Collection("scene_turns").Find(ctx, bson.M{"story_id": storyID, "scene_id": sceneID})
		turnCursor.All(ctx, &turns)

		return mcp.NewToolResultJSON(map[string]any{
			"story_id":         storyID,
			"scene_id":         sceneID,
			"generations":      generations,
			"generation_count": len(generations),
			"turns":            turns,
			"turn_count":       len(turns),
		})
	}
}

func _() {}
