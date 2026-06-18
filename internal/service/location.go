package service

import (
	"context"
	"fmt"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type LocationService struct {
	repo repository.LocationRepository
}

func NewLocationService(repo repository.LocationRepository) *LocationService {
	return &LocationService{repo: repo}
}

func (s *LocationService) Create(ctx context.Context, loc *domain.Location) error {
	return s.repo.Create(ctx, loc)
}

func (s *LocationService) Get(ctx context.Context, id string) (*domain.Location, error) {
	return s.repo.Get(ctx, id)
}

func (s *LocationService) ListByStory(ctx context.Context, storyID string) ([]*domain.Location, error) {
	return s.repo.ListByStory(ctx, storyID)
}

func (s *LocationService) Update(ctx context.Context, loc *domain.Location) error {
	return s.repo.Update(ctx, loc)
}

func (s *LocationService) DeleteByStory(ctx context.Context, storyID string) error {
	return s.repo.DeleteByStory(ctx, storyID)
}

func (s *LocationService) GetByName(ctx context.Context, storyID, name string) (*domain.Location, error) {
	locations, err := s.repo.ListByStory(ctx, storyID)
	if err != nil {
		return nil, fmt.Errorf("list locations: %w", err)
	}
	for _, l := range locations {
		if l.Name == name {
			return l, nil
		}
	}
	return nil, nil
}
