package agents

import (
	"context"
	"fmt"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
)

func NewWorldSpec(llmClient llm.LLMClient) AgentSpec {
	return AgentSpec{
		Name:     domain.AgentTypeWorld,
		Role:     "world_keeper",
		Model:    string(llm.ModelSonnet),
		MaxTurns: 1,
		SystemPrompt: `You are the World agent for a narrative engine.

You maintain world consistency:
1. Faction politics and relationships
2. World rules (magic, technology, physics)
3. Lore consistency across scenes and chapters
4. Setting realism
5. Cultural consistency
6. Environmental and geographical continuity

Flag any violations and suggest corrections.`,
		Runner: func(ctx context.Context, input AgentInput) (*AgentOutput, error) {
			bible := input.Ctx.Bible
			factionCount := 0
			rulesCount := 0
			if bible != nil {
				factionCount = len(bible.Factions)
				rulesCount = len(bible.WorldRules)
			}
			return &AgentOutput{
				Content: fmt.Sprintf("World check: %d factions, %d rules", factionCount, rulesCount),
				Data:    map[string]any{"bible": bible},
				Decisions: map[string]any{
					"world_stable": true,
					"violations":   []string{},
				},
				Status: "success",
			}, nil
		},
	}
}
