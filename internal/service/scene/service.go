package scene

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/db"
	"github.com/premchand/story-builder/internal/graph"
	"github.com/premchand/story-builder/internal/scene"
)

type DBSceneService struct {
	q *db.Queries
}

func NewDBService(q *db.Queries) *DBSceneService {
	return &DBSceneService{q: q}
}

func (s *DBSceneService) StartScene(ctx context.Context, sceneID uuid.UUID) (*scene.SceneTurn, error) {
	return nil, fmt.Errorf("multi-agent scene requires LLM integration -- not implemented")
}

func (s *DBSceneService) NextTurn(ctx context.Context, sceneID uuid.UUID) (*scene.SceneTurn, error) {
	return nil, fmt.Errorf("multi-agent scene requires LLM integration -- not implemented")
}

func (s *DBSceneService) FinishScene(ctx context.Context, sceneID uuid.UUID) (string, error) {
	return "", fmt.Errorf("multi-agent scene requires LLM integration -- not implemented")
}

func (s *DBSceneService) GetTurns(ctx context.Context, sceneID uuid.UUID) ([]scene.SceneTurn, error) {
	turns, err := s.q.ListSceneTurns(ctx, db.ToUUID(sceneID))
	if err != nil {
		return nil, err
	}
	result := make([]scene.SceneTurn, len(turns))
	for i, t := range turns {
		actorIDs := make([]uuid.UUID, len(t.ActorIds))
		for j, a := range t.ActorIds {
			actorIDs[j] = db.FromUUID(a)
		}
		result[i] = scene.SceneTurn{
			ID:         db.FromUUID(t.ID),
			NodeID:     db.FromUUID(t.SceneID),
			TurnNumber: int(t.TurnNumber),
			ActorIDs:   actorIDs,
			Prompt:     t.Prompt,
			Output:     t.Output,
			Model:      t.Model,
			Status:     t.Status,
			CreatedAt:  t.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *DBSceneService) SetSceneStructure(ctx context.Context, sceneID uuid.UUID, ss graph.SceneStructure) error {
	return s.q.UpdateSceneStructure(ctx, db.UpdateSceneStructureParams{
		ID:             db.ToUUID(sceneID),
		SceneStructure: db.JSONBytes(ss),
	})
}

func (s *DBSceneService) GetSceneStructure(ctx context.Context, sceneID uuid.UUID) (*graph.SceneStructure, error) {
	n, err := s.q.GetScene(ctx, db.ToUUID(sceneID))
	if err != nil {
		return nil, err
	}
	if len(n.SceneStructure) == 0 {
		return nil, nil
	}
	var ss graph.SceneStructure
	if err := json.Unmarshal(n.SceneStructure, &ss); err != nil {
		return nil, err
	}
	return &ss, nil
}

type MemorySceneService struct {
	turns map[uuid.UUID][]scene.SceneTurn
}

func NewMemoryService() *MemorySceneService {
	return &MemorySceneService{turns: make(map[uuid.UUID][]scene.SceneTurn)}
}

func (s *MemorySceneService) StartScene(ctx context.Context, sceneID uuid.UUID) (*scene.SceneTurn, error) {
	return nil, fmt.Errorf("multi-agent scene requires LLM integration -- not implemented in memory mode")
}

func (s *MemorySceneService) NextTurn(ctx context.Context, sceneID uuid.UUID) (*scene.SceneTurn, error) {
	return nil, fmt.Errorf("multi-agent scene requires LLM integration -- not implemented in memory mode")
}

func (s *MemorySceneService) FinishScene(ctx context.Context, sceneID uuid.UUID) (string, error) {
	return "", fmt.Errorf("multi-agent scene requires LLM integration -- not implemented in memory mode")
}

func (s *MemorySceneService) GetTurns(ctx context.Context, sceneID uuid.UUID) ([]scene.SceneTurn, error) {
	turns := s.turns[sceneID]
	r := make([]scene.SceneTurn, len(turns))
	copy(r, turns)
	return r, nil
}

func (s *MemorySceneService) SetSceneStructure(ctx context.Context, sceneID uuid.UUID, ss graph.SceneStructure) error {
	return nil
}

func (s *MemorySceneService) GetSceneStructure(ctx context.Context, sceneID uuid.UUID) (*graph.SceneStructure, error) {
	return nil, nil
}
