package agents

import (
	"context"
	"fmt"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
)

func NewCanonGuardSpec(llmClient llm.LLMClient, validateSvc llm.ValidationService) AgentSpec {
	return AgentSpec{
		Name:     domain.AgentTypeCanonGuard,
		Role:     "canon_guard",
		Model:    string(llm.ModelHaiku),
		MaxTurns: 1,
		SystemPrompt: `You are the Continuity/Canon Guard agent for a narrative engine.

Your job is to verify story truth consistency:
1. Character alive/dead status
2. Location continuity (characters can't be two places at once)
3. Timeline consistency (events in correct order)
4. Relationship state consistency
5. Canon facts are respected
6. Unresolved plot threads haven't been forgotten
7. Object/artifact continuity

Flag any violations with severity.`,
		Runner: func(ctx context.Context, input AgentInput) (*AgentOutput, error) {
			violations := make([]map[string]any, 0)

			for _, delta := range input.Ctx.CanonDeltas {
				if delta.Confidence < 0.3 {
					violations = append(violations, map[string]any{
						"type":     "low_confidence",
						"fact":     delta.Fact,
						"severity": "warning",
					})
				}
			}

			return &AgentOutput{
				Content: fmt.Sprintf("Canon check complete: %d violations found", len(violations)),
				Data:    map[string]any{"violations": violations},
				Decisions: map[string]any{
					"violations":  violations,
					"canon_clean": len(violations) == 0,
				},
				Status: "success",
			}, nil
		},
	}
}
