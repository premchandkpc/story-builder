package river

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
)

type GenerateSceneArgs struct {
	StoryID       uuid.UUID   `json:"story_id"`
	NodeID        uuid.UUID   `json:"node_id"`
	ContextHash   string      `json:"context_hash"`
	CharacterRefs []uuid.UUID `json:"character_refs"`
	LocationRef   *uuid.UUID  `json:"location_ref"`
	BeatIntent    string      `json:"beat_intent"`
	POV           string      `json:"pov"`
	Tone          string      `json:"tone"`
	TargetWords   int         `json:"target_words"`
}

func (GenerateSceneArgs) Kind() string { return "generate_scene" }

type GenerateSceneWorker struct {
	river.WorkerDefaults[GenerateSceneArgs]
}

func (w *GenerateSceneWorker) Work(ctx context.Context, job *river.Job[GenerateSceneArgs]) error {
	args := job.Args
	_ = args
	return nil
}

type ExtractStateArgs struct {
	StoryID      uuid.UUID `json:"story_id"`
	NodeID       uuid.UUID `json:"node_id"`
	GenerationID uuid.UUID `json:"generation_id"`
	SceneText    string    `json:"scene_text"`
}

func (ExtractStateArgs) Kind() string { return "extract_state" }

type ExtractStateWorker struct {
	river.WorkerDefaults[ExtractStateArgs]
}

func (w *ExtractStateWorker) Work(ctx context.Context, job *river.Job[ExtractStateArgs]) error {
	args := job.Args
	_ = args
	return nil
}

type UpdateSummaryArgs struct {
	StoryID         uuid.UUID `json:"story_id"`
	NodeID          uuid.UUID `json:"node_id"`
	PreviousSummary string    `json:"previous_summary"`
	AcceptedScene   string    `json:"accepted_scene"`
}

func (UpdateSummaryArgs) Kind() string { return "update_summary" }

type UpdateSummaryWorker struct {
	river.WorkerDefaults[UpdateSummaryArgs]
}

func (w *UpdateSummaryWorker) Work(ctx context.Context, job *river.Job[UpdateSummaryArgs]) error {
	args := job.Args
	_ = args
	return nil
}

type MergeBranchesArgs struct {
	StoryID     uuid.UUID `json:"story_id"`
	JoinNodeID  uuid.UUID `json:"join_node_id"`
	SummaryA    string    `json:"summary_a"`
	SummaryB    string    `json:"summary_b"`
	TimelineNote string   `json:"timeline_note"`
}

func (MergeBranchesArgs) Kind() string { return "merge_branches" }

type MergeBranchesWorker struct {
	river.WorkerDefaults[MergeBranchesArgs]
}

func (w *MergeBranchesWorker) Work(ctx context.Context, job *river.Job[MergeBranchesArgs]) error {
	args := job.Args
	_ = args
	return nil
}

type ValidateSceneArgs struct {
	StoryID       uuid.UUID `json:"story_id"`
	NodeID        uuid.UUID `json:"node_id"`
	GenerationID  uuid.UUID `json:"generation_id"`
	CompiledCanon string    `json:"compiled_canon"`
	CharState     string    `json:"char_state"`
	SceneText     string    `json:"scene_text"`
}

func (ValidateSceneArgs) Kind() string { return "validate_scene" }

type ValidateSceneWorker struct {
	river.WorkerDefaults[ValidateSceneArgs]
}

func (w *ValidateSceneWorker) Work(ctx context.Context, job *river.Job[ValidateSceneArgs]) error {
	args := job.Args
	_ = args
	return nil
}

func Workers() *river.Workers {
	workers := river.NewWorkers()
	river.AddWorker(workers, &GenerateSceneWorker{})
	river.AddWorker(workers, &ExtractStateWorker{})
	river.AddWorker(workers, &UpdateSummaryWorker{})
	river.AddWorker(workers, &MergeBranchesWorker{})
	river.AddWorker(workers, &ValidateSceneWorker{})
	return workers
}

const (
	QueueDefault  = "default"
	QueueGenerate = "generate"
	QueueExtract  = "extract"
	QueueMerge    = "merge"
	QueueValidate = "validate"
)

type Config struct {
	DatabaseURL string
	Workers     *river.Workers
}

type InsertGenerateSceneParams struct {
	Args        GenerateSceneArgs
	Queue       string
	MaxAttempts int
	SchedFor    *time.Time
}
