package domain

import "time"

type Job struct {
	ID         string     `bson:"_id" json:"id"`
	Type       string     `bson:"type" json:"type"`
	Status     string     `bson:"status" json:"status"`
	StoryID    string     `bson:"storyId" json:"storyId"`
	SceneID    string     `bson:"sceneId" json:"sceneId"`
	GenID      string     `bson:"genId,omitempty" json:"genId,omitempty"`
	RunID      string     `bson:"runId,omitempty" json:"runId,omitempty"`
	Error      string     `bson:"error,omitempty" json:"error,omitempty"`
	Attempts   int        `bson:"attempts" json:"attempts"`
	MaxRetries int        `bson:"maxRetries" json:"maxRetries"`
	LeaseUntil *time.Time `bson:"leaseUntil,omitempty" json:"leaseUntil,omitempty"`
	CreatedAt  time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt  time.Time  `bson:"updatedAt" json:"updatedAt"`
}

const (
	JobTypeGenerateScene = "generate_scene"
)

const (
	JobStatusPending = "pending"
	JobStatusRunning = "running"
	JobStatusDone    = "done"
	JobStatusFailed  = "failed"
)

type StoryRun struct {
	ID               string     `bson:"_id" json:"id"`
	StoryID          string     `bson:"storyId" json:"storyId"`
	SceneID          string     `bson:"sceneId,omitempty" json:"sceneId,omitempty"`
	GenID            string     `bson:"genId,omitempty" json:"genId,omitempty"`
	RunType          string     `bson:"runType" json:"runType"`
	Status           string     `bson:"status" json:"status"`
	CreatedBy        string     `bson:"createdBy,omitempty" json:"createdBy,omitempty"`
	StartedAt        *time.Time `bson:"startedAt,omitempty" json:"startedAt,omitempty"`
	FinishedAt       *time.Time `bson:"finishedAt,omitempty" json:"finishedAt,omitempty"`
	InputContextHash string     `bson:"inputContextHash,omitempty" json:"inputContextHash,omitempty"`
	CurrentStep      string     `bson:"currentStep,omitempty" json:"currentStep,omitempty"`
	ErrorSummary     string     `bson:"errorSummary,omitempty" json:"errorSummary,omitempty"`
	OutputGenID      string     `bson:"outputGenId,omitempty" json:"outputGenId,omitempty"`
	CreatedAt        time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time  `bson:"updatedAt" json:"updatedAt"`
}

type RunStep struct {
	ID         string         `bson:"_id" json:"id"`
	RunID      string         `bson:"runId" json:"runId"`
	StepName   string         `bson:"stepName" json:"stepName"`
	Status     string         `bson:"status" json:"status"`
	StartedAt  *time.Time     `bson:"startedAt,omitempty" json:"startedAt,omitempty"`
	FinishedAt *time.Time     `bson:"finishedAt,omitempty" json:"finishedAt,omitempty"`
	PromptHash string         `bson:"promptHash,omitempty" json:"promptHash,omitempty"`
	Model      string         `bson:"model,omitempty" json:"model,omitempty"`
	TokensIn   int            `bson:"tokensIn,omitempty" json:"tokensIn,omitempty"`
	TokensOut  int            `bson:"tokensOut,omitempty" json:"tokensOut,omitempty"`
	Error      string         `bson:"error,omitempty" json:"error,omitempty"`
	Artifacts  map[string]any `bson:"artifacts,omitempty" json:"artifacts,omitempty"`
	CreatedAt  time.Time      `bson:"createdAt" json:"createdAt"`
}

const (
	RunTypeGenerateScene   = "generate_scene"
	RunTypeRebuildSummary  = "rebuild_summary"
	RunTypeExtractState    = "extract_state"
	RunTypeValidate        = "validate"
)

const (
	RunStatusQueued    = "queued"
	RunStatusRunning   = "running"
	RunStatusPartial   = "partial"
	RunStatusCompleted = "completed"
	RunStatusFailed    = "failed"
	RunStatusCancelled = "cancelled"
)

const (
	StepStatusPending = "pending"
	StepStatusRunning = "running"
	StepStatusDone    = "done"
	StepStatusFailed  = "failed"
	StepStatusSkipped = "skipped"
)
