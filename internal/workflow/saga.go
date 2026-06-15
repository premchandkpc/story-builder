package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SagaStep string

const (
	SagaPlanScene          SagaStep = "plan_scene"
	SagaRetrieveMemory     SagaStep = "retrieve_memory"
	SagaRetrieveRel        SagaStep = "retrieve_relationships"
	SagaBuildContext       SagaStep = "build_context"
	SagaCompilePrompt      SagaStep = "compile_prompt"
	SagaGenerate           SagaStep = "generate"
	SagaValidate           SagaStep = "validate"
	SagaExtractMemory      SagaStep = "extract_memory"
	SagaUpdateRel          SagaStep = "update_relationships"
	SagaUpdateTimeline     SagaStep = "update_timeline"
	SagaPersist            SagaStep = "persist"
	SagaEmitEvents         SagaStep = "emit_events"
)

type SagaState struct {
	ID             uuid.UUID         `json:"id"`
	JobID          uuid.UUID         `json:"job_id"`
	StoryID        uuid.UUID         `json:"story_id"`
	SceneID        uuid.UUID         `json:"scene_id"`
	CurrentStep    SagaStep          `json:"current_step"`
	CompletedSteps []SagaStep        `json:"completed_steps"`
	FailedStep     *SagaStep         `json:"failed_step,omitempty"`
	Error          string            `json:"error,omitempty"`
	Attempts       int               `json:"attempts"`
	MaxAttempts    int               `json:"max_attempts"`
	Status         JobStatus         `json:"status"`
	Compensations  []SagaStep        `json:"compensations_run,omitempty"`
	StartedAt      time.Time         `json:"started_at"`
	CompletedAt    *time.Time        `json:"completed_at,omitempty"`
	StepDurations  map[string]int64  `json:"step_durations_ms,omitempty"`
}

type SagaEngine struct {
	store Store
}

func NewSagaEngine(store Store) *SagaEngine {
	return &SagaEngine{store: store}
}

var generationSaga = []SagaStep{
	SagaPlanScene,
	SagaRetrieveMemory,
	SagaRetrieveRel,
	SagaBuildContext,
	SagaCompilePrompt,
	SagaGenerate,
	SagaValidate,
	SagaExtractMemory,
	SagaUpdateRel,
	SagaUpdateTimeline,
	SagaPersist,
	SagaEmitEvents,
}

var compensationMap = map[SagaStep][]SagaStep{
	SagaPlanScene:      {},
	SagaRetrieveMemory: {},
	SagaRetrieveRel:    {},
	SagaBuildContext:   {},
	SagaCompilePrompt:  {},
	SagaGenerate:       {SagaPlanScene},
	SagaValidate:       {SagaGenerate, SagaPlanScene},
	SagaExtractMemory:  {SagaValidate, SagaGenerate, SagaPlanScene},
	SagaUpdateRel:      {SagaExtractMemory, SagaValidate, SagaGenerate, SagaPlanScene},
	SagaUpdateTimeline: {SagaUpdateRel, SagaExtractMemory, SagaValidate, SagaGenerate, SagaPlanScene},
	SagaPersist:        {SagaUpdateTimeline, SagaUpdateRel, SagaExtractMemory, SagaValidate, SagaGenerate, SagaPlanScene},
	SagaEmitEvents:     {SagaPersist, SagaUpdateTimeline, SagaUpdateRel, SagaExtractMemory, SagaValidate, SagaGenerate, SagaPlanScene},
}

func (e *SagaEngine) StartGenerationSaga(ctx context.Context, storyID, sceneID uuid.UUID) (*SagaState, error) {
	state := &SagaState{
		ID:            uuid.New(),
		StoryID:       storyID,
		SceneID:       sceneID,
		CurrentStep:   generationSaga[0],
		CompletedSteps: []SagaStep{},
		Attempts:      0,
		MaxAttempts:   3,
		Status:        StatusRunning,
		StepDurations: make(map[string]int64),
		StartedAt:     time.Now(),
	}
	return state, nil
}

func (e *SagaEngine) Advance(ctx context.Context, state *SagaState, step SagaStep, durationMs int64) error {
	state.CompletedSteps = append(state.CompletedSteps, step)
	state.StepDurations[string(step)] = durationMs

	for i, s := range generationSaga {
		if s == step && i+1 < len(generationSaga) {
			state.CurrentStep = generationSaga[i+1]
			return nil
		}
	}

	now := time.Now()
	state.CompletedAt = &now
	state.Status = StatusCompleted
	state.CurrentStep = ""
	return nil
}

func (e *SagaEngine) Fail(ctx context.Context, state *SagaState, failedStep SagaStep, err error) error {
	state.FailedStep = &failedStep
	state.Error = err.Error()
	state.Attempts++

	if state.Attempts >= state.MaxAttempts {
		state.Status = StatusFailed
		compensations := compensationMap[failedStep]
		for i := len(compensations) - 1; i >= 0; i-- {
			state.Compensations = append(state.Compensations, compensations[i])
		}
		return fmt.Errorf("saga %s failed at step %s after %d attempts: %w", state.ID, failedStep, state.Attempts, err)
	}

	state.Status = StatusPending
	state.CurrentStep = generationSaga[0]
	return nil
}

func (e *SagaEngine) GetStepIndex(step SagaStep) int {
	for i, s := range generationSaga {
		if s == step {
			return i
		}
	}
	return -1
}
