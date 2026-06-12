package ledger

import (
	"context"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/ledger"
)

type LedgerService interface {
	GetState(ctx context.Context, storyID, characterID, asOfNode uuid.UUID) (*ledger.CharacterState, error)
	GetStatesForNode(ctx context.Context, storyID, nodeID uuid.UUID) (map[uuid.UUID]ledger.CharacterState, error)
	ApplyDeltas(ctx context.Context, storyID, nodeID uuid.UUID, deltas ledger.StateDeltas) error
	GetStateAtBranch(ctx context.Context, storyID, nodeID uuid.UUID) (map[uuid.UUID]ledger.CharacterState, error)
}

type LedgerRepository interface {
	GetState(ctx context.Context, storyID, characterID, asOfNode uuid.UUID) (*ledger.CharacterState, error)
	GetStatesForNode(ctx context.Context, storyID, nodeID uuid.UUID) (map[uuid.UUID]ledger.CharacterState, error)
	UpsertState(ctx context.Context, state ledger.CharacterState) error
}
