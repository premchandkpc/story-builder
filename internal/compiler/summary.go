package compiler

import (
	"github.com/google/uuid"
)

type SummaryLevel string

const (
	SummaryScene SummaryLevel = "scene"
	SummaryAct   SummaryLevel = "act"
	SummaryStory SummaryLevel = "story"
)

type StorySummary struct {
	ID        uuid.UUID    `json:"id"`
	StoryID   uuid.UUID    `json:"story_id"`
	NodeID    *uuid.UUID   `json:"node_id,omitempty"`
	Level     SummaryLevel `json:"level"`
	Content   string       `json:"content"`
	WordCount int          `json:"word_count"`
	CreatedAt string       `json:"created_at"`
}

type SummaryService interface {
	UpsertSceneSummary(storyID, nodeID uuid.UUID, content string) error
	UpsertActSummary(storyID uuid.UUID, content string) error
	UpsertStorySummary(storyID uuid.UUID, content string) error
	GetSceneSummary(storyID, nodeID uuid.UUID) (*StorySummary, error)
	GetSummaryByLevel(storyID uuid.UUID, level SummaryLevel) (*StorySummary, error)
	ListSummariesByLevel(storyID uuid.UUID, level SummaryLevel) ([]StorySummary, error)
	CountSummariesByLevel(storyID uuid.UUID, level SummaryLevel) (int, error)
	ShouldElevate(storyID uuid.UUID, level SummaryLevel, threshold int) (bool, error)
}
