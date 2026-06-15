package edge

import (
	"context"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/db"
	"github.com/premchand/story-builder/internal/graph"
)

type Service interface {
	Create(ctx context.Context, storyID, fromNode, toNode uuid.UUID, edgeType string) error
	List(ctx context.Context, storyID uuid.UUID) ([]graph.Edge, error)
}

type memoryService struct {
	graph *graph.MemoryStore
}

func NewMemoryService(gs *graph.MemoryStore) *memoryService {
	return &memoryService{graph: gs}
}

func (s *memoryService) Create(ctx context.Context, storyID, fromNode, toNode uuid.UUID, edgeType string) error {
	et := graph.EdgeType(edgeType)
	if !et.Valid() {
		et = graph.EdgeTypeSeq
	}
	return s.graph.CreateEdge(storyID, fromNode, toNode, et)
}

func (s *memoryService) List(ctx context.Context, storyID uuid.UUID) ([]graph.Edge, error) {
	return s.graph.ListEdges(storyID)
}

type dbService struct {
	q *db.Queries
}

func NewDBService(q *db.Queries) *dbService {
	return &dbService{q: q}
}

func (s *dbService) Create(ctx context.Context, storyID, fromNode, toNode uuid.UUID, edgeType string) error {
	return s.q.CreateSceneEdge(ctx, db.CreateSceneEdgeParams{
		StoryID:   db.ToUUID(storyID),
		FromScene: db.ToUUID(fromNode),
		ToScene:   db.ToUUID(toNode),
		EdgeType:  edgeType,
	})
}

func (s *dbService) List(ctx context.Context, storyID uuid.UUID) ([]graph.Edge, error) {
	edges, err := s.q.ListSceneEdges(ctx, db.ToUUID(storyID))
	if err != nil {
		return nil, err
	}
	result := make([]graph.Edge, len(edges))
	for i, e := range edges {
		result[i] = graph.Edge{
			StoryID:   db.FromUUID(e.StoryID),
			FromNode:  db.FromUUID(e.FromScene),
			ToNode:    db.FromUUID(e.ToScene),
			EdgeType:  graph.EdgeType(e.EdgeType),
			Condition: e.Condition,
		}
	}
	return result, nil
}
