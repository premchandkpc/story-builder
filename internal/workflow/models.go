package workflow

import (
	"time"

	"github.com/google/uuid"
)

type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusRunning    JobStatus = "running"
	StatusPaused     JobStatus = "paused"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
	StatusCancelled  JobStatus = "cancelled"
)

type JobStep string

const (
	StepPlanner         JobStep = "planner"
	StepMemoryRetrieval JobStep = "memory_retrieval"
	StepRelRetrieval    JobStep = "relationship_retrieval"
	StepContextBuild    JobStep = "context_build"
	StepPromptCompile   JobStep = "prompt_compile"
	StepGeneration      JobStep = "generation"
	StepValidation      JobStep = "validation"
	StepMemExtract      JobStep = "memory_extraction"
	StepRelUpdate       JobStep = "relationship_update"
	StepTimelineUpdate  JobStep = "timeline_update"
	StepPersist         JobStep = "persist"
	StepEmitEvents      JobStep = "emit_events"
)

type JobType string

const (
	TypeGenerateScene     JobType = "generate_scene"
	TypeGenerateChapter   JobType = "generate_chapter"
	TypeGenerateStory     JobType = "generate_story"
	TypeGenerateDialogue  JobType = "generate_dialogue"
	TypeValidateScene     JobType = "validate_scene"
	TypeExtractMemories   JobType = "extract_memories"
)

type Job struct {
	ID           uuid.UUID     `json:"id"`
	Type         JobType       `json:"type"`
	StoryID      uuid.UUID     `json:"story_id"`
	ChapterID    uuid.UUID     `json:"chapter_id,omitempty"`
	SceneID      uuid.UUID     `json:"scene_id,omitempty"`
	Status       JobStatus     `json:"status"`
	CurrentStep  JobStep       `json:"current_step,omitempty"`
	Attempts     int           `json:"attempts"`
	MaxAttempts  int           `json:"max_attempts"`
	Error        string        `json:"error,omitempty"`
	Payload      []byte        `json:"payload,omitempty"`
	Result       []byte        `json:"result,omitempty"`
	StartedAt    *time.Time    `json:"started_at,omitempty"`
	CompletedAt  *time.Time    `json:"completed_at,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
}

type Store interface {
	Create(job *Job) error
	Get(id uuid.UUID) (*Job, error)
	Update(job *Job) error
	List(storyID uuid.UUID, status JobStatus) ([]Job, error)
	Cancel(id uuid.UUID) error
	Retry(id uuid.UUID) error
}

type WorkflowEngine struct {
	store Store
}

func NewWorkflowEngine(store Store) *WorkflowEngine {
	return &WorkflowEngine{store: store}
}

func (e *WorkflowEngine) CreateGenerationJob(jobType JobType, storyID uuid.UUID, sceneID uuid.UUID, payload []byte) (*Job, error) {
	job := &Job{
		ID:          uuid.New(),
		Type:        jobType,
		StoryID:     storyID,
		SceneID:     sceneID,
		Status:      StatusPending,
		CurrentStep: StepPlanner,
		Attempts:    0,
		MaxAttempts: 3,
		Payload:     payload,
		CreatedAt:   time.Now(),
	}
	if err := e.store.Create(job); err != nil {
		return nil, err
	}
	return job, nil
}
