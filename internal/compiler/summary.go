package compiler

import (
	"context"

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
	UpsertSceneSummary(ctx context.Context, storyID, nodeID uuid.UUID, content string) error
	UpsertActSummary(ctx context.Context, storyID uuid.UUID, content string) error
	UpsertStorySummary(ctx context.Context, storyID uuid.UUID, content string) error
	GetSceneSummary(ctx context.Context, storyID, nodeID uuid.UUID) (*StorySummary, error)
	GetSummaryByLevel(ctx context.Context, storyID uuid.UUID, level SummaryLevel) (*StorySummary, error)
	ListSummariesByLevel(ctx context.Context, storyID uuid.UUID, level SummaryLevel) ([]StorySummary, error)
	CountSummariesByLevel(ctx context.Context, storyID uuid.UUID, level SummaryLevel) (int, error)
	ShouldElevate(ctx context.Context, storyID uuid.UUID, level SummaryLevel, threshold int) (bool, error)
}
