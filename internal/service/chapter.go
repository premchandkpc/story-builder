package service

import (
	"context"
	"fmt"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type ChapterSvc struct {
	repo repository.ChapterRepository
}

func NewChapterSvc(repo repository.ChapterRepository) *ChapterSvc {
	return &ChapterSvc{repo: repo}
}

func (s *ChapterSvc) Create(ctx context.Context, c *domain.Chapter) (*domain.Chapter, error) {
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, fmt.Errorf("create chapter: %w", err)
	}
	return c, nil
}

func (s *ChapterSvc) Get(ctx context.Context, id string) (*domain.Chapter, error) {
	return s.repo.Get(ctx, id)
}

func (s *ChapterSvc) ListByStory(ctx context.Context, storyID string) ([]*domain.Chapter, error) {
	return s.repo.ListByStory(ctx, storyID)
}

func (s *ChapterSvc) ListByAct(ctx context.Context, storyID string, actNumber int) ([]*domain.Chapter, error) {
	return s.repo.ListByAct(ctx, storyID, actNumber)
}

func (s *ChapterSvc) Update(ctx context.Context, c *domain.Chapter) (*domain.Chapter, error) {
	c.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, fmt.Errorf("update chapter: %w", err)
	}
	return c, nil
}
