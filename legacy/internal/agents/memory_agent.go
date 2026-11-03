package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/trace"
)

func NewMemorySpec(llmClient llm.LLMClient) AgentSpec {
	return AgentSpec{
		Name:     domain.AgentTypeMemory,
		Role:     "memory_keeper",
		Model:    string(llm.ModelLocal),
		MaxTurns: 1,
		SystemPrompt: `You are the Memory agent for a narrative engine.

Analyze the latest scene and suggest memory updates:
1. Important events characters should remember
2. Dialogue lines worth preserving
3. Observations characters made
4. Relationship changes to record
5. Injuries or significant physical changes
6. World lore revelations

Output JSON:
{"memory_suggestions": [{"character": "string", "type": "event|dialogue|observation|injury|relationship_change", "content": "string", "importance": 0.0-1.0}], "cleanup_needed": bool, "summary": "string"}`,
		Runner: func(ctx context.Context, input AgentInput) (*AgentOutput, error) {
			ctx, span := trace.StartSpan(ctx, "agent.memory."+input.Directive)
			defer trace.End(span)

			userMsg := buildMemoryPrompt(input)

			resp, err := llmClient.Complete(ctx, llm.CompletionRequest{
				Model:        llm.ModelLocal,
				System:       "You are a memory management agent. Output valid JSON only.",
				UserMessage:  userMsg,
				Temperature:  0.0,
				MaxTokens:    1024,
				ValidateJSON: true,
			})
			if err != nil {
				trace.SetError(span, err)
				return nil, fmt.Errorf("memory agent llm call: %w", err)
			}

			var parsed struct {
				Suggestions []map[string]any `json:"memory_suggestions"`
				Cleanup     bool              `json:"cleanup_needed"`
				Summary     string            `json:"summary"`
			}
			if err := json.Unmarshal([]byte(resp.Content), &parsed); err != nil {
				parsed.Summary = "memory analysis incomplete"
			}

			memCount := 0
			for _, mems := range input.Ctx.Memories {
				memCount += len(mems)
			}

			content := fmt.Sprintf("Memory: %s (%d existing, %d suggestions)", parsed.Summary, memCount, len(parsed.Suggestions))

			return &AgentOutput{
				Content: content,
				Data:    map[string]any{"suggestions": parsed.Suggestions, "existing_count": memCount},
				Decisions: map[string]any{
					"suggestions":   parsed.Suggestions,
					"cleanup_needed": parsed.Cleanup,
					"existing":      memCount,
				},
				Status: "success",
			}, nil
		},
	}
}

func buildMemoryPrompt(input AgentInput) string {
	var b strings.Builder

	b.WriteString("=== SCENE ===\n")
	b.WriteString(fmt.Sprintf("Title: %s\n", input.Ctx.Scene.Title))
	if input.Ctx.Scene.BeatIntent != "" {
		b.WriteString(fmt.Sprintf("Intent: %s\n", input.Ctx.Scene.BeatIntent))
	}
	b.WriteString("\n")

	for _, c := range input.Ctx.Characters {
		b.WriteString(fmt.Sprintf("=== CHARACTER: %s ===\n", c.Name))
		if mems, ok := input.Ctx.Memories[c.CharID]; ok && len(mems) > 0 {
			b.WriteString("Existing memories:\n")
			for _, m := range mems {
				b.WriteString(fmt.Sprintf("- [%.1f] [%s] %s\n", m.Importance, m.Type, m.Content))
			}
		} else {
			b.WriteString("No existing memories.\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("=== LATEST TURNS ===\n")
	for _, t := range input.Ctx.Turns {
		b.WriteString(fmt.Sprintf("[%s]: %s\n\n", t.Role, t.Output))
	}

	b.WriteString("Analyze what these characters should remember. Output JSON with memory suggestions.\n")
	return b.String()
}
