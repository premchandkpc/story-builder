package scene

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/premchand/story-builder/internal/db"
	"github.com/premchand/story-builder/internal/graph"
	"github.com/premchand/story-builder/internal/scene"
)

func dbSceneTurnToSceneTurn(t db.SceneTurn) *scene.SceneTurn {
	actorIDs := make([]uuid.UUID, len(t.ActorIds))
	for i, a := range t.ActorIds {
		actorIDs[i] = db.FromUUID(a)
	}
	return &scene.SceneTurn{
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

type DBSceneService struct {
	q *db.Queries
}

func NewDBService(q *db.Queries) *DBSceneService {
	return &DBSceneService{q: q}
}

func (s *DBSceneService) StartScene(ctx context.Context, sceneID uuid.UUID) (*scene.SceneTurn, error) {
	turn, err := s.q.CreateSceneTurn(ctx, db.CreateSceneTurnParams{
		SceneID:    db.ToUUID(sceneID),
		TurnNumber: 1,
		ActorIds:   []pgtype.UUID{},
		Prompt:     "",
		Output:     "",
		Model:      "",
		Status:     "in_progress",
	})
	if err != nil {
		return nil, fmt.Errorf("start scene: %w", err)
	}
	return dbSceneTurnToSceneTurn(turn), nil
}

func (s *DBSceneService) NextTurn(ctx context.Context, sceneID uuid.UUID) (*scene.SceneTurn, error) {
	turns, err := s.q.ListSceneTurns(ctx, db.ToUUID(sceneID))
	if err != nil {
		return nil, fmt.Errorf("list turns: %w", err)
	}
	turn, err := s.q.CreateSceneTurn(ctx, db.CreateSceneTurnParams{
		SceneID:    db.ToUUID(sceneID),
		TurnNumber: int32(len(turns) + 1),
		ActorIds:   []pgtype.UUID{},
		Prompt:     "",
		Output:     "",
		Model:      "",
		Status:     "in_progress",
	})
	if err != nil {
		return nil, fmt.Errorf("next turn: %w", err)
	}
	return dbSceneTurnToSceneTurn(turn), nil
}

func (s *DBSceneService) FinishScene(ctx context.Context, sceneID uuid.UUID) (string, error) {
	if err := s.q.SetSceneStatus(ctx, db.SetSceneStatusParams{
		ID:     db.ToUUID(sceneID),
		Status: "completed",
	}); err != nil {
		return "", fmt.Errorf("finish scene: %w", err)
	}
	return "completed", nil
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
	if s.turns == nil {
		s.turns = make(map[uuid.UUID][]scene.SceneTurn)
	}
	turn := scene.SceneTurn{
		ID:         uuid.New(),
		NodeID:     sceneID,
		TurnNumber: 1,
		ActorIDs:   nil,
		Status:     "in_progress",
	}
	s.turns[sceneID] = append(s.turns[sceneID], turn)
	return &turn, nil
}

func (s *MemorySceneService) NextTurn(ctx context.Context, sceneID uuid.UUID) (*scene.SceneTurn, error) {
	existing := s.turns[sceneID]
	turn := scene.SceneTurn{
		ID:         uuid.New(),
		NodeID:     sceneID,
		TurnNumber: len(existing) + 1,
		ActorIDs:   nil,
		Status:     "in_progress",
	}
	s.turns[sceneID] = append(existing, turn)
	return &turn, nil
}

func (s *MemorySceneService) FinishScene(ctx context.Context, sceneID uuid.UUID) (string, error) {
	turns := s.turns[sceneID]
	for i := range turns {
		turns[i].Status = "completed"
	}
	s.turns[sceneID] = turns
	return "completed", nil
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
