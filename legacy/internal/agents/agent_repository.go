package agents

import (
	"context"

	"github.com/premchand/story-builder/internal/domain"
)

type SceneTurnRepository interface {
	Create(ctx context.Context, t *domain.SceneTurn) error
	Update(ctx context.Context, t *domain.SceneTurn) error
}
