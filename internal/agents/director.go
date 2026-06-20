package agents

import (
	"context"
	"fmt"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
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

Output a turn plan with:
- who_acts: character ID(s) to act next
- pressure: current scene pressure level (0.0-1.0)
- escalation: what conflict to escalate
- end_scene: boolean, whether the scene should end after this turn`,
		Runner: func(ctx context.Context, input AgentInput) (*AgentOutput, error) {
			scene := input.Ctx.Scene
			if scene == nil {
				return nil, fmt.Errorf("scene required for director")
			}

			decisions := map[string]any{
				"beat_intent":    scene.BeatIntent,
				"participants":   scene.Participants,
				"pov":            scene.POV,
				"tone":           scene.Tone,
				"pressure":       0.5,
				"escalation":     scene.BeatIntent,
				"end_scene":      false,
			}

			return &AgentOutput{
				Content:   fmt.Sprintf("Director plan for scene %s: beat=%s, participants=%v", scene.ID, scene.BeatIntent, scene.Participants),
				Data:      map[string]any{"scene": scene},
				Decisions: decisions,
				Status:    "success",
			}, nil
		},
	}
}
