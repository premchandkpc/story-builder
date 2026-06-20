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

func (s *ChapterSvc) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *ChapterSvc) Update(ctx context.Context, c *domain.Chapter) (*domain.Chapter, error) {
	existing, err := s.repo.Get(ctx, c.ID)
	if err != nil {
		return nil, fmt.Errorf("get chapter for update: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("chapter not found")
	}
	if c.Title != "" {
		existing.Title = c.Title
	}
	if c.Summary != "" {
		existing.Summary = c.Summary
	}
	if c.Goal != "" {
		existing.Goal = c.Goal
	}
	if c.Scenes != nil {
		existing.Scenes = c.Scenes
	}
	if c.Status != "" {
		existing.Status = c.Status
	}
	if c.ActNumber != 0 {
		existing.ActNumber = c.ActNumber
	}
	if c.ChapterNum != 0 {
		existing.ChapterNum = c.ChapterNum
	}
	existing.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update chapter: %w", err)
	}
	return existing, nil
}
