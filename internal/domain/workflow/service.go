package workflow

import (
	"context"

	"github.com/google/uuid"
)

type WorkflowService interface {
	StartGenerationPipeline(ctx context.Context, nodeID uuid.UUID) error
	OnGenerationCompleted(ctx context.Context, generationID uuid.UUID) error
	OnValidationCompleted(ctx context.Context, generationID uuid.UUID, passed bool) error
	OnStateExtracted(ctx context.Context, generationID uuid.UUID) error
	OnSummaryUpdated(ctx context.Context, nodeID uuid.UUID) error
}
