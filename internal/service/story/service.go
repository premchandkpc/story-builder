package story

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/premchand/story-builder/internal/db"
	"github.com/premchand/story-builder/internal/graph"
)

type StoryService interface {
	Create(ctx context.Context, title string) (*graph.Story, error)
	Get(ctx context.Context, id uuid.UUID) (*graph.Story, error)
	Update(ctx context.Context, id uuid.UUID, title string) (*graph.Story, error)
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

func (s *memoryService) Update(ctx context.Context, id uuid.UUID, title string) (*graph.Story, error) {
	st, err := s.graph.GetStory(id)
	if err != nil {
		return nil, err
	}
	st.Title = title
	return st, nil
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
		ID:        db.FromUUID(st.ID),
		Title:     st.Title,
		Genre:     st.Genre,
		Theme:     st.Theme,
		MainPrompt: st.MainPrompt,
		GeneralPrompt: st.GeneralPrompt,
		CanonPins: make(map[string]interface{}),
		CreatedAt: st.CreatedAt.Time,
	}, nil
}

func (s *dbService) Get(ctx context.Context, id uuid.UUID) (*graph.Story, error) {
	st, err := s.q.GetStory(ctx, db.ToUUID(id))
	if err != nil {
		return nil, err
	}
	var pins map[string]interface{}
	json.Unmarshal(st.CanonPins, &pins)
	if pins == nil {
		pins = make(map[string]interface{})
	}
	return &graph.Story{
		ID:        db.FromUUID(st.ID),
		Title:     st.Title,
		Genre:     st.Genre,
		Theme:     st.Theme,
		MainPrompt: st.MainPrompt,
		GeneralPrompt: st.GeneralPrompt,
		CanonPins: pins,
		CreatedAt: st.CreatedAt.Time,
	}, nil
}

func (s *dbService) Update(ctx context.Context, id uuid.UUID, title string) (*graph.Story, error) {
	existing, err := s.q.GetStory(ctx, db.ToUUID(id))
	if err != nil {
		return nil, err
	}
	st, err := s.q.UpdateStory(ctx, db.UpdateStoryParams{
		ID:            db.ToUUID(id),
		Title:         title,
		Genre:         existing.Genre,
		Theme:         existing.Theme,
		MainPrompt:    existing.MainPrompt,
		GeneralPrompt: existing.GeneralPrompt,
	})
	if err != nil {
		return nil, err
	}
	var pins map[string]interface{}
	json.Unmarshal(st.CanonPins, &pins)
	if pins == nil {
		pins = make(map[string]interface{})
	}
	return &graph.Story{
		ID:            db.FromUUID(st.ID),
		Title:         st.Title,
		Genre:         st.Genre,
		Theme:         st.Theme,
		MainPrompt:    st.MainPrompt,
		GeneralPrompt: st.GeneralPrompt,
		CanonPins:     pins,
		CreatedAt:     st.CreatedAt.Time,
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
			ID:        db.FromUUID(st.ID),
			Title:     st.Title,
			CanonPins: make(map[string]interface{}),
			CreatedAt: st.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *dbService) CreateEdge(ctx context.Context, storyID, fromNode, toNode uuid.UUID, edgeType string) error {
	return s.q.CreateSceneEdge(ctx, db.CreateSceneEdgeParams{
		StoryID:   db.ToUUID(storyID),
		FromScene: db.ToUUID(fromNode),
		ToScene:   db.ToUUID(toNode),
		EdgeType:  edgeType,
	})
}

func (s *dbService) ListEdges(ctx context.Context, storyID uuid.UUID) ([]graph.Edge, error) {
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

func toDomainNode(n db.Scene) *graph.Node {
	refs := make([]uuid.UUID, len(n.CharacterRefs))
	for i, r := range n.CharacterRefs {
		refs[i] = db.FromUUID(r)
	}
	var locRef *uuid.UUID
	if n.LocationRef.Valid {
		l := db.FromUUID(n.LocationRef)
		locRef = &l
	}
	var ss *graph.SceneStructure
	if len(n.SceneStructure) > 0 {
		var s graph.SceneStructure
		if json.Unmarshal(n.SceneStructure, &s) == nil {
			ss = &s
		}
	}
	var parentSceneID *uuid.UUID
	if n.ParentSceneID.Valid {
		p := db.FromUUID(n.ParentSceneID)
		parentSceneID = &p
	}
	return &graph.Node{
		ID:               db.FromUUID(n.ID),
		StoryID:          db.FromUUID(n.StoryID),
		ChapterID:        db.FromUUID(n.ChapterID),
		Title:            n.Title,
		BeatIntent:       n.BeatIntent,
		CharacterRefs:    refs,
		LocationRef:      locRef,
		POV:              n.Pov,
		Tone:             n.Tone,
		TargetWords:      int(n.TargetWords),
		Status:           graph.NodeStatus(n.Status),
		SceneStructure:   ss,
		ParentSceneID:    parentSceneID,
		TimelinePosition: n.TimelinePosition,
		FlowType:         graph.FlowType(n.FlowType),
		MaxTurns:         int(n.MaxTurns),
		CreatedAt:        n.CreatedAt.Time,
		UpdatedAt:        n.UpdatedAt.Time,
	}
}

func getNode(ctx context.Context, q *db.Queries, id uuid.UUID) (*graph.Node, error) {
	n, err := q.GetScene(ctx, db.ToUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("node %s not found", id)
		}
		return nil, err
	}
	return toDomainNode(n), nil
}

func listNodes(ctx context.Context, q *db.Queries, storyID uuid.UUID) ([]graph.Node, error) {
	nodes, err := q.ListScenes(ctx, db.ToUUID(storyID))
	if err != nil {
		return nil, err
	}
	result := make([]graph.Node, len(nodes))
	for i, n := range nodes {
		result[i] = *toDomainNode(n)
	}
	return result, nil
}
