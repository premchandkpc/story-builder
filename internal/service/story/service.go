package story

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/premchand/story-builder/internal/db"
	"github.com/premchand/story-builder/internal/graph"
)

type StoryService interface {
	Create(ctx context.Context, title string) (*graph.Story, error)
	Get(ctx context.Context, id uuid.UUID) (*graph.Story, error)
	List(ctx context.Context) ([]graph.Story, error)
	CreateEdge(ctx context.Context, storyID, fromNode, toNode uuid.UUID, edgeType string) error
	ListEdges(ctx context.Context, storyID uuid.UUID) ([]graph.Edge, error)
	GetNode(ctx context.Context, id uuid.UUID) (*graph.Node, error)
	ListNodes(ctx context.Context, storyID uuid.UUID) ([]graph.Node, error)
	TopologicalSort(ctx context.Context, storyID uuid.UUID) ([]graph.Node, error)
}

type memoryService struct {
	graph *graph.MemoryStore
}

func NewMemoryService(gs *graph.MemoryStore) *memoryService {
	return &memoryService{graph: gs}
}

func (s *memoryService) Create(ctx context.Context, title string) (*graph.Story, error) {
	return s.graph.CreateStory(title)
}

func (s *memoryService) Get(ctx context.Context, id uuid.UUID) (*graph.Story, error) {
	return s.graph.GetStory(id)
}

func (s *memoryService) List(ctx context.Context) ([]graph.Story, error) {
	return s.graph.ListStories()
}

func (s *memoryService) CreateEdge(ctx context.Context, storyID, fromNode, toNode uuid.UUID, edgeType string) error {
	et := graph.EdgeType(edgeType)
	if !et.Valid() {
		et = graph.EdgeTypeSeq
	}
	return s.graph.CreateEdge(storyID, fromNode, toNode, et)
}

func (s *memoryService) ListEdges(ctx context.Context, storyID uuid.UUID) ([]graph.Edge, error) {
	return s.graph.ListEdges(storyID)
}

func (s *memoryService) GetNode(ctx context.Context, id uuid.UUID) (*graph.Node, error) {
	return s.graph.GetNode(id)
}

func (s *memoryService) ListNodes(ctx context.Context, storyID uuid.UUID) ([]graph.Node, error) {
	return s.graph.ListNodes(storyID)
}

func (s *memoryService) TopologicalSort(ctx context.Context, storyID uuid.UUID) ([]graph.Node, error) {
	return s.graph.TopologicalSort(storyID)
}

type dbService struct {
	q *db.Queries
}

func NewDBService(q *db.Queries) *dbService {
	return &dbService{q: q}
}

func (s *dbService) Create(ctx context.Context, title string) (*graph.Story, error) {
	st, err := s.q.CreateStory(ctx, title)
	if err != nil {
		return nil, err
	}
	return &graph.Story{
		ID:        fromUUID(st.ID),
		Title:     st.Title,
		CanonPins: make(map[string]interface{}),
		CreatedAt: st.CreatedAt.Time,
	}, nil
}

func (s *dbService) Get(ctx context.Context, id uuid.UUID) (*graph.Story, error) {
	st, err := s.q.GetStory(ctx, toUUID(id))
	if err != nil {
		return nil, err
	}
	var pins map[string]interface{}
	json.Unmarshal(st.CanonPins, &pins)
	if pins == nil {
		pins = make(map[string]interface{})
	}
	return &graph.Story{
		ID:        fromUUID(st.ID),
		Title:     st.Title,
		CanonPins: pins,
		CreatedAt: st.CreatedAt.Time,
	}, nil
}

func (s *dbService) List(ctx context.Context) ([]graph.Story, error) {
	stories, err := s.q.ListStories(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]graph.Story, len(stories))
	for i, st := range stories {
		result[i] = graph.Story{
			ID:        fromUUID(st.ID),
			Title:     st.Title,
			CanonPins: make(map[string]interface{}),
			CreatedAt: st.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *dbService) CreateEdge(ctx context.Context, storyID, fromNode, toNode uuid.UUID, edgeType string) error {
	return s.q.CreateEdge(ctx, db.CreateEdgeParams{
		StoryID:  toUUID(storyID),
		FromNode: toUUID(fromNode),
		ToNode:   toUUID(toNode),
		EdgeType: edgeType,
	})
}

func (s *dbService) ListEdges(ctx context.Context, storyID uuid.UUID) ([]graph.Edge, error) {
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

func (s *dbService) GetNode(ctx context.Context, id uuid.UUID) (*graph.Node, error) {
	return getNode(ctx, s.q, id)
}

func (s *dbService) ListNodes(ctx context.Context, storyID uuid.UUID) ([]graph.Node, error) {
	return listNodes(ctx, s.q, storyID)
}

func (s *dbService) TopologicalSort(ctx context.Context, storyID uuid.UUID) ([]graph.Node, error) {
	nodes, err := s.ListNodes(ctx, storyID)
	if err != nil {
		return nil, err
	}
	edges, err := s.ListEdges(ctx, storyID)
	if err != nil {
		return nil, err
	}
	return graph.TopologicalSort(nodes, edges)
}

func toDomainNode(n db.Node) *graph.Node {
	refs := make([]uuid.UUID, len(n.CharacterRefs))
	for i, r := range n.CharacterRefs {
		refs[i] = fromUUID(r)
	}
	var locRef *uuid.UUID
	if n.LocationRef.Valid {
		l := fromUUID(n.LocationRef)
		locRef = &l
	}
	var ss *graph.SceneStructure
	if len(n.SceneStructure) > 0 {
		var s graph.SceneStructure
		if json.Unmarshal(n.SceneStructure, &s) == nil {
			ss = &s
		}
	}
	return &graph.Node{
		ID:             fromUUID(n.ID),
		StoryID:        fromUUID(n.StoryID),
		BeatIntent:     n.BeatIntent,
		CharacterRefs:  refs,
		LocationRef:    locRef,
		POV:            n.Pov,
		Tone:           n.Tone,
		TargetWords:    int(n.TargetWords),
		Status:         graph.NodeStatus(n.Status),
		SceneStructure: ss,
		CreatedAt:      n.CreatedAt.Time,
		UpdatedAt:      n.UpdatedAt.Time,
	}
}

func getNode(ctx context.Context, q *db.Queries, id uuid.UUID) (*graph.Node, error) {
	n, err := q.GetNode(ctx, toUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("node %s not found", id)
		}
		return nil, err
	}
	return toDomainNode(n), nil
}

func listNodes(ctx context.Context, q *db.Queries, storyID uuid.UUID) ([]graph.Node, error) {
	nodes, err := q.ListNodes(ctx, toUUID(storyID))
	if err != nil {
		return nil, err
	}
	result := make([]graph.Node, len(nodes))
	for i, n := range nodes {
		result[i] = *toDomainNode(n)
	}
	return result, nil
}

func toUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func fromUUID(id pgtype.UUID) uuid.UUID {
	return id.Bytes
}

func jsonBytes(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
