package agents

import (
	"context"
	"time"

	"github.com/premchand/story-builder/internal/domain"
)

type AgentContext struct {
	StoryID        string
	SceneID        string
	TurnID         string
	Story          *domain.Story
	Scene          *domain.Scene
	Characters     []*domain.Character
	CharStates     []*domain.CharacterState
	Bible          *domain.StoryBible
	BluePrint      *domain.StoryBlueprint
	Edges          []*domain.SceneEdge
	Turns          []*domain.SceneTurn
	Timeline       []*domain.TimelineEvent
	Memories       map[string][]*domain.CharacterMemory
	CanonDeltas    []*domain.CanonDelta
	Summaries      []*domain.Summary
	ParticipantIDs []string
}

type AgentInput struct {
	Ctx       *AgentContext
	Payload   map[string]any
	Directive string
}

type AgentOutput struct {
	Content   string
	Data      map[string]any
	Decisions map[string]any
	Status    string
	Error     string
}

type Agent interface {
	Name() string
	Role() string
	Run(ctx context.Context, input AgentInput) (*AgentOutput, error)
}

type AgentRunner func(ctx context.Context, input AgentInput) (*AgentOutput, error)

type AgentSpec struct {
	Name     string
	Role     string
	Model    string
	MaxTurns int
	Timeout  time.Duration
	SystemPrompt string
	Runner   AgentRunner
}

type OrchestrationPlan struct {
	SceneID     string
	TurnOrder   []TurnStep
	MaxTurns    int
	Directive   string
	Proposals   []CharacterProposal
}

type TurnStep struct {
	AgentType string
	AgentID   string
	Phase     string
	Required  bool
	Blocking  bool
}

type OrchestrationResult struct {
	SceneID     string
	Turns       []*domain.SceneTurn
	Accepted    bool
	CriticScore float64
	Error       string
}
