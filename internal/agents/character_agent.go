package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
)

func NewCharacterSpec(llmClient llm.LLMClient, proseSvc llm.ProseService) AgentSpec {
	return AgentSpec{
		Name:     domain.AgentTypeCharacter,
		Role:     "character",
		Model:    string(llm.ModelSonnet),
		MaxTurns: 5,
		SystemPrompt: `You are a Character agent in a narrative engine.

You role-play ONE character in the current scene. Rules:
1. Stay in-character: voice, goals, beliefs, emotional state, secrets
2. Act on what your character knows — not what you as an author know
3. React to other characters' actions and scene pressure
4. Output dialogue, action, or internal response
5. Narrate only from your character's perspective — do not describe other characters' internal states

Your output should be first-person or third-person limited (as appropriate for the scene POV), showing what your character says and does. Use dialogue for speech, *asterisks for actions*, and (parentheticals for internal thoughts).

Your context includes:
- character card (personality, backstory, goals, flaws, voice)
- current emotional/physical state
- what you know and don't know
- relationship state with other characters present
- recent memories relevant to this scene`,
		Runner: func(ctx context.Context, input AgentInput) (*AgentOutput, error) {
			charID := resolveCharID(input)

			var character *domain.Character
			for _, c := range input.Ctx.Characters {
				if c.CharID == charID || c.ID == charID {
					character = c
					break
				}
			}

			var state *domain.CharacterState
			for _, s := range input.Ctx.CharStates {
				if s.CharacterID == charID {
					state = s
					break
				}
			}

			if character == nil {
				return nil, fmt.Errorf("character %s not found in context", charID)
			}

			userMsg := buildCharacterPrompt(input, character, state, charID)

			resp, err := llmClient.Complete(ctx, llm.CompletionRequest{
				Model:       llm.ModelSonnet,
				System:      fmt.Sprintf("You are %s. Respond in-character as %s.", character.Name, character.Name),
				UserMessage: userMsg,
				Temperature: 0.8,
				MaxTokens:   2048,
			})
			if err != nil {
				return nil, fmt.Errorf("character agent llm call: %w", err)
			}

			emotion := "unknown"
			if state != nil && state.EmotionalState != "" {
				emotion = state.EmotionalState
			} else if state != nil && state.Mood != "" {
				emotion = state.Mood
			}

			return &AgentOutput{
				Content: resp.Content,
				Data:    map[string]any{"character": character},
				Decisions: map[string]any{
					"character_id": charID,
					"emotion":      emotion,
					"action_type":  input.Directive,
				},
				Status: "success",
			}, nil
		},
	}
}

func resolveCharID(input AgentInput) string {
	if id, ok := input.Payload["characterId"].(string); ok && id != "" {
		return id
	}

	participants := input.Ctx.ParticipantIDs
	if len(participants) == 0 {
		return ""
	}
	if len(participants) == 1 {
		return participants[0]
	}

	charTurnCount := 0
	for _, t := range input.Ctx.Turns {
		if t.Role == "character" {
			charTurnCount++
		}
	}

	return participants[charTurnCount%len(participants)]
}

func buildCharacterPrompt(input AgentInput, character *domain.Character, state *domain.CharacterState, charID string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Scene: %s\n", input.Ctx.Scene.Title))
	if input.Ctx.Scene.BeatIntent != "" {
		b.WriteString(fmt.Sprintf("Beat Intent: %s\n", input.Ctx.Scene.BeatIntent))
	}
	b.WriteString(fmt.Sprintf("Tone: %s | POV: %s | Pressure: escalating\n\n", input.Ctx.Scene.Tone, input.Ctx.Scene.POV))

	b.WriteString(fmt.Sprintf("=== CHARACTER: %s ===\n", character.Name))
	b.WriteString(fmt.Sprintf("Persona: %s\n", character.Persona))
	if len(character.Traits) > 0 {
		b.WriteString(fmt.Sprintf("Traits: %s\n", strings.Join(character.Traits, ", ")))
	}
	if len(character.Goals) > 0 {
		b.WriteString(fmt.Sprintf("Goals: %s\n", strings.Join(character.Goals, "; ")))
	}
	if len(character.Flaws) > 0 {
		b.WriteString(fmt.Sprintf("Flaws: %s\n", strings.Join(character.Flaws, "; ")))
	}
	if character.Fear != "" {
		b.WriteString(fmt.Sprintf("Fear: %s\n", character.Fear))
	}
	if character.FalseBelief != "" {
		b.WriteString(fmt.Sprintf("False Belief: %s\n", character.FalseBelief))
	}
	if len(character.VoiceSamples) > 0 {
		b.WriteString(fmt.Sprintf("Voice Style: %s\n", strings.Join(character.VoiceSamples, " | ")))
	}

	if state != nil {
		b.WriteString("\n=== CURRENT STATE ===\n")
		if state.EmotionalState != "" {
			b.WriteString(fmt.Sprintf("Emotion: %s\n", state.EmotionalState))
		}
		if state.Mood != "" {
			b.WriteString(fmt.Sprintf("Mood: %s\n", state.Mood))
		}
		if state.PhysicalState != "" {
			b.WriteString(fmt.Sprintf("Physical: %s\n", state.PhysicalState))
		}
		if state.Location != "" {
			b.WriteString(fmt.Sprintf("Location: %s\n", state.Location))
		}
		if state.ActiveGoal != "" {
			b.WriteString(fmt.Sprintf("Active Goal: %s\n", state.ActiveGoal))
		}
		if len(state.Knowledge) > 0 {
			b.WriteString(fmt.Sprintf("Knows: %s\n", strings.Join(state.Knowledge, "; ")))
		}
		if len(state.DoesNotKnow) > 0 {
			b.WriteString(fmt.Sprintf("Does NOT Know: %s\n", strings.Join(state.DoesNotKnow, "; ")))
		}
		if state.Health > 0 {
			b.WriteString(fmt.Sprintf("Health: %d\n", state.Health))
		}
	}

	if mems, ok := input.Ctx.Memories[charID]; ok && len(mems) > 0 {
		b.WriteString("\n=== RELEVANT MEMORIES ===\n")
		maxMem := 5
		if len(mems) < maxMem {
			maxMem = len(mems)
		}
		for _, m := range mems[:maxMem] {
			b.WriteString(fmt.Sprintf("- [%s] %s\n", m.Type, m.Content))
		}
	}

	b.WriteString("\n=== RECENT TURNS ===\n")
	for _, t := range input.Ctx.Turns {
		b.WriteString(fmt.Sprintf("[%s]: %s\n", t.Role, t.Output))
	}

	b.WriteString(fmt.Sprintf("\n=== YOUR TASK ===\n"))
	b.WriteString(fmt.Sprintf("Phase: %s\n", input.Directive))
	b.WriteString("Produce dialogue, action, or internal response for this character. Write in-character only.\n")

	return b.String()
}
