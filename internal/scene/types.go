package scene

import (
	"time"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/graph"
)

type SceneTurn struct {
	ID         uuid.UUID   `json:"id"`
	NodeID     uuid.UUID   `json:"node_id"`
	TurnNumber int         `json:"turn_number"`
	ActorIDs   []uuid.UUID `json:"actor_ids"`
	Prompt     string      `json:"prompt"`
	Output     string      `json:"output"`
	Model      string      `json:"model"`
	Status     string      `json:"status"`
	CreatedAt  time.Time   `json:"created_at"`
}

type SceneService interface {
	StartScene(nodeID uuid.UUID) (*SceneTurn, error)
	NextTurn(nodeID uuid.UUID) (*SceneTurn, error)
	FinishScene(nodeID uuid.UUID) (string, error)
	GetTurns(nodeID uuid.UUID) ([]SceneTurn, error)
	SetSceneStructure(nodeID uuid.UUID, ss graph.SceneStructure) error
	GetSceneStructure(nodeID uuid.UUID) (*graph.SceneStructure, error)
}

type AgentPromptInput struct {
	CharacterName string
	Traits        []string
	VoiceSamples  []string
	Mood          string
	Location      string
	Knows         []string
	DoesNotKnow   []string
	Relationships map[string]string
	SituationFlow string
	BeatIntent    string
	SceneSetting  string
	PreviousTurns string
}
