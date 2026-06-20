package agents

import (
	"context"
	"fmt"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
)

func NewMemorySpec(llmClient llm.LLMClient) AgentSpec {
	return AgentSpec{
		Name:     domain.AgentTypeMemory,
		Role:     "memory_keeper",
		Model:    string(llm.ModelLocal),
		MaxTurns: 1,
		SystemPrompt: `You are the Memory agent for a narrative engine.

Maintain layered memory across the story:
1. Character memory — what each character knows and remembers
2. World memory — lore, history, faction relations
3. Scene memory — what happened in each scene
4. Story memory — full narrative arc
5. Narrative memory — authorial intent, themes, dramatic questions

Each memory layer feeds different agents:
- Character agents access character memory
- World agent accesses world memory  
- Director accesses narrative memory
- Critic accesses all layers for evaluation`,
		Runner: func(ctx context.Context, input AgentInput) (*AgentOutput, error) {
			memCount := 0
			for _, mems := range input.Ctx.Memories {
				memCount += len(mems)
			}
			return &AgentOutput{
				Content: fmt.Sprintf("Memory check: %d memories across %d characters", memCount, len(input.Ctx.Memories)),
				Data:    map[string]any{"memory_count": memCount},
				Decisions: map[string]any{
					"memory_count":   memCount,
					"needs_cleanup":  false,
				},
				Status: "success",
			}, nil
		},
	}
}
