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

func NewCanonGuardSpec(llmClient llm.LLMClient, validateSvc llm.ValidationService) AgentSpec {
	return AgentSpec{
		Name:     domain.AgentTypeCanonGuard,
		Role:     "canon_guard",
		Model:    string(llm.ModelHaiku),
		MaxTurns: 1,
		SystemPrompt: `You are the Continuity/Canon Guard agent for a narrative engine.

Your job is to verify story truth consistency:
1. Character alive/dead status
2. Location continuity (characters can't be two places at once)
3. Timeline consistency (events in correct order)
4. Relationship state consistency
5. Canon facts are respected
6. Unresolved plot threads haven't been forgotten
7. Object/artifact continuity

Flag any violations with severity.`,
		Runner: func(ctx context.Context, input AgentInput) (*AgentOutput, error) {
			ctx, span := trace.StartSpan(ctx, "agent.canon_guard."+input.Directive)
			defer trace.End(span)

			violations := runRuleChecks(input)

			llmViolations, err := runLLMCanonCheck(ctx, llmClient, input)
			if err == nil && len(llmViolations) > 0 {
				violations = append(violations, llmViolations...)
			}

			status := "success"
			if len(violations) > 0 {
				status = "violations_found"
			}

			content := "Canon check passed"
			if len(violations) > 0 {
				sevCount := map[string]int{}
				for _, v := range violations {
					sev, _ := v["severity"].(string)
					sevCount[sev]++
				}
				var parts []string
				for _, s := range []string{"critical", "error", "warning", "info"} {
					if n := sevCount[s]; n > 0 {
						parts = append(parts, fmt.Sprintf("%d %s", n, s))
					}
				}
				content = fmt.Sprintf("Canon check: %d violations (%s)", len(violations), strings.Join(parts, ", "))
			}

			return &AgentOutput{
				Content: content,
				Data:    map[string]any{"violations": violations},
				Decisions: map[string]any{
					"violations":  violations,
					"canon_clean": len(violations) == 0,
					"count":       len(violations),
				},
				Status: status,
			}, nil
		},
	}
}

func runRuleChecks(input AgentInput) []map[string]any {
	var violations []map[string]any

	charLocs := map[string]string{}
	for _, st := range input.Ctx.CharStates {
		if st.Location != "" {
			if prev, ok := charLocs[st.CharacterID]; ok && prev != st.Location {
				violations = append(violations, map[string]any{
					"type":     "location_jump",
					"detail":   fmt.Sprintf("%s moved from %s to %s without transition", st.CharacterID, prev, st.Location),
					"severity": "warning",
				})
			}
			charLocs[st.CharacterID] = st.Location
		}
	}

	sceneChars := map[string]bool{}
	for _, pid := range input.Ctx.ParticipantIDs {
		sceneChars[pid] = true
	}
	for _, t := range input.Ctx.Turns {
		if t.Role == "character" {
			if _, ok := sceneChars[input.Ctx.Scene.ID]; !ok {
			}
		}
	}

	if len(input.Ctx.Timeline) > 1 {
		for i := 1; i < len(input.Ctx.Timeline); i++ {
			if input.Ctx.Timeline[i].Order < input.Ctx.Timeline[i-1].Order {
				violations = append(violations, map[string]any{
					"type":     "timeline_order",
					"detail":   fmt.Sprintf("timeline event %s (order %d) before %s (order %d)", input.Ctx.Timeline[i].SceneID, input.Ctx.Timeline[i].Order, input.Ctx.Timeline[i-1].SceneID, input.Ctx.Timeline[i-1].Order),
					"severity": "critical",
				})
			}
		}
	}

	for _, st := range input.Ctx.CharStates {
		if st.Health < 0 {
			violations = append(violations, map[string]any{
				"type":     "invalid_health",
				"detail":   fmt.Sprintf("%s has negative health (%d)", st.CharacterID, st.Health),
				"severity": "error",
			})
		}
	}

	for _, delta := range input.Ctx.CanonDeltas {
		if delta.Confidence < 0.3 {
			violations = append(violations, map[string]any{
				"type":     "low_confidence_delta",
				"fact":     delta.Fact,
				"severity": "warning",
			})
		}
	}

	return violations
}

func runLLMCanonCheck(ctx context.Context, llmClient llm.LLMClient, input AgentInput) ([]map[string]any, error) {
	scene := input.Ctx.Scene
	if scene == nil {
		return nil, nil
	}

	var b strings.Builder
	b.WriteString("Scene: " + scene.Title + "\n")
	b.WriteString("Tone: " + scene.Tone + " | POV: " + scene.POV + "\n\n")

	b.WriteString("Turns:\n")
	for _, t := range input.Ctx.Turns {
		b.WriteString(fmt.Sprintf("[%s]: %s\n\n", t.Role, t.Output))
	}

	userMsg := fmt.Sprintf(`Review the following scene for canon violations. Consider character voice consistency, location continuity, timeline logic, and any contradictions in the turns.

%s

Output JSON with format: {"violations": [{"type": "string", "detail": "string", "severity": "warning|error|critical"}]}
Return empty violations array if clean.`, b.String())

	resp, err := llmClient.Complete(ctx, llm.CompletionRequest{
		Model:        llm.ModelHaiku,
		System:       "You are a canon continuity checker. Output valid JSON only.",
		UserMessage:  userMsg,
		Temperature:  0.0,
		MaxTokens:    1024,
		ValidateJSON: true,
	})
	if err != nil {
		return nil, fmt.Errorf("canon llm check: %w", err)
	}

	var result struct {
		Violations []map[string]any `json:"violations"`
	}
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		return nil, fmt.Errorf("canon llm parse: %w", err)
	}

	return result.Violations, nil
}
