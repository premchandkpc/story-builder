package domain

import "time"

type Job struct {
	ID               string     `bson:"_id" json:"id"`
	Type             string     `bson:"type" json:"type"`
	Status           string     `bson:"status" json:"status"`
	StoryID          string     `bson:"storyId" json:"storyId"`
	SceneID          string     `bson:"sceneId" json:"sceneId"`
	GenID            string     `bson:"genId,omitempty" json:"genId,omitempty"`
	RunID            string     `bson:"runId,omitempty" json:"runId,omitempty"`
	Error            string     `bson:"error,omitempty" json:"error,omitempty"`
	Attempts         int        `bson:"attempts" json:"attempts"`
	MaxRetries       int        `bson:"maxRetries" json:"maxRetries"`
	WorkerID         string     `bson:"workerId,omitempty" json:"workerId,omitempty"`
	HeartbeatAt      *time.Time `bson:"heartbeatAt,omitempty" json:"heartbeatAt,omitempty"`
	LeaseUntil       *time.Time `bson:"leaseUntil,omitempty" json:"leaseUntil,omitempty"`
	DeadLetterReason string     `bson:"deadLetterReason,omitempty" json:"deadLetterReason,omitempty"`
	Version          int        `bson:"version" json:"version"`
	CreatedAt        time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time  `bson:"updatedAt" json:"updatedAt"`
}

const (
	JobTypeGenerateScene = "generate_scene"
)

const (
	JobStatusPending    = "pending"
	JobStatusRunning    = "running"
	JobStatusDone       = "done"
	JobStatusFailed     = "failed"
	JobStatusDeadLetter = "dead_letter"
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
	PromptSnapshot   *PromptSnapshot `bson:"promptSnapshot,omitempty" json:"promptSnapshot,omitempty"`
	CreatedAt        time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time  `bson:"updatedAt" json:"updatedAt"`
}

type RunStep struct {
	ID               string         `bson:"_id" json:"id"`
	RunID            string         `bson:"runId" json:"runId"`
	StepName         string         `bson:"stepName" json:"stepName"`
	Status           string         `bson:"status" json:"status"`
	StartedAt        *time.Time     `bson:"startedAt,omitempty" json:"startedAt,omitempty"`
	FinishedAt       *time.Time     `bson:"finishedAt,omitempty" json:"finishedAt,omitempty"`
	PromptHash       string         `bson:"promptHash,omitempty" json:"promptHash,omitempty"`
	Model            string         `bson:"model,omitempty" json:"model,omitempty"`
	TokensIn         int            `bson:"tokensIn,omitempty" json:"tokensIn,omitempty"`
	TokensOut        int            `bson:"tokensOut,omitempty" json:"tokensOut,omitempty"`
	Error            string         `bson:"error,omitempty" json:"error,omitempty"`
	Artifacts        map[string]any `bson:"artifacts,omitempty" json:"artifacts,omitempty"`
	OutputSnippet    string         `bson:"outputSnippet,omitempty" json:"outputSnippet,omitempty"`
	PromptSnippet    string         `bson:"promptSnippet,omitempty" json:"promptSnippet,omitempty"`
	EstimatedCostUSD float64        `bson:"estimatedCostUsd,omitempty" json:"estimatedCostUsd,omitempty"`
	CreatedAt        time.Time      `bson:"createdAt" json:"createdAt"`
}

const (
	RunTypeGenerateScene  = "generate_scene"
	RunTypeRebuildSummary = "rebuild_summary"
	RunTypeExtractState   = "extract_state"
	RunTypeValidate       = "validate"
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

type PromptSection struct {
	Name    string `bson:"name" json:"name"`
	Tokens  int    `bson:"tokens" json:"tokens"`
	Snippet string `bson:"contentSnippet,omitempty" json:"contentSnippet,omitempty"`
}

type PromptSnapshot struct {
	System   string          `bson:"system,omitempty" json:"system,omitempty"`
	Tokens   int             `bson:"tokenCount,omitempty" json:"tokenCount,omitempty"`
	Sections []PromptSection `bson:"sections,omitempty" json:"sections,omitempty"`
}

type ModelCost struct {
	Tokens int     `bson:"tokens" json:"tokens"`
	Cost   float64 `bson:"cost" json:"cost"`
}

type CostSummary struct {
	TotalTokens    int                  `bson:"totalTokens" json:"totalTokens"`
	EstimatedCost  float64              `bson:"estimatedCost" json:"estimatedCost"`
	ByModel        map[string]ModelCost `bson:"byModel,omitempty" json:"byModel,omitempty"`
}

type RunStats struct {
	Total       int     `bson:"total" json:"total"`
	Completed   int     `bson:"completed" json:"completed"`
	Failed      int     `bson:"failed" json:"failed"`
	Cancelled   int     `bson:"cancelled" json:"cancelled"`
	Running     int     `bson:"running" json:"running"`
	FailureRate float64 `bson:"failureRate" json:"failureRate"`
}

type SceneLock struct {
	SceneID    string    `bson:"_id" json:"scene_id"`
	StoryID    string    `bson:"storyId" json:"story_id"`
	GenID      string    `bson:"genId,omitempty" json:"gen_id,omitempty"`
	WorkerID   string    `bson:"workerId" json:"worker_id"`
	AcquiredAt time.Time `bson:"acquiredAt" json:"acquired_at"`
	TTL        time.Time `bson:"ttl" json:"ttl"`
	Version    int       `bson:"version" json:"version"`
}
