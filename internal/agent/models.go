package agent

import (
	"time"

	"github.com/google/uuid"
)

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
	IntentAccuse    Intent = "accuse"
	IntentInvestigate Intent = "investigate"
)

type ActionType string

const (
	ActionSpeak    ActionType = "speak"
	ActionMove     ActionType = "move"
	ActionAttack   ActionType = "attack"
	ActionHide     ActionType = "hide"
	ActionUseItem  ActionType = "use_item"
	ActionInteract ActionType = "interact"
	ActionWait     ActionType = "wait"
)

type CharacterAgent struct {
	ID            uuid.UUID            `json:"id"`
	Name          string               `json:"name"`
	Description   string               `json:"description"`
	Archetype     string               `json:"archetype"`
	Personality   map[string]float64   `json:"personality"`
	Goals         []Goal               `json:"goals"`
	Beliefs       []Belief             `json:"beliefs"`
	Emotion       string               `json:"emotion"`
	Intensity     float64              `json:"intensity"`
	Stress        float64              `json:"stress"`
	Energy        float64              `json:"energy"`
	Location      string               `json:"location"`
	Traits        []string             `json:"traits"`
	VoiceSamples  []string             `json:"voice_samples"`
	Relationships map[string]float64   `json:"relationships"`
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

type AgentDecision struct {
	CharacterID uuid.UUID   `json:"character_id"`
	Intent      Intent      `json:"intent"`
	Action      ActionType  `json:"action"`
	ActionDesc  string      `json:"action_description"`
	TargetID    uuid.UUID   `json:"target_id,omitempty"`
	Emotion     string      `json:"emotion"`
	Dialogue    string      `json:"dialogue,omitempty"`
	Confidence  float64     `json:"confidence"`
}

type ActionScore struct {
	Intent        Intent   `json:"intent"`
	GoalAlignment float64  `json:"goal_alignment"`
	Risk          float64  `json:"risk"`
	Reward        float64  `json:"reward"`
	EmotionBias   float64  `json:"emotion_bias"`
	Total         float64  `json:"total"`
}

type DirectorNote struct {
	Intervention     string  `json:"intervention"`
	TensionAdjustment float64 `json:"tension_adjustment"`
	Pacing           string  `json:"pacing"`
	FocusCharacter   string  `json:"focus_character,omitempty"`
}

type DirectorDecision struct {
	SceneID     uuid.UUID    `json:"scene_id"`
	Note        DirectorNote `json:"note"`
	CreatedAt   time.Time    `json:"created_at"`
}

type ThinkingPipeline struct {
	AgentID     uuid.UUID   `json:"agent_id"`
	Step        string      `json:"step"`
	Memories    []string    `json:"memories,omitempty"`
	GoalEval    string      `json:"goal_evaluation,omitempty"`
	Intent      Intent      `json:"intent,omitempty"`
	Action      string      `json:"action,omitempty"`
	Dialogue    string      `json:"dialogue,omitempty"`
}

type AgentService interface {
	Think(agent *CharacterAgent, sceneContext map[string]any) (*AgentDecision, error)
	DecideIntent(agent *CharacterAgent, context map[string]any) (Intent, error)
	ScoreActions(agent *CharacterAgent, context map[string]any) ([]ActionScore, error)
	GenerateAction(agent *CharacterAgent, intent Intent) (ActionType, string, error)
	GenerateDialogue(agent *CharacterAgent, intent Intent, targetID uuid.UUID) (string, error)
}

type DirectorService interface {
	Direct(sceneID uuid.UUID, characters []CharacterAgent, context map[string]any) (*DirectorDecision, error)
	AdjustPacing(currentPacing string, tension float64) string
	SuggestIntervention(characters []CharacterAgent, context map[string]any) string
}
