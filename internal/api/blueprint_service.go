package api

import (
	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/narrative"
)

type inMemoryBlueprintService struct {
	store *narrative.MemoryStore
}

func NewInMemoryBlueprintService() *inMemoryBlueprintService {
	return &inMemoryBlueprintService{store: narrative.NewMemoryStore()}
}

func (s *inMemoryBlueprintService) Save(storyID uuid.UUID, bp *narrative.Blueprint) error {
	if s.store == nil {
		return nil
	}
	return s.store.Save(storyID, bp)
}

func (s *inMemoryBlueprintService) Get(storyID uuid.UUID) (*narrative.Blueprint, error) {
	if s.store == nil {
		return nil, nil
	}
	return s.store.Get(storyID)
}
