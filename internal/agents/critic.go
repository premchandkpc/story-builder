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
- Wasted opportunities (setup without payoff)

Output valid JSON with format:
{"score": 0.0-1.0, "critiques": ["string"], "strengths": ["string"], "summary": "one-line verdict"}`,
		Runner: func(ctx context.Context, input AgentInput) (*AgentOutput, error) {
			ctx, span := trace.StartSpan(ctx, "agent.critic."+input.Directive)
			defer trace.End(span)

			userMsg := buildCriticPrompt(input)

			resp, err := llmClient.Complete(ctx, llm.CompletionRequest{
				Model:        llm.ModelHaiku,
				System:       "You are a narrative critic. Output valid JSON only.",
				UserMessage:  userMsg,
				Temperature:  0.0,
				MaxTokens:    1024,
				ValidateJSON: true,
			})
			if err != nil {
				trace.SetError(span, err)
				return nil, fmt.Errorf("critic agent llm call: %w", err)
			}

			var parsed struct {
				Score     float64  `json:"score"`
				Critiques []string `json:"critiques"`
				Strengths []string `json:"strengths"`
				Summary   string   `json:"summary"`
			}
			if err := json.Unmarshal([]byte(resp.Content), &parsed); err != nil {
				parsed = struct {
					Score     float64  `json:"score"`
					Critiques []string `json:"critiques"`
					Strengths []string `json:"strengths"`
					Summary   string   `json:"summary"`
				}{Score: 0.5, Critiques: []string{"parse error"}, Summary: "review incomplete"}
			}

			content := fmt.Sprintf("Scene score: %.2f — %s", parsed.Score, parsed.Summary)
			if len(parsed.Critiques) > 0 {
				content += "\nIssues: " + strings.Join(parsed.Critiques, "; ")
			}

			return &AgentOutput{
				Content: content,
				Data: map[string]any{
					"critiques": parsed.Critiques,
					"strengths": parsed.Strengths,
					"summary":   parsed.Summary,
				},
				Decisions: map[string]any{
					"score":     parsed.Score,
					"critiques": parsed.Critiques,
					"useful":    parsed.Score >= 0.5,
					"summary":   parsed.Summary,
				},
				Status: "success",
			}, nil
		},
	}
}

func buildCriticPrompt(input AgentInput) string {
	var b strings.Builder

	b.WriteString("=== SCENE ===\n")
	b.WriteString(fmt.Sprintf("Title: %s\n", input.Ctx.Scene.Title))
	if input.Ctx.Scene.BeatIntent != "" {
		b.WriteString(fmt.Sprintf("Intent: %s\n", input.Ctx.Scene.BeatIntent))
	}
	b.WriteString(fmt.Sprintf("Tone: %s | POV: %s | Flow: %s\n\n", input.Ctx.Scene.Tone, input.Ctx.Scene.POV, input.Ctx.Scene.FlowType))

	b.WriteString("=== CHARACTERS ===\n")
	for _, c := range input.Ctx.Characters {
		b.WriteString(fmt.Sprintf("- %s: %s\n", c.Name, c.Persona))
	}
	b.WriteString("\n")

	b.WriteString("=== TURNS ===\n")
	for _, t := range input.Ctx.Turns {
		b.WriteString(fmt.Sprintf("[%s] %s\n\n", t.Role, t.Output))
	}

	b.WriteString("Evaluate this scene. Output JSON with score (0.0-1.0), critiques, strengths, and summary.\n")
	return b.String()
}
