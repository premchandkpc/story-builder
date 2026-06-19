package domain

import (
	"fmt"
	"time"
)

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

var validStoryTransitions = map[string][]string{
	StoryStatusDraft:     {StoryStatusActive},
	StoryStatusActive:    {StoryStatusDraft, StoryStatusCompleted},
	StoryStatusCompleted: {StoryStatusArchived},
	StoryStatusArchived:  {},
}

func (s *Story) CanTransitionTo(target string) error {
	allowed, ok := validStoryTransitions[s.Status]
	if !ok {
		return fmt.Errorf("unknown story status: %s", s.Status)
	}
	if s.Status == target {
		return nil
	}
	for _, a := range allowed {
		if a == target {
			return nil
		}
	}
	return fmt.Errorf("cannot transition story from %s to %s", s.Status, target)
}
