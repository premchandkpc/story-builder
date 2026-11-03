package orchestration

import (
	"context"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type RunRecorder struct {
	RunRepo  repository.RunRepository
	StepRepo repository.RunStepRepository
}

func NewRunRecorder(runRepo repository.RunRepository, stepRepo repository.RunStepRepository) *RunRecorder {
	return &RunRecorder{RunRepo: runRepo, StepRepo: stepRepo}
}

func (r *RunRecorder) CreateRun(ctx context.Context, storyID, sceneID, genID, runType string) (*domain.StoryRun, error) {
	now := time.Now()
	run := &domain.StoryRun{
		StoryID:   storyID,
		SceneID:   sceneID,
		GenID:     genID,
		RunType:   runType,
		Status:    domain.RunStatusRunning,
		StartedAt: &now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := r.RunRepo.Create(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

func (r *RunRecorder) RecordStep(ctx context.Context, runID, stepName string, opts StepRecordOptions) error {
	step := &domain.RunStep{
		RunID:            runID,
		StepName:         stepName,
		Status:           opts.Status,
		StartedAt:        opts.StartedAt,
		FinishedAt:       opts.FinishedAt,
		PromptHash:       opts.PromptHash,
		Model:            opts.Model,
		TokensIn:         opts.TokensIn,
		TokensOut:        opts.TokensOut,
		Error:            opts.Error,
		Artifacts:        opts.Artifacts,
		OutputSnippet:    opts.OutputSnippet,
		PromptSnippet:    opts.PromptSnippet,
		EstimatedCostUSD: opts.EstimatedCostUSD,
		CreatedAt:        time.Now(),
	}
	return r.StepRepo.Create(ctx, step)
}

type StepRecordOptions struct {
	Status           string
	StartedAt        *time.Time
	FinishedAt       *time.Time
	PromptHash       string
	Model            string
	TokensIn         int
	TokensOut        int
	Error            string
	Artifacts        map[string]any
	OutputSnippet    string
	PromptSnippet    string
	EstimatedCostUSD float64
}

func (r *RunRecorder) CompleteRun(ctx context.Context, runID string, status string, currentStep string, errorSummary string) error {
	run, err := r.RunRepo.Get(ctx, runID)
	if err != nil || run == nil {
		return err
	}
	now := time.Now()
	run.Status = status
	run.FinishedAt = &now
	run.CurrentStep = currentStep
	if errorSummary != "" {
		run.ErrorSummary = errorSummary
	}
	run.UpdatedAt = now
	return r.RunRepo.Update(ctx, run)
}

func (r *RunRecorder) UpdateRunStep(ctx context.Context, runID, stepName string, status string) error {
	run, err := r.RunRepo.Get(ctx, runID)
	if err != nil || run == nil {
		return err
	}
	run.CurrentStep = stepName
	run.UpdatedAt = time.Now()
	return r.RunRepo.Update(ctx, run)
}
