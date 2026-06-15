package node

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

type Service interface {
	Create(ctx context.Context, storyID uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int) (*graph.Node, error)
	Get(ctx context.Context, id uuid.UUID) (*graph.Node, error)
	Update(ctx context.Context, id uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int, sceneStructure *graph.SceneStructure) (*graph.Node, error)
	SetSceneStructure(ctx context.Context, id uuid.UUID, ss graph.SceneStructure) error
	List(ctx context.Context, storyID uuid.UUID) ([]graph.Node, error)
}

type memoryService struct {
	g *graph.MemoryStore
}

func NewMemoryService(gs *graph.MemoryStore) *memoryService {
	return &memoryService{g: gs}
}

func (s *memoryService) Create(ctx context.Context, storyID uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int) (*graph.Node, error) {
	return s.g.CreateNode(storyID, beatIntent, characterRefs, locationRef, pov, tone, targetWords)
}

func (s *memoryService) Get(ctx context.Context, id uuid.UUID) (*graph.Node, error) {
	return s.g.GetNode(id)
}

func (s *memoryService) Update(ctx context.Context, id uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int, sceneStructure *graph.SceneStructure) (*graph.Node, error) {
	return s.g.UpdateNode(id, beatIntent, characterRefs, locationRef, pov, tone, targetWords, sceneStructure)
}

func (s *memoryService) SetSceneStructure(ctx context.Context, id uuid.UUID, ss graph.SceneStructure) error {
	return s.g.SetSceneStructure(id, ss)
}

func (s *memoryService) List(ctx context.Context, storyID uuid.UUID) ([]graph.Node, error) {
	return s.g.ListNodes(storyID)
}

type dbService struct {
	q *db.Queries
}

func NewDBService(q *db.Queries) *dbService {
	return &dbService{q: q}
}

func (s *dbService) Create(ctx context.Context, storyID uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int) (*graph.Node, error) {
	refs := make([]pgtype.UUID, len(characterRefs))
	for i, r := range characterRefs {
		refs[i] = db.ToUUID(r)
	}
	var locRef pgtype.UUID
	if locationRef != nil {
		locRef = db.ToUUID(*locationRef)
	}

	// Find the first chapter for this story
	chapters, err := s.q.ListChapters(ctx, db.ToUUID(storyID))
	if err != nil || len(chapters) == 0 {
		return nil, fmt.Errorf("no chapters found for story %s", storyID)
	}

	n, err := s.q.CreateScene(ctx, db.CreateSceneParams{
		ChapterID:     chapters[0].ID,
		StoryID:       db.ToUUID(storyID),
		BeatIntent:    beatIntent,
		CharacterRefs: refs,
		LocationRef:   locRef,
		Pov:           pov,
		Tone:          tone,
		TargetWords:   int32(targetWords),
	})
	if err != nil {
		return nil, err
	}
	return toDomainNode(n), nil
}

func (s *dbService) Get(ctx context.Context, id uuid.UUID) (*graph.Node, error) {
	return getNode(ctx, s.q, id)
}

func (s *dbService) Update(ctx context.Context, id uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int, sceneStructure *graph.SceneStructure) (*graph.Node, error) {
	refs := make([]pgtype.UUID, len(characterRefs))
	for i, r := range characterRefs {
		refs[i] = db.ToUUID(r)
	}
	var locRef pgtype.UUID
	if locationRef != nil {
		locRef = db.ToUUID(*locationRef)
	}
	ssBytes := db.JSONBytes(graph.SceneStructure{FlowType: graph.FlowMonologue, SituationFlow: ""})
	if sceneStructure != nil {
		ssBytes = db.JSONBytes(sceneStructure)
	}
	n, err := s.q.UpdateScene(ctx, db.UpdateSceneParams{
		ID:               db.ToUUID(id),
		BeatIntent:       beatIntent,
		CharacterRefs:    refs,
		LocationRef:      locRef,
		Pov:              pov,
		Tone:             tone,
		TargetWords:      int32(targetWords),
		SceneStructure:   ssBytes,
		Title:            "",
		TimelinePosition: "",
		FlowType:         string(graph.FlowMonologue),
		MaxTurns:         5,
	})
	if err != nil {
		return nil, err
	}
	return toDomainNode(n), nil
}

func (s *dbService) SetSceneStructure(ctx context.Context, id uuid.UUID, ss graph.SceneStructure) error {
	return s.q.UpdateSceneStructure(ctx, db.UpdateSceneStructureParams{
		ID:             db.ToUUID(id),
		SceneStructure: db.JSONBytes(ss),
	})
}

func (s *dbService) List(ctx context.Context, storyID uuid.UUID) ([]graph.Node, error) {
	return listNodes(ctx, s.q, storyID)
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
