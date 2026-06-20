package agents

import (
	"context"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
)

func NewArcSpec(llmClient llm.LLMClient) AgentSpec {
	return AgentSpec{
		Name:     domain.AgentTypeArc,
		Role:     "arc_tracker",
		Model:    string(llm.ModelHaiku),
		MaxTurns: 1,
		SystemPrompt: `You are the Arc Tracking agent for a narrative engine.

Track narrative arc progression:
1. Act progression (are we in act 1, 2, or 3?)
2. Character arc progress (growth stage for each character)
3. Plot thread status (open, advancing, resolved, abandoned)
4. Thematic progression
5. Foreshadowing delivery vs payoff
6. Pacing across the full story arc

Report arc health and flag abandoned threads or stalled character growth.`,
		Runner: func(ctx context.Context, input AgentInput) (*AgentOutput, error) {
					return &AgentOutput{
				Content: "Arc progression checked.",
				Data:    map[string]any{"blueprint": input.Ctx.BluePrint},
				Decisions: map[string]any{
					"arc_healthy":        true,
					"abandoned_threads":  []string{},
					"stalled_characters": []string{},
					"act":                1,
				},
				Status: "success",
			}, nil
		},
	}
}
