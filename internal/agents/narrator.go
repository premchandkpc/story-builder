package agents

import (
	"context"
	"fmt"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
)

func NewNarratorSpec(llmClient llm.LLMClient, proseSvc llm.ProseService) AgentSpec {
	return AgentSpec{
		Name:     domain.AgentTypeNarrator,
		Role:     "narrator",
		Model:    string(llm.ModelSonnet),
		MaxTurns: 1,
		SystemPrompt: `You are the Narrator agent for a narrative engine.

Your job is to weave character actions and dialogue into coherent narrative prose.

Rules:
1. Frame each turn with narrative context (where, when, sensory details)
2. Stitch character outputs into a cohesive flow
3. Add pacing, transition, and atmosphere
4. Maintain consistent POV
5. Do NOT invent new character actions — only narrate what the Character agents produced
6. Keep the prose tight and vivid`,
		Runner: func(ctx context.Context, input AgentInput) (*AgentOutput, error) {
			phase := input.Directive
			return &AgentOutput{
				Content: fmt.Sprintf("Narrative frame for %s turn.", phase),
				Data:    map[string]any{"phase": phase, "sceneId": input.Ctx.SceneID},
				Decisions: map[string]any{
					"tone":       input.Ctx.Scene.Tone,
					"pov":        input.Ctx.Scene.POV,
					"pace":       "normal",
				},
				Status: "success",
			}, nil
		},
	}
}
