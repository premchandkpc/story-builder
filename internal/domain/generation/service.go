package generation

import (
	"context"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/compiler"
)

type GenerationService interface {
	Generate(ctx context.Context, nodeID uuid.UUID) (*compiler.Generation, error)
	AcceptGeneration(ctx context.Context, nodeID, generationID uuid.UUID) error
	ListGenerations(ctx context.Context, nodeID uuid.UUID) ([]compiler.Generation, error)
	GetGeneration(ctx context.Context, id uuid.UUID) (*compiler.Generation, error)
	IsStale(ctx context.Context, nodeID uuid.UUID, contextHash string) (bool, error)
}

type GenerationRepository interface {
	Create(ctx context.Context, g *compiler.Generation) error
	Get(ctx context.Context, id uuid.UUID) (*compiler.Generation, error)
	ListByNode(ctx context.Context, nodeID uuid.UUID) ([]compiler.Generation, error)
	UpdateOutput(ctx context.Context, id uuid.UUID, output, model string) error
	UpdateValidation(ctx context.Context, id uuid.UUID, validationData []byte) error
	Accept(ctx context.Context, id uuid.UUID) error
	GetAcceptedByNode(ctx context.Context, nodeID uuid.UUID) (*compiler.Generation, error)
}
