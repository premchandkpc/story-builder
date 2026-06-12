package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type PipelineStatus string

const (
	PipelineRunning   PipelineStatus = "running"
	PipelineCompleted PipelineStatus = "completed"
	PipelineFailed    PipelineStatus = "failed"
)

type WorkflowState struct {
	PipelineID  uuid.UUID     `json:"pipeline_id"`
	StoryID     uuid.UUID     `json:"story_id"`
	NodeID      uuid.UUID     `json:"node_id"`
	Status      PipelineStatus `json:"status"`
	CurrentStep string        `json:"current_step"`
	Progress    float64       `json:"progress"`
	StartedAt   time.Time     `json:"started_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	Error       string        `json:"error,omitempty"`
}

type WorkflowStore struct {
	client RedisClient
	ttl    time.Duration
}

func NewWorkflowStore(client RedisClient) *WorkflowStore {
	return &WorkflowStore{
		client: client,
		ttl:    1 * time.Hour,
	}
}

func (s *WorkflowStore) Set(ctx context.Context, pipelineID string, state *WorkflowState) error {
	key := fmt.Sprintf(string(PrefixPipeline), pipelineID)
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("workflow marshal: %w", err)
	}
	return s.client.Set(ctx, key, string(data), s.ttl)
}

func (s *WorkflowStore) Get(ctx context.Context, pipelineID string) (*WorkflowState, error) {
	key := fmt.Sprintf(string(PrefixPipeline), pipelineID)
	data, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	var state WorkflowState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return nil, fmt.Errorf("workflow unmarshal: %w", err)
	}
	return &state, nil
}

func (s *WorkflowStore) UpdateStatus(ctx context.Context, pipelineID string, status PipelineStatus, step string) error {
	state, err := s.Get(ctx, pipelineID)
	if err != nil {
		return err
	}
	state.Status = status
	state.CurrentStep = step
	state.UpdatedAt = time.Now()
	if status == PipelineCompleted {
		state.Progress = 1.0
	}
	return s.Set(ctx, pipelineID, state)
}
