package agents

import (
	"context"
	"fmt"
	"strings"

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
2. Stitch character outputs into a cohesive narrative flow
3. Add pacing, transition, and atmosphere
4. Maintain consistent POV — third-person limited unless POV specifies otherwise
5. Do NOT invent new character actions or dialogue — only narrate what the Character agents produced
6. Keep the prose tight and vivid
7. Do not include meta-commentary, labels, or formatting notes

Output only narrative prose — no JSON, no annotations.`,
		Runner: func(ctx context.Context, input AgentInput) (*AgentOutput, error) {
			scene := input.Ctx.Scene

			userMsg := buildNarratorPrompt(input, scene)

			resp, err := llmClient.Complete(ctx, llm.CompletionRequest{
				Model:       llm.ModelSonnet,
				System:      "You are a narrative prose writer. Output only story prose, no meta-text.",
				UserMessage: userMsg,
				Temperature: 0.5,
				MaxTokens:   2048,
			})
			if err != nil {
				return nil, fmt.Errorf("narrator agent llm call: %w", err)
			}

			return &AgentOutput{
				Content: resp.Content,
				Data:    map[string]any{"sceneId": scene.ID},
				Decisions: map[string]any{
					"tone": scene.Tone,
					"pov":  scene.POV,
					"pace": "normal",
				},
				Status: "success",
			}, nil
		},
	}
}

func buildNarratorPrompt(input AgentInput, scene *domain.Scene) string {
	var b strings.Builder

	b.WriteString("=== SCENE CONTEXT ===\n")
	b.WriteString(fmt.Sprintf("Title: %s\n", scene.Title))
	if scene.BeatIntent != "" {
		b.WriteString(fmt.Sprintf("Beat Intent: %s\n", scene.BeatIntent))
	}
	b.WriteString(fmt.Sprintf("Tone: %s | POV: %s | Flow: %s\n", scene.Tone, scene.POV, scene.FlowType))

	b.WriteString("\n=== LOCATION ===\n")
	if scene.LocationRef != "" {
		b.WriteString(fmt.Sprintf("Setting: %s\n", scene.LocationRef))
		for _, loc := range input.Ctx.CharStates {
			if loc.Location != "" {
				b.WriteString(fmt.Sprintf("Active location: %s\n", loc.Location))
				break
			}
		}
	}

	b.WriteString("\n=== CHARACTER TURNS TO NARRATE ===\n")
	for _, t := range input.Ctx.Turns {
		b.WriteString(fmt.Sprintf("[%s]: %s\n", t.Role, t.Output))
	}

	b.WriteString("\n=== YOUR TASK ===\n")
	b.WriteString("Weave the above character turns into narrative prose. ")
	b.WriteString("Maintain the tone and POV specified. ")
	b.WriteString("Use the turns in order, adding narrative framing, sensory detail, and pacing. ")
	b.WriteString("Do not add new character actions or dialogue beyond what is shown.\n")

	return b.String()
}
