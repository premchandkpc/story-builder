package edge

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
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
	return s.q.CreateEdge(ctx, db.CreateEdgeParams{
		StoryID:  toUUID(storyID),
		FromNode: toUUID(fromNode),
		ToNode:   toUUID(toNode),
		EdgeType: edgeType,
	})
}

func (s *dbService) List(ctx context.Context, storyID uuid.UUID) ([]graph.Edge, error) {
	edges, err := s.q.ListEdges(ctx, toUUID(storyID))
	if err != nil {
		return nil, err
	}
	result := make([]graph.Edge, len(edges))
	for i, e := range edges {
		result[i] = graph.Edge{
			StoryID:  fromUUID(e.StoryID),
			FromNode: fromUUID(e.FromNode),
			ToNode:   fromUUID(e.ToNode),
			EdgeType: graph.EdgeType(e.EdgeType),
		}
	}
	return result, nil
}

func toUUID(id uuid.UUID) pgtype.UUID { return pgtype.UUID{Bytes: id, Valid: true} }
func fromUUID(id pgtype.UUID) uuid.UUID { return id.Bytes }
