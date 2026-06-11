package api

import (
	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/timeline"
)

type TimelineService interface {
	Save(storyID uuid.UUID, event *timeline.Event) error
	List(storyID uuid.UUID) ([]timeline.Event, error)
}

type inMemoryTimelineService struct {
	store *timeline.MemoryStore
}

func NewInMemoryTimelineService() *inMemoryTimelineService {
	return &inMemoryTimelineService{store: timeline.NewMemoryStore()}
}

func (s *inMemoryTimelineService) Save(storyID uuid.UUID, event *timeline.Event) error {
	if s.store == nil {
		return nil
	}
	return s.store.Save(storyID, event)
}

func (s *inMemoryTimelineService) List(storyID uuid.UUID) ([]timeline.Event, error) {
	if s.store == nil {
		return nil, nil
	}
	return s.store.List(storyID)
}
