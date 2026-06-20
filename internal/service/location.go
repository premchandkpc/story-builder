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
	existing, err := s.repo.Get(ctx, loc.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("location not found")
	}
	if loc.Name != "" {
		existing.Name = loc.Name
	}
	if loc.Description != "" {
		existing.Description = loc.Description
	}
	if loc.LocType != "" {
		existing.LocType = loc.LocType
	}
	if loc.ParentID != "" {
		existing.ParentID = loc.ParentID
	}
	if loc.Props != nil {
		existing.Props = loc.Props
	}
	if loc.Features != nil {
		existing.Features = loc.Features
	}
	if loc.Atmosphere != "" {
		existing.Atmosphere = loc.Atmosphere
	}
	if loc.Children != nil {
		existing.Children = loc.Children
	}
	return s.repo.Update(ctx, existing)
}

func (s *LocationService) DeleteByStory(ctx context.Context, storyID string) error {
	return s.repo.DeleteByStory(ctx, storyID)
}

func (s *LocationService) GetByName(ctx context.Context, storyID, name string) (*domain.Location, error) {
	return s.repo.GetByName(ctx, storyID, name)
}
