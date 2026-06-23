package service

import (
	"context"
	"fmt"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type RunService struct {
	runRepo  repository.RunRepository
	stepRepo repository.RunStepRepository
	jobRepo  repository.JobRepository
}

func NewRunService(runRepo repository.RunRepository, stepRepo repository.RunStepRepository, jobRepo repository.JobRepository) *RunService {
	return &RunService{runRepo: runRepo, stepRepo: stepRepo, jobRepo: jobRepo}
}

func (s *RunService) Create(ctx context.Context, run *domain.StoryRun) error {
	return s.runRepo.Create(ctx, run)
}

func (s *RunService) Get(ctx context.Context, id string) (*domain.StoryRun, error) {
	return s.runRepo.Get(ctx, id)
}

func (s *RunService) ListByStory(ctx context.Context, storyID string, limit int) ([]*domain.StoryRun, error) {
	return s.runRepo.ListByStory(ctx, storyID, limit)
}

func (s *RunService) ListByScene(ctx context.Context, sceneID string, limit int) ([]*domain.StoryRun, error) {
	return s.runRepo.ListByScene(ctx, sceneID, limit)
}

func (s *RunService) ListSteps(ctx context.Context, runID string) ([]*domain.RunStep, error) {
	return s.stepRepo.ListByRun(ctx, runID)
}

func (s *RunService) Cancel(ctx context.Context, runID string) error {
	run, err := s.runRepo.Get(ctx, runID)
	if err != nil {
		return fmt.Errorf("get run: %w", err)
	}
	if run == nil {
		return fmt.Errorf("run not found: %w", ErrNotFound)
	}
	if run.Status == domain.RunStatusCompleted || run.Status == domain.RunStatusCancelled {
		return fmt.Errorf("run already %s", run.Status)
	}
	now := time.Now()
	run.Status = domain.RunStatusCancelled
	run.FinishedAt = &now
	run.ErrorSummary = "cancelled by user"
	if err := s.runRepo.Update(ctx, run); err != nil {
		return fmt.Errorf("cancel run: %w", err)
	}
	return nil
}

func (s *RunService) AddStep(ctx context.Context, step *domain.RunStep) error {
	return s.stepRepo.Create(ctx, step)
}

type NarrativeEventService struct {
	repo repository.NarrativeEventRepository
}

func NewNarrativeEventService(repo repository.NarrativeEventRepository) *NarrativeEventService {
	return &NarrativeEventService{repo: repo}
}

func (s *NarrativeEventService) Append(ctx context.Context, e *domain.NarrativeEvent) error {
	return s.repo.Append(ctx, e)
}

func (s *NarrativeEventService) ListByStory(ctx context.Context, storyID string, limit int) ([]*domain.NarrativeEvent, error) {
	return s.repo.ListByStory(ctx, storyID, limit)
}

func (s *NarrativeEventService) ListByScene(ctx context.Context, sceneID string, limit int) ([]*domain.NarrativeEvent, error) {
	return s.repo.ListByScene(ctx, sceneID, limit)
}
