package timeline

import (
	"context"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/timeline"
)

type TimelineService interface {
	Save(ctx context.Context, storyID uuid.UUID, event *timeline.Event) error
	List(ctx context.Context, storyID uuid.UUID) ([]timeline.Event, error)
}

type MemoryTimelineService struct {
	store *timeline.MemoryStore
}

func NewMemoryService() *MemoryTimelineService {
	return &MemoryTimelineService{store: timeline.NewMemoryStore()}
}

func (s *MemoryTimelineService) Save(ctx context.Context, storyID uuid.UUID, event *timeline.Event) error {
	return s.store.Save(storyID, event)
}

func (s *MemoryTimelineService) List(ctx context.Context, storyID uuid.UUID) ([]timeline.Event, error) {
	return s.store.List(storyID)
}
