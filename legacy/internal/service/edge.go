package service

import (
	"context"
	"fmt"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type EdgeService struct {
	repo repository.SceneEdgeRepository
}

func NewEdgeService(repo repository.SceneEdgeRepository) *EdgeService {
	return &EdgeService{repo: repo}
}

func (s *EdgeService) Create(ctx context.Context, e *domain.SceneEdge) (*domain.SceneEdge, error) {
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, fmt.Errorf("create edge: %w", err)
	}
	return e, nil
}

func (s *EdgeService) List(ctx context.Context, storyID string) ([]*domain.SceneEdge, error) {
	return s.repo.ListByStory(ctx, storyID)
}

func (s *EdgeService) Delete(ctx context.Context, storyID, from, to string) error {
	return s.repo.Delete(ctx, storyID, from, to)
}

func (s *EdgeService) DeleteByID(ctx context.Context, edgeID string) error {
	existing, err := s.repo.Get(ctx, edgeID)
	if err != nil {
		return fmt.Errorf("get edge: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("edge not found: %w", ErrNotFound)
	}
	return s.repo.DeleteByID(ctx, edgeID)
}
