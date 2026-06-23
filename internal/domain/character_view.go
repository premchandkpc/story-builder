package domain

import "time"

type CharacterView struct {
	CharacterID  string                 `bson:"_id"`
	StoryID      string                 `bson:"storyId"`
	CurrentState CharacterStateSnapshot `bson:"currentState"`
	EventIDs     []string               `bson:"eventIds"`
	Version      int64                  `bson:"version"`
	UpdatedAt    time.Time              `bson:"updatedAt"`
}

type CharacterStateSnapshot struct {
	Location       string        `bson:"location,omitempty"`
	Health         int           `bson:"health,omitempty"`
	EmotionalState string        `bson:"emotionalState,omitempty"`
	Mood           string        `bson:"mood,omitempty"`
	ActiveGoal     string        `bson:"activeGoal,omitempty"`
	Knowledge      []string      `bson:"knowledge,omitempty"`
	Relationships  []RelSnapshot `bson:"relationships,omitempty"`
}

type RelSnapshot struct {
	TargetID  string  `bson:"targetId"`
	Trust     float64 `bson:"trust"`
	Respect   float64 `bson:"respect"`
	Fear      float64 `bson:"fear"`
	Affection float64 `bson:"affection"`
}
