package service

import (
	"context"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type SummaryService struct {
	repo repository.SummaryRepository
}

func NewSummaryService(repo repository.SummaryRepository) *SummaryService {
	return &SummaryService{repo: repo}
}

func (s *SummaryService) GetByLevel(ctx context.Context, storyID, level string) (*domain.Summary, error) {
	return s.repo.GetByLevel(ctx, storyID, level)
}

func (s *SummaryService) GetSceneSummary(ctx context.Context, storyID, sceneID string) (*domain.Summary, error) {
	return s.repo.GetSceneSummary(ctx, storyID, sceneID)
}
