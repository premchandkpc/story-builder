package agents

import (
	"context"
	"fmt"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
)

func NewStateExtractorSpec(llmClient llm.LLMClient, extractSvc llm.ExtractionService) AgentSpec {
	return AgentSpec{
		Name:     domain.AgentTypeStateExtract,
		Role:     "state_extractor",
		Model:    string(llm.ModelLocal),
		MaxTurns: 1,
		SystemPrompt: `You are the State Extraction agent for a narrative engine.

After a scene completes, extract all structured state changes:
1. Character emotional state changes
2. New facts learned by each character
3. Relationship changes between characters
4. Timeline events (deaths, births, discoveries, etc.)
5. Location changes
6. Wounds, promises, betrayals, discoveries
7. Unresolved hooks for future scenes
8. Changes to world state (factions, politics, environment)

Output structured deltas for each category.`,
		Runner: func(ctx context.Context, input AgentInput) (*AgentOutput, error) {
			deltas := make([]map[string]any, 0)

			for _, cs := range input.Ctx.CharStates {
				if len(cs.Changes) > 0 {
					deltas = append(deltas, map[string]any{
						"character_id": cs.CharacterID,
						"changes":      cs.Changes,
						"new_mood":     cs.EmotionalState,
						"new_location": cs.Location,
					})
				}
			}

			return &AgentOutput{
				Content: fmt.Sprintf("Extracted %d state deltas", len(deltas)),
				Data:    map[string]any{"deltas": deltas},
				Decisions: map[string]any{
					"deltas":        deltas,
					"needs_review":  false,
				},
				Status: "success",
			}, nil
		},
	}
}
