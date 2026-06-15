package relationship

import (
	"time"

	"github.com/google/uuid"
)

type Relationship struct {
	StoryID    uuid.UUID `json:"story_id"`
	CharA      uuid.UUID `json:"char_a"`
	CharB      uuid.UUID `json:"char_b"`
	Trust      float64   `json:"trust"`
	Respect    float64   `json:"respect"`
	Fear       float64   `json:"fear"`
	Affection  float64   `json:"affection"`
	Dependency float64   `json:"dependency"`
	Rivalry    float64   `json:"rivalry"`
	Loyalty    float64   `json:"loyalty"`
	Suspicion  float64   `json:"suspicion"`
	History    []Delta   `json:"history"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Delta struct {
	Timestamp  time.Time `json:"timestamp"`
	SceneID    uuid.UUID `json:"scene_id"`
	Trust      float64   `json:"trust_delta"`
	Respect    float64   `json:"respect_delta"`
	Fear       float64   `json:"fear_delta"`
	Affection  float64   `json:"affection_delta"`
	Reason     string    `json:"reason"`
}

type Store interface {
	Get(storyID, charA, charB uuid.UUID) (*Relationship, error)
	Upsert(rel *Relationship) error
	GetAllForCharacter(storyID, charID uuid.UUID) ([]Relationship, error)
	GetHistory(storyID, charA, charB uuid.UUID) ([]Delta, error)
}
