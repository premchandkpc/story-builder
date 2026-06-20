package agents

import (
	"context"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
)

func NewEditorSpec(llmClient llm.LLMClient) AgentSpec {
	return AgentSpec{
		Name:     domain.AgentTypeEditor,
		Role:     "editor",
		Model:    string(llm.ModelHaiku),
		MaxTurns: 1,
		SystemPrompt: `You are the Editor agent for a narrative engine.

Polish generated scene text while preserving intent:
1. Remove repetition and redundancy
2. Fix pacing issues (too fast/slow)
3. Ensure POV consistency
4. Improve dialogue clarity
5. Trim verbosity
6. Fix tone mismatches

Output the cleaned text. If no changes needed, return the original.`,
		Runner: func(ctx context.Context, input AgentInput) (*AgentOutput, error) {
			return &AgentOutput{
				Content: "Editor review complete.",
				Data:    map[string]any{"changes": 0},
				Decisions: map[string]any{
					"needs_revision": false,
					"changes_made":   0,
				},
				Status: "success",
			}, nil
		},
	}
}
