package agents

import (
	"context"
	"fmt"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
)

func NewCriticSpec(llmClient llm.LLMClient) AgentSpec {
	return AgentSpec{
		Name:     domain.AgentTypeCritic,
		Role:     "critic",
		Model:    string(llm.ModelHaiku),
		MaxTurns: 1,
		SystemPrompt: `You are the Critic agent for a narrative engine.

Judge whether the scene was dramatically useful. Score 0.0-1.0:
1. Did the scene advance the plot? (0.0-0.3)
2. Did conflict escalate or resolve? (0.0-0.25)
3. Did character arc move? (0.0-0.2)
4. Did stakes become clearer? (0.0-0.15)
5. Was there meaningful emotional change? (0.0-0.1)

Also flag:
- Redundant scenes (already covered by summary)
- Static scenes (nothing changed)
- Passive scenes (things happened to characters, not by them)
- Wasted opportunities (setup without payoff)`,
		Runner: func(ctx context.Context, input AgentInput) (*AgentOutput, error) {
			score := 0.7
			critiques := make([]string, 0)
			if len(input.Ctx.Turns) == 0 {
				score = 0.3
				critiques = append(critiques, "no turns executed")
			}

			return &AgentOutput{
				Content: fmt.Sprintf("Scene score: %.2f", score),
				Data:    map[string]any{"critiques": critiques},
				Decisions: map[string]any{
					"score":     score,
					"critiques": critiques,
					"useful":    score >= 0.5,
				},
				Status: "success",
			}, nil
		},
	}
}
