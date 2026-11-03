//go:build integration

package integration

import (
	"context"
	"fmt"

	"github.com/premchand/story-builder/internal/llm"
)

type mockProseService struct {
	generateFn func(ctx context.Context, params llm.PromptParams) (*llm.CompletionResponse, error)
}

func (m *mockProseService) GenerateScene(ctx context.Context, params llm.PromptParams) (*llm.CompletionResponse, error) {
	if m.generateFn != nil {
		return m.generateFn(ctx, params)
	}
	return &llm.CompletionResponse{
		Content: fmt.Sprintf("Scene prose for %s. Characters interact and the plot advances.", params.BeatIntent),
		Model:   "mock-sonnet",
	}, nil
}

type mockExtractionService struct{}

func (m *mockExtractionService) ExtractState(ctx context.Context, sceneText string, roster map[string]string) (*llm.StateDeltas, error) {
	return &llm.StateDeltas{Deltas: []llm.StateDelta{}, OpenThreads: []string{}}, nil
}

type mockSummaryService struct{}

func (m *mockSummaryService) UpdateSummary(ctx context.Context, previousSummary, newScene string) (string, error) {
	cut := len(newScene)
	if cut > 80 {
		cut = 80
	}
	return previousSummary + "\n" + newScene[:cut] + "...", nil
}

type mockValidationService struct{}

func (m *mockValidationService) ValidateAgainstCanon(ctx context.Context, canonXML, charState, draft string) (map[string]any, error) {
	return map[string]any{"violations": []any{}}, nil
}

type mockEmbeddingService struct{}

func (m *mockEmbeddingService) GenerateEmbedding(_ context.Context, _ string) ([]float64, error) {
	return []float64{0.1, 0.2, 0.3}, nil
}

func (m *mockEmbeddingService) Model() string { return "mock-embed" }

type mockOutlineService struct{}

func (m *mockOutlineService) GenerateOutline(ctx context.Context, synopsis string) (*llm.StoryOutline, error) {
	return &llm.StoryOutline{
		Title:    "Mock Story",
		Synopsis: synopsis,
		Characters: []llm.StoryOutlineCharacter{
			{Name: "Hero", Persona: "protagonist"},
		},
		Beats: []llm.StoryOutlineBeat{
			{Title: "Chapter 1", BeatIntent: "The hero begins", CharacterNames: []string{"Hero"}, Tone: "neutral", TargetWords: 500, Act: 1},
		},
		Edges: []llm.StoryOutlineEdge{
			{From: "Chapter 1", To: "Chapter 1", Type: "seq"},
		},
	}, nil
}
