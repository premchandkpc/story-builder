package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

Report arc health and flag abandoned threads or stalled character growth.
Output JSON:
{"arc_healthy": bool, "act": int, "character_arcs": [{"name": "string", "status": "string"}], "abandoned_threads": [{"thread": "string", "detail": "string"}], "stalled_characters": [{"name": "string", "reason": "string"}], "pacing": "string", "summary": "string"}`,
		Runner: func(ctx context.Context, input AgentInput) (*AgentOutput, error) {
			userMsg := buildArcPrompt(input)

			resp, err := llmClient.Complete(ctx, llm.CompletionRequest{
				Model:        llm.ModelHaiku,
				System:       "You are an arc/tracking analyst. Output valid JSON only.",
				UserMessage:  userMsg,
				Temperature:  0.2,
				MaxTokens:    1024,
				ValidateJSON: true,
			})
			if err != nil {
				return nil, fmt.Errorf("arc agent llm call: %w", err)
			}

			var parsed struct {
				ArcHealthy   bool       `json:"arc_healthy"`
				Act          int        `json:"act"`
				Summary      string     `json:"summary"`
			}
			if err := json.Unmarshal([]byte(resp.Content), &parsed); err != nil {
				parsed.ArcHealthy = true
				parsed.Act = 1
			}

			content := fmt.Sprintf("Arc check: %s (act %d)", parsed.Summary, parsed.Act)

			return &AgentOutput{
				Content: content,
				Data:    map[string]any{"blueprint": input.Ctx.BluePrint},
				Decisions: map[string]any{
					"arc_healthy": parsed.ArcHealthy,
					"act":         parsed.Act,
					"summary":     parsed.Summary,
				},
				Status: "success",
			}, nil
		},
	}
}

func buildArcPrompt(input AgentInput) string {
	var b strings.Builder
	b.WriteString("=== SCENE ===\n")
	b.WriteString(fmt.Sprintf("Title: %s\n", input.Ctx.Scene.Title))
	if input.Ctx.Scene.BeatIntent != "" {
		b.WriteString(fmt.Sprintf("Intent: %s\n", input.Ctx.Scene.BeatIntent))
	}
	b.WriteString(fmt.Sprintf("Flow: %s\n\n", input.Ctx.Scene.FlowType))

	if input.Ctx.BluePrint != nil {
		b.WriteString("=== BLUEPRINT ===\n")
		for _, act := range input.Ctx.BluePrint.Acts {
			b.WriteString(fmt.Sprintf("Act %d: %s — %s\n", act.Number, act.Title, act.Summary))
		}
		for _, c := range input.Ctx.BluePrint.CharacterArcs {
			b.WriteString(fmt.Sprintf("CharacterArc: %s — want=%s need=%s stage=%s\n", c.CharacterName, c.Want, c.Need, c.GrowthStage))
		}
		for _, pt := range input.Ctx.BluePrint.PlotThreads {
			b.WriteString(fmt.Sprintf("Thread: %s [%s]\n", pt.Description, pt.Status))
		}
		b.WriteString("\n")
	}

	b.WriteString("=== CHARACTERS ===\n")
	for _, c := range input.Ctx.Characters {
		b.WriteString(fmt.Sprintf("- %s: persona=%s arc=%s\n", c.Name, c.Persona, c.ArcType))
	}
	b.WriteString("\n=== TURNS ===\n")
	for _, t := range input.Ctx.Turns {
		b.WriteString(fmt.Sprintf("[%s]: %s\n\n", t.Role, t.Output))
	}

	b.WriteString("Evaluate arc progression. Output JSON.\n")
	return b.String()
}
