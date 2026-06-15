package memory

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/memory"
)

type Service interface {
	StoreMemory(ctx context.Context, storyID, characterID, sceneID uuid.UUID, mType memory.MemType, summary string, importance float64) (*memory.Memory, error)
	RetrieveMemories(ctx context.Context, storyID, characterID uuid.UUID) ([]memory.Memory, error)
	SearchMemories(ctx context.Context, query memory.RetrievalQuery) ([]memory.RankedMemory, error)
}

type memoryService struct {
	store memory.Store
}

func NewService(store memory.Store) Service {
	return &memoryService{store: store}
}

func (s *memoryService) StoreMemory(ctx context.Context, storyID, characterID, sceneID uuid.UUID, mType memory.MemType, summary string, importance float64) (*memory.Memory, error) {
	m := &memory.Memory{
		StoryID:     storyID,
		CharacterID: characterID,
		SceneID:     sceneID,
		Type:        mType,
		Summary:     summary,
		Importance:  importance,
	}
	if err := s.store.Create(m); err != nil {
		return nil, fmt.Errorf("memory service: store: %w", err)
	}
	return m, nil
}

func (s *memoryService) RetrieveMemories(ctx context.Context, storyID, characterID uuid.UUID) ([]memory.Memory, error) {
	return s.store.RetrieveRecent(storyID, characterID, 20)
}

func (s *memoryService) SearchMemories(ctx context.Context, query memory.RetrievalQuery) ([]memory.RankedMemory, error) {
	return s.store.Search(query)
}
