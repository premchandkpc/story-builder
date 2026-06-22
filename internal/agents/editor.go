package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/trace"
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

Output only the polished prose. If no changes needed, return the original.`,
		Runner: func(ctx context.Context, input AgentInput) (*AgentOutput, error) {
			ctx, span := trace.StartSpan(ctx, "agent.editor."+input.Directive)
			defer trace.End(span)

			narratorOutput := ""
			for _, t := range input.Ctx.Turns {
				if t.Role == "narrator" {
					narratorOutput = t.Output
				}
			}
			if narratorOutput == "" {
				for _, t := range input.Ctx.Turns {
					if t.Output != "" {
						narratorOutput = t.Output
					}
				}
			}
			if narratorOutput == "" {
				return &AgentOutput{
					Content:   "",
					Data:      map[string]any{"changes": 0},
					Decisions: map[string]any{"needs_revision": false, "changes_made": 0},
					Status:    "success",
				}, nil
			}

			userMsg := fmt.Sprintf(`Polish the following narrative text. Preserve all story content, characters, and events. Fix only prose quality issues.

Tone: %s | POV: %s

=== TEXT TO POLISH ===
%s`, input.Ctx.Scene.Tone, input.Ctx.Scene.POV, narratorOutput)

			resp, err := llmClient.Complete(ctx, llm.CompletionRequest{
				Model:       llm.ModelHaiku,
				System:      "You are a prose editor. Output only the polished text, no commentary.",
				UserMessage: userMsg,
				Temperature: 0.2,
				MaxTokens:   4096,
			})
			if err != nil {
				trace.SetError(span, err)
				return nil, fmt.Errorf("editor agent llm call: %w", err)
			}

			changed := strings.TrimSpace(resp.Content) != strings.TrimSpace(narratorOutput)

			return &AgentOutput{
				Content: resp.Content,
				Data:    map[string]any{"changes": 0},
				Decisions: map[string]any{
					"needs_revision": changed,
					"changes_made":   0,
				},
				Status: "success",
			}, nil
		},
	}
}
