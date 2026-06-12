package api

import (
	"context"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/timeline"
)

type TimelineService interface {
	Save(ctx context.Context, storyID uuid.UUID, event *timeline.Event) error
	List(ctx context.Context, storyID uuid.UUID) ([]timeline.Event, error)
}

type inMemoryTimelineService struct {
	store *timeline.MemoryStore
}

func NewInMemoryTimelineService() *inMemoryTimelineService {
	return &inMemoryTimelineService{store: timeline.NewMemoryStore()}
}

func (s *inMemoryTimelineService) Save(ctx context.Context, storyID uuid.UUID, event *timeline.Event) error {
	if s.store == nil {
		return nil
	}
	return s.store.Save(storyID, event)
}

func (s *inMemoryTimelineService) List(ctx context.Context, storyID uuid.UUID) ([]timeline.Event, error) {
	if s.store == nil {
		return nil, nil
	}
	return s.store.List(storyID)
}
