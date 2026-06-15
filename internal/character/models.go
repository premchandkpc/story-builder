package character

import (
	"time"

	"github.com/google/uuid"
)

type Definition struct {
	ID              uuid.UUID         `json:"id"`
	Version         int               `json:"version"`
	Name            string            `json:"name"`
	Persona         string            `json:"persona,omitempty"`
	Backstory       string            `json:"backstory,omitempty"`
	MoralAlignment string            `json:"moral_alignment,omitempty"`
	Personality     []string          `json:"personality,omitempty"`
	Flaws           []string          `json:"flaws,omitempty"`
	Goals           []string          `json:"goals,omitempty"`
	Traits          []string          `json:"traits"`
	VoiceSamples    []string          `json:"voice_samples"`
	ParentID        *uuid.UUID        `json:"parent_id,omitempty"`
	Relationships   map[string]string `json:"relationships"`
	CreatedAt       time.Time         `json:"created_at"`
}

type State struct {
	StoryID       uuid.UUID         `json:"story_id"`
	CharacterID   uuid.UUID         `json:"character_id"`
	AsOfScene     uuid.UUID         `json:"as_of_scene"`
	Location      string            `json:"location"`
	Knows         []string          `json:"knows"`
	DoesNotKnow   []string          `json:"does_not_know"`
	Mood          string            `json:"mood"`
	Relationships map[string]string `json:"relationships"`
	Items         []string          `json:"items"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

type MemoryType string

const (
	MemObservation    MemoryType = "observation"
	MemConversation   MemoryType = "conversation"
	MemRelationship   MemoryType = "relationship"
	MemTrauma         MemoryType = "trauma"
	MemAchievement    MemoryType = "achievement"
	MemWorldKnowledge MemoryType = "world_knowledge"
	MemSecret         MemoryType = "secret"
	MemBelief         MemoryType = "belief"
)

type Memory struct {
	ID              uuid.UUID `json:"id"`
	StoryID         uuid.UUID `json:"story_id"`
	CharacterID     uuid.UUID `json:"character_id"`
	SceneID         uuid.UUID `json:"scene_id"`
	Type            MemoryType `json:"type"`
	Summary         string    `json:"summary"`
	Importance      float64   `json:"importance"`
	EmotionalWeight float64   `json:"emotional_weight"`
	Embedding       []float32 `json:"embedding,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type RetrievalQuery struct {
	StoryID     uuid.UUID   `json:"story_id"`
	CharacterID uuid.UUID   `json:"character_id,omitempty"`
	SceneID     uuid.UUID   `json:"scene_id,omitempty"`
	Types       []MemoryType `json:"types,omitempty"`
	QueryText   string      `json:"query_text"`
	MaxResults  int         `json:"max_results"`
	MinScore    float64     `json:"min_score"`
}

type RankedMemory struct {
	Memory
	Score float64 `json:"score"`
}
