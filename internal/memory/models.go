package memory

import (
	"time"

	"github.com/google/uuid"
)

type MemType string

const (
	TypeObservation   MemType = "observation"
	TypeConversation  MemType = "conversation"
	TypeRelationship  MemType = "relationship"
	TypeTrauma        MemType = "trauma"
	TypeAchievement   MemType = "achievement"
	TypeWorldKnowledge MemType = "world_knowledge"
	TypeSecret        MemType = "secret"
	TypeBelief        MemType = "belief"
)

type Memory struct {
	ID              uuid.UUID `json:"id"`
	StoryID         uuid.UUID `json:"story_id"`
	CharacterID     uuid.UUID `json:"character_id"`
	SceneID         uuid.UUID `json:"scene_id"`
	Type            MemType   `json:"type"`
	Summary         string    `json:"summary"`
	Importance      float64   `json:"importance"`
	EmotionalWeight float64   `json:"emotional_weight"`
	RetrievalScore  float64   `json:"retrieval_score"`
	Embedding       []float32 `json:"embedding,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type RetrievalQuery struct {
	StoryID     uuid.UUID `json:"story_id"`
	CharacterID uuid.UUID `json:"character_id,omitempty"`
	SceneID     uuid.UUID `json:"scene_id,omitempty"`
	Types       []MemType `json:"types,omitempty"`
	QueryText   string    `json:"query_text"`
	MaxResults  int       `json:"max_results"`
	MinScore    float64   `json:"min_score"`
}

type RankedMemory struct {
	Memory
	Score float64 `json:"score"`
}

func (m *Memory) Rank(querySimilarity, importanceWeight, recencyWeight, emotionalWeight float64, now time.Time) float64 {
	similarityScore := querySimilarity
	impScore := m.Importance * importanceWeight
	ageHours := now.Sub(m.CreatedAt).Hours()
	recencyScore := (1.0 / (1.0 + ageHours/24.0)) * recencyWeight
	emoScore := m.EmotionalWeight * emotionalWeight
	m.RetrievalScore = similarityScore + impScore + recencyScore + emoScore
	return m.RetrievalScore
}

type Store interface {
	Create(m *Memory) error
	Get(id uuid.UUID) (*Memory, error)
	Search(query RetrievalQuery) ([]RankedMemory, error)
	RetrieveRecent(storyID, characterID uuid.UUID, limit int) ([]Memory, error)
	RetrieveByType(storyID uuid.UUID, typ MemType, limit int) ([]Memory, error)
	Delete(id uuid.UUID) error
}
