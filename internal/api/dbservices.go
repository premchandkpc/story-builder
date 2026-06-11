package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pgvector/pgvector-go"
	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/compiler"
	"github.com/premchand/story-builder/internal/db"
	"github.com/premchand/story-builder/internal/graph"
)

func NewDBCharService(q *db.Queries) *dbCharService {
	return &dbCharService{q: q}
}

type dbCharService struct{ q *db.Queries }

func toUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func fromUUID(u pgtype.UUID) uuid.UUID {
	id, _ := uuid.FromBytes(u.Bytes[:])
	return id
}

func jsonBytes(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

func (s *dbCharService) Create(name string, traits, voiceSamples []string, relationships map[string]string) (*canon.Character, error) {
	c, err := s.q.CreateCharacter(context.Background(), db.CreateCharacterParams{
		Name:          name,
		Traits:        jsonBytes(traits),
		VoiceSamples:  voiceSamples,
		Relationships: jsonBytes(relationships),
	})
	if err != nil {
		return nil, err
	}
	return toDomainChar(c), nil
}

func (s *dbCharService) Get(id uuid.UUID, version int) (*canon.Character, error) {
	if version > 0 {
		c, err := s.q.GetCharacterAtVersion(context.Background(), db.GetCharacterAtVersionParams{
			ID:      toUUID(id),
			Version: int32(version),
		})
		if err != nil {
			return nil, err
		}
		return toDomainChar(c), nil
	}
	c, err := s.q.GetCharacterLatest(context.Background(), toUUID(id))
	if err != nil {
		return nil, err
	}
	return toDomainCharFromLatest(c), nil
}

func (s *dbCharService) Update(id uuid.UUID, traits, voiceSamples []string, relationships map[string]string) (*canon.Character, error) {
	c, err := s.q.UpdateCharacter(context.Background(), db.UpdateCharacterParams{
		ID:           toUUID(id),
		Column2:      jsonBytes(traits),
		VoiceSamples: voiceSamples,
		Column4:      jsonBytes(relationships),
	})
	if err != nil {
		return nil, err
	}
	return toDomainChar(c), nil
}

func (s *dbCharService) List() ([]canon.Character, error) {
	chars, err := s.q.ListCharacters(context.Background())
	if err != nil {
		return nil, err
	}
	result := make([]canon.Character, len(chars))
	for i, c := range chars {
		result[i] = *toDomainCharFromLatest(c)
	}
	return result, nil
}

func toDomainChar(c db.Character) *canon.Character {
	var traits []string
	json.Unmarshal(c.Traits, &traits)
	var rel map[string]string
	json.Unmarshal(c.Relationships, &rel)
	return &canon.Character{
		ID:            fromUUID(c.ID),
		Version:       int(c.Version),
		Name:          c.Name,
		Traits:        traits,
		VoiceSamples:  c.VoiceSamples,
		Relationships: rel,
		CreatedAt:     c.CreatedAt.Time,
	}
}

func toDomainCharFromLatest(c db.LatestCharacter) *canon.Character {
	var traits []string
	json.Unmarshal(c.Traits, &traits)
	var rel map[string]string
	json.Unmarshal(c.Relationships, &rel)
	return &canon.Character{
		ID:            fromUUID(c.ID),
		Version:       int(c.Version),
		Name:          c.Name,
		Traits:        traits,
		VoiceSamples:  c.VoiceSamples,
		Relationships: rel,
		CreatedAt:     c.CreatedAt.Time,
	}
}

type dbLocService struct{ q *db.Queries }

func NewDBLocService(q *db.Queries) *dbLocService {
	return &dbLocService{q: q}
}

func (s *dbLocService) Create(name, description string, props []string) (*canon.Location, error) {
	l, err := s.q.CreateLocation(context.Background(), db.CreateLocationParams{
		Name:        name,
		Description: description,
		Props:       jsonBytes(props),
	})
	if err != nil {
		return nil, err
	}
	return toDomainLoc(l), nil
}

func (s *dbLocService) Get(id uuid.UUID, version int) (*canon.Location, error) {
	if version > 0 {
		l, err := s.q.GetLocationAtVersion(context.Background(), db.GetLocationAtVersionParams{
			ID:      toUUID(id),
			Version: int32(version),
		})
		if err != nil {
			return nil, err
		}
		return toDomainLoc(l), nil
	}
	l, err := s.q.GetLocationLatest(context.Background(), toUUID(id))
	if err != nil {
		return nil, err
	}
	return toDomainLocFromLatest(l), nil
}

func (s *dbLocService) Update(id uuid.UUID, description string, props []string) (*canon.Location, error) {
	l, err := s.q.UpdateLocation(context.Background(), db.UpdateLocationParams{
		ID:          toUUID(id),
		Description: description,
		Column3:     jsonBytes(props),
	})
	if err != nil {
		return nil, err
	}
	return toDomainLoc(l), nil
}

func (s *dbLocService) List() ([]canon.Location, error) {
	locs, err := s.q.ListLocations(context.Background())
	if err != nil {
		return nil, err
	}
	result := make([]canon.Location, len(locs))
	for i, l := range locs {
		result[i] = *toDomainLocFromLatest(l)
	}
	return result, nil
}

func toDomainLoc(l db.Location) *canon.Location {
	var props []string
	json.Unmarshal(l.Props, &props)
	return &canon.Location{
		ID:          fromUUID(l.ID),
		Version:     int(l.Version),
		Name:        l.Name,
		Description: l.Description,
		Props:       props,
		CreatedAt:   l.CreatedAt.Time,
	}
}

func toDomainLocFromLatest(l db.LatestLocation) *canon.Location {
	var props []string
	json.Unmarshal(l.Props, &props)
	return &canon.Location{
		ID:          fromUUID(l.ID),
		Version:     int(l.Version),
		Name:        l.Name,
		Description: l.Description,
		Props:       props,
		CreatedAt:   l.CreatedAt.Time,
	}
}

type dbLoreService struct{ q *db.Queries }

func NewDBLoreService(q *db.Queries) *dbLoreService {
	return &dbLoreService{q: q}
}

func (s *dbLoreService) Create(tags []string, content string) (*canon.Lore, error) {
	l, err := s.q.CreateLore(context.Background(), db.CreateLoreParams{
		Tags:    tags,
		Content: content,
	})
	if err != nil {
		return nil, err
	}
	return &canon.Lore{
		ID:        fromUUID(l.ID),
		Tags:      l.Tags,
		Content:   l.Content,
		CreatedAt: l.CreatedAt.Time,
	}, nil
}

func (s *dbLoreService) List() ([]canon.Lore, error) {
	items, err := s.q.ListLore(context.Background())
	if err != nil {
		return nil, err
	}
	result := make([]canon.Lore, len(items))
	for i, l := range items {
		result[i] = canon.Lore{
			ID:        fromUUID(l.ID),
			Tags:      l.Tags,
			Content:   l.Content,
			CreatedAt: l.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *dbLoreService) SearchByTags(tags []string) ([]canon.Lore, error) {
	items, err := s.q.SearchLoreByTags(context.Background(), tags)
	if err != nil {
		return nil, err
	}
	result := make([]canon.Lore, len(items))
	for i, l := range items {
		result[i] = canon.Lore{
			ID:        fromUUID(l.ID),
			Tags:      l.Tags,
			Content:   l.Content,
			CreatedAt: l.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *dbLoreService) SearchSimilar(embedding []float32, limit int) ([]canon.Lore, error) {
	vec := pgvector.NewVector(embedding)
	items, err := s.q.SearchLoreSimilar(context.Background(), db.SearchLoreSimilarParams{
		Column1: vec,
		Limit:   int32(limit),
	})
	if err != nil {
		return nil, err
	}
	result := make([]canon.Lore, len(items))
	for i, l := range items {
		result[i] = canon.Lore{
			ID:        fromUUID(l.ID),
			Tags:      l.Tags,
			Content:   l.Content,
			CreatedAt: l.CreatedAt.Time,
		}
	}
	return result, nil
}

type dbGraphStoryService struct{ q *db.Queries }

func NewDBGraphStoryService(q *db.Queries) *dbGraphStoryService {
	return &dbGraphStoryService{q: q}
}

func (s *dbGraphStoryService) Create(title string) (*graph.Story, error) {
	st, err := s.q.CreateStory(context.Background(), title)
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

func (s *dbGraphStoryService) Get(id uuid.UUID) (*graph.Story, error) {
	st, err := s.q.GetStory(context.Background(), toUUID(id))
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

func (s *dbGraphStoryService) List() ([]graph.Story, error) {
	stories, err := s.q.ListStories(context.Background())
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

func (s *dbGraphStoryService) CreateEdge(storyID, fromNode, toNode uuid.UUID, edgeType string) error {
	return s.q.CreateEdge(context.Background(), db.CreateEdgeParams{
		StoryID:  toUUID(storyID),
		FromNode: toUUID(fromNode),
		ToNode:   toUUID(toNode),
		EdgeType: edgeType,
	})
}

func (s *dbGraphStoryService) ListEdges(storyID uuid.UUID) ([]graph.Edge, error) {
	edges, err := s.q.ListEdges(context.Background(), toUUID(storyID))
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

func (s *dbGraphStoryService) GetNode(id uuid.UUID) (*graph.Node, error) {
	return getNode(s.q, id)
}

func (s *dbGraphStoryService) ListNodes(storyID uuid.UUID) ([]graph.Node, error) {
	return listNodes(s.q, storyID)
}

func (s *dbGraphStoryService) TopologicalSort(storyID uuid.UUID) ([]graph.Node, error) {
	nodes, err := listNodes(s.q, storyID)
	if err != nil {
		return nil, err
	}
	edges, err := s.ListEdges(storyID)
	if err != nil {
		return nil, err
	}
	return graph.TopologicalSort(nodes, edges)
}

type dbGraphNodeService struct{ q *db.Queries }

func NewDBGraphNodeService(q *db.Queries) *dbGraphNodeService {
	return &dbGraphNodeService{q: q}
}

func (s *dbGraphNodeService) Create(storyID uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int) (*graph.Node, error) {
	refs := make([]pgtype.UUID, len(characterRefs))
	for i, r := range characterRefs {
		refs[i] = toUUID(r)
	}
	var locRef pgtype.UUID
	if locationRef != nil {
		locRef = toUUID(*locationRef)
	}
	n, err := s.q.CreateNode(context.Background(), db.CreateNodeParams{
		StoryID:       toUUID(storyID),
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

func (s *dbGraphNodeService) Get(id uuid.UUID) (*graph.Node, error) {
	return getNode(s.q, id)
}

func (s *dbGraphNodeService) Update(id uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int) (*graph.Node, error) {
	refs := make([]pgtype.UUID, len(characterRefs))
	for i, r := range characterRefs {
		refs[i] = toUUID(r)
	}
	var locRef pgtype.UUID
	if locationRef != nil {
		locRef = toUUID(*locationRef)
	}
	n, err := s.q.UpdateNode(context.Background(), db.UpdateNodeParams{
		ID:            toUUID(id),
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

func (s *dbGraphNodeService) List(storyID uuid.UUID) ([]graph.Node, error) {
	return listNodes(s.q, storyID)
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
	return &graph.Node{
		ID:            fromUUID(n.ID),
		StoryID:       fromUUID(n.StoryID),
		BeatIntent:    n.BeatIntent,
		CharacterRefs: refs,
		LocationRef:   locRef,
		POV:           n.Pov,
		Tone:          n.Tone,
		TargetWords:   int(n.TargetWords),
		Status:        graph.NodeStatus(n.Status),
		CreatedAt:     n.CreatedAt.Time,
		UpdatedAt:     n.UpdatedAt.Time,
	}
}

func getNode(q *db.Queries, id uuid.UUID) (*graph.Node, error) {
	n, err := q.GetNode(context.Background(), toUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("node %s not found", id)
		}
		return nil, err
	}
	return toDomainNode(n), nil
}

func listNodes(q *db.Queries, storyID uuid.UUID) ([]graph.Node, error) {
	nodes, err := q.ListNodes(context.Background(), toUUID(storyID))
	if err != nil {
		return nil, err
	}
	result := make([]graph.Node, len(nodes))
	for i, n := range nodes {
		result[i] = *toDomainNode(n)
	}
	return result, nil
}

type dbGenerationService struct{ q *db.Queries }

func NewDBGenerationService(q *db.Queries) *dbGenerationService {
	return &dbGenerationService{q: q}
}

func (s *dbGenerationService) Generate(nodeID uuid.UUID) (*compiler.Generation, error) {
	return nil, fmt.Errorf("generation requires LLM integration — not implemented")
}

func (s *dbGenerationService) AcceptGeneration(nodeID, genID uuid.UUID) error {
	return s.q.AcceptGeneration(context.Background(), toUUID(genID))
}

func (s *dbGenerationService) ListGenerations(nodeID uuid.UUID) ([]compiler.Generation, error) {
	gens, err := s.q.ListGenerationsForNode(context.Background(), toUUID(nodeID))
	if err != nil {
		return nil, err
	}
	result := make([]compiler.Generation, len(gens))
	for i, g := range gens {
		result[i] = compiler.Generation{
			ID:             fromUUID(g.ID).String(),
			NodeID:         fromUUID(g.NodeID).String(),
			ContextHash:    g.ContextHash,
			PromptSnapshot: g.PromptSnapshot,
			Output:         g.Output,
			Model:          g.Model,
			Accepted:       g.Accepted,
		}
	}
	return result, nil
}
