package api

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/graph"
	"github.com/premchand/story-builder/internal/scene"
)

type memorySceneService struct {
	turns map[uuid.UUID][]scene.SceneTurn
}

func NewMemorySceneService() *memorySceneService {
	return &memorySceneService{turns: make(map[uuid.UUID][]scene.SceneTurn)}
}

func (s *memorySceneService) StartScene(ctx context.Context, nodeID uuid.UUID) (*scene.SceneTurn, error) {
	return nil, fmt.Errorf("multi-agent scene requires LLM integration -- not implemented in memory mode")
}

func (s *memorySceneService) NextTurn(ctx context.Context, nodeID uuid.UUID) (*scene.SceneTurn, error) {
	return nil, fmt.Errorf("multi-agent scene requires LLM integration -- not implemented in memory mode")
}

func (s *memorySceneService) FinishScene(ctx context.Context, nodeID uuid.UUID) (string, error) {
	return "", fmt.Errorf("multi-agent scene requires LLM integration -- not implemented in memory mode")
}

func (s *memorySceneService) GetTurns(ctx context.Context, nodeID uuid.UUID) ([]scene.SceneTurn, error) {
	turns := s.turns[nodeID]
	r := make([]scene.SceneTurn, len(turns))
	copy(r, turns)
	return r, nil
}

func (s *memorySceneService) SetSceneStructure(ctx context.Context, nodeID uuid.UUID, ss graph.SceneStructure) error {
	return nil
}

func (s *memorySceneService) GetSceneStructure(ctx context.Context, nodeID uuid.UUID) (*graph.SceneStructure, error) {
	return nil, nil
}
