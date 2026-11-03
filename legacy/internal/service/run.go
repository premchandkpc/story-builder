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

func (s *RunService) GetPromptSections(ctx context.Context, runID string) (*domain.PromptSnapshot, error) {
	run, err := s.runRepo.Get(ctx, runID)
	if err != nil || run == nil {
		return nil, err
	}
	return run.PromptSnapshot, nil
}

func (s *RunService) GetRunCost(ctx context.Context, runID string) (*domain.CostSummary, error) {
	steps, err := s.stepRepo.ListByRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	cost := &domain.CostSummary{
		ByModel: make(map[string]domain.ModelCost),
	}
	for _, step := range steps {
		cost.TotalTokens += step.TokensIn + step.TokensOut
		cost.EstimatedCost += step.EstimatedCostUSD
		mc := cost.ByModel[step.Model]
		mc.Tokens += step.TokensIn + step.TokensOut
		mc.Cost += step.EstimatedCostUSD
		cost.ByModel[step.Model] = mc
	}
	return cost, nil
}

func (s *RunService) GetStoryRunStats(ctx context.Context, storyID string) (*domain.RunStats, error) {
	runs, err := s.runRepo.ListByStory(ctx, storyID, 0)
	if err != nil {
		return nil, err
	}
	stats := &domain.RunStats{}
	for _, r := range runs {
		stats.Total++
		switch r.Status {
		case domain.RunStatusCompleted:
			stats.Completed++
		case domain.RunStatusFailed:
			stats.Failed++
		case domain.RunStatusCancelled:
			stats.Cancelled++
		case domain.RunStatusRunning:
			stats.Running++
		}
	}
	if stats.Total > 0 {
		stats.FailureRate = float64(stats.Failed) / float64(stats.Total)
	}
	return stats, nil
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

func (s *NarrativeEventService) ListByRun(ctx context.Context, runID string, limit int) ([]*domain.NarrativeEvent, error) {
	return s.repo.ListByRun(ctx, runID, limit)
}
