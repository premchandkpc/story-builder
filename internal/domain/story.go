package domain

import "time"

type Story struct {
	ID            string                 `bson:"_id" json:"id"`
	Title         string                 `bson:"title" json:"title"`
	Genre         string                 `bson:"genre,omitempty" json:"genre,omitempty"`
	Theme         string                 `bson:"theme,omitempty" json:"theme,omitempty"`
	MainPrompt    string                 `bson:"mainPrompt,omitempty" json:"mainPrompt,omitempty"`
	GeneralPrompt string                 `bson:"generalPrompt,omitempty" json:"generalPrompt,omitempty"`
	CanonPins     map[string]any         `bson:"canonPins,omitempty" json:"canonPins,omitempty"`
	RootSceneID   string                 `bson:"rootSceneId,omitempty" json:"rootSceneId,omitempty"`
	Status        string                 `bson:"status" json:"status"`
	Blueprint     *StoryBlueprint        `bson:"blueprint,omitempty" json:"blueprint,omitempty"`
	CreatedAt     time.Time              `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time              `bson:"updatedAt" json:"updatedAt"`
}

const (
	StoryStatusDraft     = "draft"
	StoryStatusActive    = "active"
	StoryStatusCompleted = "completed"
	StoryStatusArchived  = "archived"
)
