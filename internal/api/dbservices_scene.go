package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/db"
	"github.com/premchand/story-builder/internal/graph"
	"github.com/premchand/story-builder/internal/scene"
)

type dbSceneService struct{ q *db.Queries }

func NewDBSceneService(q *db.Queries) *dbSceneService {
	return &dbSceneService{q: q}
}

func (s *dbSceneService) StartScene(ctx context.Context, nodeID uuid.UUID) (*scene.SceneTurn, error) {
	return nil, fmt.Errorf("multi-agent scene requires LLM integration -- not implemented")
}

func (s *dbSceneService) NextTurn(ctx context.Context, nodeID uuid.UUID) (*scene.SceneTurn, error) {
	return nil, fmt.Errorf("multi-agent scene requires LLM integration -- not implemented")
}

func (s *dbSceneService) FinishScene(ctx context.Context, nodeID uuid.UUID) (string, error) {
	return "", fmt.Errorf("multi-agent scene requires LLM integration -- not implemented")
}

func (s *dbSceneService) GetTurns(ctx context.Context, nodeID uuid.UUID) ([]scene.SceneTurn, error) {
	turns, err := s.q.ListSceneTurns(ctx, toUUID(nodeID))
	if err != nil {
		return nil, err
	}
	result := make([]scene.SceneTurn, len(turns))
	for i, t := range turns {
		actorIDs := make([]uuid.UUID, len(t.ActorIds))
		for j, a := range t.ActorIds {
			actorIDs[j] = fromUUID(a)
		}
		result[i] = scene.SceneTurn{
			ID:         fromUUID(t.ID),
			NodeID:     fromUUID(t.NodeID),
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

func (s *dbSceneService) SetSceneStructure(ctx context.Context, nodeID uuid.UUID, ss graph.SceneStructure) error {
	return s.q.UpdateNodeSceneStructure(ctx, db.UpdateNodeSceneStructureParams{
		ID:             toUUID(nodeID),
		SceneStructure: jsonBytes(ss),
	})
}

func (s *dbSceneService) GetSceneStructure(ctx context.Context, nodeID uuid.UUID) (*graph.SceneStructure, error) {
	n, err := s.q.GetNode(ctx, toUUID(nodeID))
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
