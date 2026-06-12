package blueprint

import (
	"context"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/narrative"
)

type BlueprintService interface {
	Save(ctx context.Context, storyID uuid.UUID, bp *narrative.Blueprint) error
	Get(ctx context.Context, storyID uuid.UUID) (*narrative.Blueprint, error)
}

type MemoryBlueprintService struct {
	store *narrative.MemoryStore
}

func NewMemoryService() *MemoryBlueprintService {
	return &MemoryBlueprintService{store: narrative.NewMemoryStore()}
}

func (s *MemoryBlueprintService) Save(ctx context.Context, storyID uuid.UUID, bp *narrative.Blueprint) error {
	return s.store.Save(storyID, bp)
}

func (s *MemoryBlueprintService) Get(ctx context.Context, storyID uuid.UUID) (*narrative.Blueprint, error) {
	return s.store.Get(storyID)
}
