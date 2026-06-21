package agents

import (
	"context"
	"fmt"
	"strings"

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
			sceneText := buildSceneText(input)
			roster := buildRoster(input)

			stateDeltas, err := extractSvc.ExtractState(ctx, sceneText, roster)
			if err != nil {
				return nil, fmt.Errorf("state extraction: %w", err)
			}

			deltas := make([]map[string]any, 0, len(stateDeltas.Deltas))
			canonDeltas := make([]domain.CanonDelta, 0, len(stateDeltas.Deltas))

			for _, sd := range stateDeltas.Deltas {
				deltas = append(deltas, map[string]any{
					"character":           sd.Character,
					"new_location":        sd.NewLocation,
					"learned":             sd.Learned,
					"mood":                sd.Mood,
					"relationship_changes": sd.RelationshipChanges,
					"items_gained":        sd.ItemsGained,
					"items_lost":          sd.ItemsLost,
				})

				fact := extractFact(sd)
				if fact != "" {
					cat := "character_state"
					if sd.NewLocation != "" {
						cat = "location"
					}
					canonDeltas = append(canonDeltas, domain.CanonDelta{
						SceneID:    input.Ctx.SceneID,
						StoryID:    input.Ctx.StoryID,
						Category:   cat,
						Fact:       fact,
						Confidence: 0.8,
					})
				}
			}

			for _, thread := range stateDeltas.OpenThreads {
				canonDeltas = append(canonDeltas, domain.CanonDelta{
					SceneID:    input.Ctx.SceneID,
					StoryID:    input.Ctx.StoryID,
					Category:   "plot_thread",
					Fact:       thread,
					Confidence: 0.6,
				})
			}

			return &AgentOutput{
				Content: fmt.Sprintf("Extracted %d state deltas, %d canon updates", len(deltas), len(canonDeltas)),
				Data:    map[string]any{"deltas": deltas, "canonDeltas": canonDeltas},
				Decisions: map[string]any{
					"deltas":       deltas,
					"canon_deltas": canonDeltas,
					"needs_review": false,
				},
				Status: "success",
			}, nil
		},
	}
}

func buildSceneText(input AgentInput) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Scene: %s\n", input.Ctx.Scene.Title))
	if input.Ctx.Scene.BeatIntent != "" {
		b.WriteString(fmt.Sprintf("Intent: %s\n", input.Ctx.Scene.BeatIntent))
	}
	b.WriteString("\n=== NARRATIVE ===\n")
	for _, t := range input.Ctx.Turns {
		b.WriteString(fmt.Sprintf("%s\n\n", t.Output))
	}
	return b.String()
}

func buildRoster(input AgentInput) map[string]string {
	roster := make(map[string]string, len(input.Ctx.Characters))
	for _, c := range input.Ctx.Characters {
		roster[c.CharID] = c.Name
	}
	return roster
}

func extractFact(sd llm.StateDelta) string {
	var parts []string
	if sd.Mood != "" {
		parts = append(parts, fmt.Sprintf("mood: %s", sd.Mood))
	}
	if sd.NewLocation != "" {
		parts = append(parts, fmt.Sprintf("location: %s", sd.NewLocation))
	}
	if len(sd.Learned) > 0 {
		parts = append(parts, fmt.Sprintf("learned: %s", strings.Join(sd.Learned, ", ")))
	}
	if len(sd.ItemsGained) > 0 {
		parts = append(parts, fmt.Sprintf("gained: %s", strings.Join(sd.ItemsGained, ", ")))
	}
	if len(sd.ItemsLost) > 0 {
		parts = append(parts, fmt.Sprintf("lost: %s", strings.Join(sd.ItemsLost, ", ")))
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("%s: %s", sd.Character, strings.Join(parts, "; "))
}
