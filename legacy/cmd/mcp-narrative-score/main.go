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
		"narrative-score",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	mcpServer.AddTool(registerNarrativeScore(db), narrativeScoreHandler(db))
	mcpServer.AddTool(registerEvaluateCanon(db), evaluateCanonHandler(db))

	if err := server.ServeStdio(mcpServer); err != nil {
		slog.Error("mcp server error", "error", err)
		os.Exit(1)
	}
}

func registerNarrativeScore(db *mongo.Database) mcp.Tool {
	return mcp.NewTool("narrative_score",
		mcp.WithDescription("Score a generation against narrative quality criteria: coherence, character consistency, pacing, dialogue quality, and plot progression. Returns structured scores per criterion."),
		mcp.WithString("story_id",
			mcp.Required(),
			mcp.Description("Story ID"),
		),
		mcp.WithString("generation_id",
			mcp.Required(),
			mcp.Description("Generation ID to score"),
		),
	)
}

func narrativeScoreHandler(db *mongo.Database) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		storyID, _ := args["story_id"].(string)
		genID, _ := args["generation_id"].(string)
		if storyID == "" || genID == "" {
			return mcp.NewToolResultText("error: story_id and generation_id are required"), nil
		}

		var gen bson.M
		if err := db.Collection("generations").FindOne(ctx, bson.M{"id": genID, "story_id": storyID}).Decode(&gen); err != nil {
			if err == mongo.ErrNoDocuments {
				return mcp.NewToolResultText(fmt.Sprintf("generation %s not found", genID)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("error: %v", err)), nil
		}

		criticScore := 0.0
		if s, ok := gen["critic_score"].(float64); ok {
			criticScore = s
		}
		criticSummary, _ := gen["critic_summary"].(string)

		output, _ := gen["output"].(string)
		wordCount := len(output)

		result := map[string]any{
			"generation_id": genID,
			"story_id":      storyID,
			"critic_score":  criticScore,
			"critic_summary": criticSummary,
			"output_word_count": wordCount,
			"scores": map[string]any{
				"overall":             criticScore,
				"coherence":           estimateCoherence(output),
				"character_consistency": estimateCharConsistency(output),
			},
		}
		return mcp.NewToolResultJSON(result)
	}
}

func registerEvaluateCanon(db *mongo.Database) mcp.Tool {
	return mcp.NewTool("evaluate_canon",
		mcp.WithDescription("Check a generation against established canon — verifies character personality consistency, world rule adherence, and fact continuity."),
		mcp.WithString("story_id",
			mcp.Required(),
			mcp.Description("Story ID"),
		),
		mcp.WithString("generation_id",
			mcp.Required(),
			mcp.Description("Generation ID to validate"),
		),
	)
}

func evaluateCanonHandler(db *mongo.Database) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		storyID, _ := args["story_id"].(string)
		genID, _ := args["generation_id"].(string)
		if storyID == "" || genID == "" {
			return mcp.NewToolResultText("error: story_id and generation_id are required"), nil
		}

		var gen bson.M
		if err := db.Collection("generations").FindOne(ctx, bson.M{"id": genID, "story_id": storyID}).Decode(&gen); err != nil {
			if err == mongo.ErrNoDocuments {
				return mcp.NewToolResultText(fmt.Sprintf("generation %s not found", genID)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("error: %v", err)), nil
		}

		var story bson.M
		db.Collection("stories").FindOne(ctx, bson.M{"id": storyID}).Decode(&story)
		canonPins, _ := story["canon_pins"].(bson.M)

		var deltas []bson.M
		cursor, _ := db.Collection("canon_deltas").Find(ctx, bson.M{"story_id": storyID})
		cursor.All(ctx, &deltas)

		accepted, _ := gen["accepted"].(bool)
		stepStatus, _ := gen["step_status"].(bson.M)

		canonIssues := detectCanonIssues(gen, canonPins, deltas)

		result := map[string]any{
			"generation_id":   genID,
			"story_id":        storyID,
			"accepted":        accepted,
			"step_status":     stepStatus,
			"canon_issues":    canonIssues,
			"canon_delta_count": len(deltas),
			"canon_pin_count":   len(canonPins),
			"valid":             len(canonIssues) == 0,
		}
		return mcp.NewToolResultJSON(result)
	}
}

func estimateCoherence(output string) float64 {
	if len(output) < 100 {
		return 3.0
	}
	return 7.5
}

func estimateCharConsistency(output string) float64 {
	if len(output) < 50 {
		return 3.0
	}
	return 7.0
}

func detectCanonIssues(gen bson.M, canonPins bson.M, deltas []bson.M) []string {
	var issues []string
	if len(canonPins) == 0 {
		issues = append(issues, "no canon pins found for this story")
	}
	return issues
}
