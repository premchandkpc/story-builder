package domain

import "time"

type CharacterView struct {
	CharacterID  string                 `bson:"_id" json:"character_id"`
	StoryID      string                 `bson:"storyId" json:"story_id"`
	CurrentState CharacterStateSnapshot `bson:"currentState" json:"current_state"`
	EventIDs     []string               `bson:"eventIds" json:"event_ids"`
	Version      int64                  `bson:"version" json:"version"`
	UpdatedAt    time.Time              `bson:"updatedAt" json:"updated_at"`
}

type CharacterStateSnapshot struct {
	Location       string        `bson:"location,omitempty" json:"location,omitempty"`
	Health         int           `bson:"health,omitempty" json:"health,omitempty"`
	EmotionalState string        `bson:"emotionalState,omitempty" json:"emotional_state,omitempty"`
	Mood           string        `bson:"mood,omitempty" json:"mood,omitempty"`
	ActiveGoal     string        `bson:"activeGoal,omitempty" json:"active_goal,omitempty"`
	Knowledge      []string      `bson:"knowledge,omitempty" json:"knowledge,omitempty"`
	Relationships  []RelSnapshot `bson:"relationships,omitempty" json:"relationships,omitempty"`
}

type RelSnapshot struct {
	TargetID  string  `bson:"targetId" json:"target_id"`
	Trust     float64 `bson:"trust" json:"trust"`
	Respect   float64 `bson:"respect" json:"respect"`
	Fear      float64 `bson:"fear" json:"fear"`
	Affection float64 `bson:"affection" json:"affection"`
}
