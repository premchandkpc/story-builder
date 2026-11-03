package domain

import "time"

type CharacterMemory struct {
	ID          string    `bson:"_id" json:"id"`
	StoryID     string    `bson:"storyId" json:"storyId"`
	CharacterID string    `bson:"characterId" json:"characterId"`
	SceneID     string    `bson:"sceneId,omitempty" json:"sceneId,omitempty"`
	Content     string    `bson:"content" json:"content"`
	Type        string    `bson:"type,omitempty" json:"type,omitempty"`
	Importance  float64   `bson:"importance" json:"importance"`
	Embedding   []float64 `bson:"embedding,omitempty" json:"embedding,omitempty"`
	CreatedAt   time.Time `bson:"createdAt" json:"createdAt"`
}

const (
	MemoryTypeEvent            = "event"
	MemoryTypeDialogue         = "dialogue"
	MemoryTypeObservation      = "observation"
	MemoryTypeInjury           = "injury"
	MemoryTypeRelationshipChange = "relationship_change"
)
