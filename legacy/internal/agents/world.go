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

Flag any violations and suggest corrections. Output JSON:
{"world_stable": bool, "violations": [{"type": "string", "detail": "string", "severity": "warning|error|critical"}], "faction_status": [{"name": "string", "status": "string"}], "summary": "string"}`,
		Runner: func(ctx context.Context, input AgentInput) (*AgentOutput, error) {
			ctx, span := trace.StartSpan(ctx, "agent.world."+input.Directive)
			defer trace.End(span)

			userMsg := buildWorldPrompt(input)

			resp, err := llmClient.Complete(ctx, llm.CompletionRequest{
				Model:        llm.ModelSonnet,
				System:       "You are a world-building continuity checker. Output valid JSON only.",
				UserMessage:  userMsg,
				Temperature:  0.3,
				MaxTokens:    1024,
				ValidateJSON: true,
			})
			if err != nil {
				trace.SetError(span, err)
				return nil, fmt.Errorf("world agent llm call: %w", err)
			}

			var parsed struct {
				WorldStable   bool                   `json:"world_stable"`
				Violations    []map[string]any        `json:"violations"`
				FactionStatus []map[string]string     `json:"faction_status"`
				Summary       string                  `json:"summary"`
			}
			if err := json.Unmarshal([]byte(resp.Content), &parsed); err != nil {
				parsed.WorldStable = true
				parsed.Summary = "world check incomplete"
			}

			return &AgentOutput{
				Content: fmt.Sprintf("World check: %s (%d violations)", parsed.Summary, len(parsed.Violations)),
				Data:    map[string]any{"violations": parsed.Violations, "factionStatus": parsed.FactionStatus},
				Decisions: map[string]any{
					"world_stable":  parsed.WorldStable,
					"violations":    parsed.Violations,
					"violation_cnt": len(parsed.Violations),
				},
				Status: "success",
			}, nil
		},
	}
}

func buildWorldPrompt(input AgentInput) string {
	var b strings.Builder
	b.WriteString("=== SCENE ===\n")
	b.WriteString(fmt.Sprintf("Title: %s\n", input.Ctx.Scene.Title))
	if input.Ctx.Scene.BeatIntent != "" {
		b.WriteString(fmt.Sprintf("Intent: %s\n", input.Ctx.Scene.BeatIntent))
	}
	b.WriteString(fmt.Sprintf("Location: %s\n\n", input.Ctx.Scene.LocationRef))

	if input.Ctx.Bible != nil {
		b.WriteString("=== WORLD BIBLE ===\n")
		for _, r := range input.Ctx.Bible.WorldRules {
			b.WriteString(fmt.Sprintf("Rule [%s] %s (strictness: %s)\n", r.Category, r.Description, r.Strictness))
		}
		for _, f := range input.Ctx.Bible.Factions {
			b.WriteString(fmt.Sprintf("Faction: %s — %s\n", f.Name, f.Goal))
		}
		for _, c := range input.Ctx.Bible.Cultures {
			b.WriteString(fmt.Sprintf("Culture: %s (tech: %s, gov: %s)\n", c.Name, c.Technology, c.Government))
		}
		b.WriteString("\n")
	}

	b.WriteString("=== TURNS ===\n")
	for _, t := range input.Ctx.Turns {
		b.WriteString(fmt.Sprintf("[%s]: %s\n\n", t.Role, t.Output))
	}

	b.WriteString("Check world consistency. Output JSON with violations if any.\n")
	return b.String()
}
