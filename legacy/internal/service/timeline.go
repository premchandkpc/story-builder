package service

import (
	"context"
	"fmt"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type TimelineService struct {
	repo repository.TimelineRepository
}

func NewTimelineService(repo repository.TimelineRepository) *TimelineService {
	return &TimelineService{repo: repo}
}

func (s *TimelineService) Create(ctx context.Context, e *domain.TimelineEvent) (*domain.TimelineEvent, error) {
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, fmt.Errorf("create timeline event: %w", err)
	}
	return e, nil
}

func (s *TimelineService) List(ctx context.Context, storyID string) ([]*domain.TimelineEvent, error) {
	return s.repo.ListByStory(ctx, storyID)
}

func (s *TimelineService) CreateCrossStory(ctx context.Context, e *domain.TimelineEvent) (*domain.TimelineEvent, error) {
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, fmt.Errorf("create cross-story event: %w", err)
	}
	return e, nil
}

func (s *TimelineService) ListCrossStory(ctx context.Context, storyID string) ([]*domain.TimelineEvent, error) {
	return s.repo.ListByRelatedStories(ctx, storyID)
}
