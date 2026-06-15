package scene

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Agent struct {
	CharacterName  string            `json:"character_name"`
	Description    string            `json:"description"`
	Archetype      string            `json:"archetype"`
	Personality    map[string]any    `json:"personality"`
	Goals          []Goal            `json:"goals"`
	Beliefs        []Belief          `json:"beliefs"`
	Emotion        string            `json:"emotion"`
	Intensity      float64           `json:"intensity"`
	Stress         float64           `json:"stress"`
	Energy         float64           `json:"energy"`
	CurrentGoal    string            `json:"current_goal"`
	Traits         []string          `json:"traits"`
	VoiceSamples   []string          `json:"voice_samples"`
}

type Goal struct {
	ID          string  `json:"id"`
	Description string  `json:"description"`
	Priority    float64 `json:"priority"`
	Status      string  `json:"status"`
	Type        string  `json:"type"`
}

type Belief struct {
	Statement  string  `json:"statement"`
	Confidence float64 `json:"confidence"`
}

type Intent string

const (
	IntentThreaten  Intent = "threaten"
	IntentFlirt     Intent = "flirt"
	IntentLie       Intent = "lie"
	IntentPersuade  Intent = "persuade"
	IntentAttack    Intent = "attack"
	IntentHide      Intent = "hide"
	IntentReveal    Intent = "reveal"
	IntentDefend    Intent = "defend"
	IntentQuestion  Intent = "question"
	IntentSupport   Intent = "support"
	IntentBetray    Intent = "betray"
	IntentNegotiate Intent = "negotiate"
)

type AgentAction struct {
	Action  string `json:"action"`
	Emotion string `json:"emotion"`
	Target  string `json:"target,omitempty"`
	Intent  Intent `json:"intent"`
}

type AgentOutput struct {
	Intent   Intent      `json:"intent"`
	Emotion  string      `json:"emotion"`
	Action   *AgentAction `json:"action,omitempty"`
	Dialogue string      `json:"dialogue,omitempty"`
}

type ThinkingStep struct {
	CharacterName string   `json:"character_name"`
	Memories      []string `json:"memories"`
	GoalEval      string   `json:"goal_evaluation"`
	Intent        Intent   `json:"intent"`
	Action        string   `json:"action"`
	Dialogue      string   `json:"dialogue"`
}

func BuildAgentSystemPrompt(agent Agent, location string, knownFacts, unknownFacts []string, sceneGoal string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`You are %s.

Description: %s
Archetype: %s
`, agent.CharacterName, agent.Description, agent.Archetype))

	if len(agent.Goals) > 0 {
		b.WriteString("\nYour goals (in priority order):\n")
		for _, g := range agent.Goals {
			b.WriteString(fmt.Sprintf("- [%s] %.0f: %s\n", g.Status, g.Priority, g.Description))
		}
	}

	if len(agent.Beliefs) > 0 {
		b.WriteString("\nWhat you believe:\n")
		for _, bl := range agent.Beliefs {
			b.WriteString(fmt.Sprintf("- %s (confidence: %.0f%%)\n", bl.Statement, bl.Confidence))
		}
	}

	b.WriteString(fmt.Sprintf(`
Current emotion: %s (%.0f%% intensity)
Stress: %.0f  Energy: %.0f

Current location: %s
Scene goal: %s

What you know:
- %s

What you don't know:
- %s

IMPORTANT: Before you speak, decide your INTENT first. 
Your intent should be one of: threaten, flirt, lie, persuade, attack, hide, reveal, defend, question, support, betray, negotiate.

Then decide your ACTION.

Then generate dialogue.

Output format:
{
  "thinking": "...",
  "intent": "...",
  "action": "...",
  "dialogue": "..."
}`,
		agent.Emotion, agent.Intensity,
		agent.Stress, agent.Energy,
		location, sceneGoal,
		strings.Join(knownFacts, "\n- "),
		strings.Join(unknownFacts, "\n- "),
	))

	return b.String()
}

func BuildNarrativeDirectorPrompt(sceneGoal, conflict string, chars []string, pacing string) string {
	return fmt.Sprintf(`You are the Narrative Director.

Scene goal: %s
Conflict: %s
Characters present: %s

Current pacing: %s

Decide if the scene needs:
- A twist
- A reveal
- A cliffhanger
- Increased tension
- A new clue
- An interruption

Output:
{
  "director_note": "...",
  "intervention": "...",
  "tension_adjustment": 0
}`,
		sceneGoal, conflict, strings.Join(chars, ", "), pacing,
	)
}

func ParseAgentOutput(raw string) (*AgentOutput, error) {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var out AgentOutput
	if err := json.Unmarshal([]byte(cleaned), &out); err != nil {
		return nil, fmt.Errorf("parse agent output: %w", err)
	}
	if out.Intent == "" {
		out.Intent = IntentQuestion
	}
	if out.Action == nil {
		out.Action = &AgentAction{Action: "speaks", Emotion: "neutral", Intent: out.Intent}
	}
	return &out, nil
}
