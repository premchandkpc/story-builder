package scene

import (
	"context"

	"github.com/premchand/story-builder/internal/domain"
)

type TurnRepository interface {
	Create(ctx context.Context, t *domain.SceneTurn) error
	Get(ctx context.Context, id string) (*domain.SceneTurn, error)
	Update(ctx context.Context, t *domain.SceneTurn) error
	ListByScene(ctx context.Context, sceneID string) ([]*domain.SceneTurn, error)
	ListByRole(ctx context.Context, sceneID, role string) ([]*domain.SceneTurn, error)
	DeleteByScene(ctx context.Context, sceneID string) error
}

type ActorRepository interface {
	Create(ctx context.Context, r *domain.AgentRun) error
	List(ctx context.Context, filter domain.AgentRunFilter) ([]*domain.AgentRun, error)
	DeleteByStory(ctx context.Context, storyID string) error
}

type CanonDeltaRepository interface {
	Create(ctx context.Context, d *domain.CanonDelta) error
	ListByScene(ctx context.Context, sceneID string) ([]*domain.CanonDelta, error)
	ListByStory(ctx context.Context, storyID string) ([]*domain.CanonDelta, error)
	DeleteByStory(ctx context.Context, storyID string) error
}

type TurnOrchestrator struct {
	turnRepo   TurnRepository
	actorRepo  ActorRepository
	canonRepo  CanonDeltaRepository
}

func NewTurnOrchestrator(
	turnRepo TurnRepository,
	actorRepo ActorRepository,
	canonRepo CanonDeltaRepository,
) *TurnOrchestrator {
	return &TurnOrchestrator{
		turnRepo:  turnRepo,
		actorRepo: actorRepo,
		canonRepo: canonRepo,
	}
}

func (o *TurnOrchestrator) GetTurns(ctx context.Context, sceneID string) ([]*domain.SceneTurn, error) {
	return o.turnRepo.ListByScene(ctx, sceneID)
}

func (o *TurnOrchestrator) GetTurnsByRole(ctx context.Context, sceneID, role string) ([]*domain.SceneTurn, error) {
	return o.turnRepo.ListByRole(ctx, sceneID, role)
}

func (o *TurnOrchestrator) GetCanonDeltas(ctx context.Context, sceneID string) ([]*domain.CanonDelta, error) {
	return o.canonRepo.ListByScene(ctx, sceneID)
}

func (o *TurnOrchestrator) RecordStateDelta(ctx context.Context, d *domain.CanonDelta) error {
	return o.canonRepo.Create(ctx, d)
}
