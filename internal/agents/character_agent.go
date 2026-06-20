package agents

import (
	"context"
	"fmt"

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
5. Do NOT narrate outside your character's perspective

Your context includes:
- character card (personality, backstory, goals, flaws)
- current emotional/ physical state
- what you know and don't know
- relationship state with other characters present
- recent memories relevant to this scene`,
		Runner: func(ctx context.Context, input AgentInput) (*AgentOutput, error) {
			charID := ""
			if id, ok := input.Payload["characterId"].(string); ok {
				charID = id
			}
			if charID == "" && len(input.Ctx.ParticipantIDs) > 0 {
				charID = input.Ctx.ParticipantIDs[0]
			}
			if charID == "" {
				return nil, fmt.Errorf("no character selected for character agent")
			}

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

			response := fmt.Sprintf("%s reacts to the scene.", charID)
			decisions := map[string]any{
				"character_id": charID,
				"emotion":      "neutral",
				"action_type":  "dialogue",
			}
			if character != nil {
				goals := "unknown"
				if len(character.Goals) > 0 {
					goals = character.Goals[0]
				}
				response = fmt.Sprintf("%s: *acting on %s*", character.Name, goals)
			}
			if state != nil {
				decisions["emotion"] = state.EmotionalState
			}

			return &AgentOutput{
				Content:   response,
				Data:      map[string]any{"character": character},
				Decisions: decisions,
				Status:    "success",
			}, nil
		},
	}
}
