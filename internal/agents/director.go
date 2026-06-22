package agents

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/trace"
)

func NewDirectorSpec(llmClient llm.LLMClient, proseSvc llm.ProseService) AgentSpec {
	return AgentSpec{
		Name:     domain.AgentTypeDirector,
		Role:     "scene_director",
		Model:    string(llm.ModelSonnet),
		MaxTurns: 1,
		SystemPrompt: `You are the Director agent for a narrative engine.

Your job is to plan each scene turn:
1. Read the scene goal, active characters, current state, and unresolved conflicts
2. Decide who acts next and what pressure/conflict to escalate
3. Signal when the scene should end

Output valid JSON only with these fields:
{
  "who_acts": ["character_id_1", "character_id_2"],
  "pressure": 0.5,
  "escalation": "description of conflict to escalate",
  "end_scene": false,
  "reasoning": "brief explanation of the plan"
}`,
		Runner: func(ctx context.Context, input AgentInput) (*AgentOutput, error) {
			ctx, span := trace.StartSpan(ctx, "agent.director."+input.Directive)
			defer trace.End(span)

			scene := input.Ctx.Scene
			if scene == nil {
				err := fmt.Errorf("scene required for director")
				trace.SetError(span, err)
				return nil, err
			}

			trace.SetAttribute(span, "sceneId", scene.ID)
			trace.SetAttribute(span, "flowType", scene.FlowType)
			trace.SetAttribute(span, "turnCount", len(input.Ctx.Turns))

			participants := scene.Participants
			if len(participants) == 0 {
				participants = input.Ctx.ParticipantIDs
			}

			chars := ""
			for _, c := range input.Ctx.Characters {
				chars += fmt.Sprintf("- %s (%s): %s\n", c.Name, c.CharID, c.Persona)
			}
			states := ""
			for _, st := range input.Ctx.CharStates {
				states += fmt.Sprintf("- %s in scene %s: mood=%s, location=%s\n", st.CharacterID, st.SceneID, st.Mood, st.Location)
			}
			prevTurns := ""
			for _, t := range input.Ctx.Turns {
				prevTurns += fmt.Sprintf("Turn %d [%s]: %s\n", t.Number, t.Role, t.Output)
			}

			proposalsBlock := ""
			if props, ok := input.Payload["proposals"]; ok {
				if propsList, ok := props.([]CharacterProposal); ok && len(propsList) > 0 {
					proposalsBlock = "\nCharacter Proposals:\n"
					for _, p := range propsList {
						proposalsBlock += fmt.Sprintf("- %s wants to: %s\n", p.CharacterID, p.Content)
					}
					proposalsBlock += "\n"
				}
			}

			userMsg := fmt.Sprintf(`Scene: %s
Beat Intent: %s
POV: %s
Tone: %s
Flow Type: %s
Max Turns: %d
Participants: %v
%s
Characters:
%s

Character States:
%s

Previous Turns:
%s

Produce a JSON turn plan.`, scene.Title, scene.BeatIntent, scene.POV, scene.Tone, scene.FlowType, scene.MaxTurns, participants, proposalsBlock, chars, states, prevTurns)

			resp, err := llmClient.Complete(ctx, llm.CompletionRequest{
				Model:        llm.ModelSonnet,
				System:       `You are the Director agent. Output ONLY valid JSON.`,
				UserMessage:  userMsg,
				Temperature:  0.3,
				MaxTokens:    1024,
				ValidateJSON: true,
			})
			if err != nil {
				trace.SetError(span, err)
				return nil, fmt.Errorf("director llm call: %w", err)
			}

			var decisions map[string]any
			if err := json.Unmarshal([]byte(resp.Content), &decisions); err != nil {
				decisions = map[string]any{
					"beat_intent":  scene.BeatIntent,
					"participants": participants,
					"pov":          scene.POV,
					"tone":         scene.Tone,
					"pressure":     0.5,
					"escalation":   scene.BeatIntent,
					"end_scene":    false,
				}
			}

			return &AgentOutput{
				Content:   resp.Content,
				Data:      map[string]any{"scene": scene},
				Decisions: decisions,
				Status:    "success",
			}, nil
		},
	}
}
