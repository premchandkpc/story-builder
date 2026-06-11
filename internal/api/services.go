package api

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/compiler"
	"github.com/premchand/story-builder/internal/graph"
)

type charService struct {
	chars   []canon.Character
	version map[uuid.UUID]int
}

func NewCharService() *charService {
	return &charService{version: make(map[uuid.UUID]int)}
}

func (s *charService) Create(name string, traits, voiceSamples []string, relationships map[string]string) (*canon.Character, error) {
	c := canon.Character{
		ID:            uuid.New(),
		Version:       1,
		Name:          name,
		Traits:        traits,
		VoiceSamples:  voiceSamples,
		Relationships: relationships,
		CreatedAt:     time.Now(),
	}
	s.chars = append(s.chars, c)
	s.version[c.ID] = 2
	return &c, nil
}

func (s *charService) Get(id uuid.UUID, version int) (*canon.Character, error) {
	var latest *canon.Character
	for i := range s.chars {
		if s.chars[i].ID == id {
			if version > 0 && s.chars[i].Version == version {
				return &s.chars[i], nil
			}
			if latest == nil || s.chars[i].Version > latest.Version {
				latest = &s.chars[i]
			}
		}
	}
	if latest == nil {
		return nil, fmt.Errorf("character %s not found", id)
	}
	return latest, nil
}

func (s *charService) Update(id uuid.UUID, traits, voiceSamples []string, relationships map[string]string) (*canon.Character, error) {
	next := s.version[id]
	if next == 0 {
		return nil, fmt.Errorf("character %s not found", id)
	}
	c := canon.Character{
		ID:            id,
		Version:       next,
		Name:          s.chars[0].Name,
		Traits:        traits,
		VoiceSamples:  voiceSamples,
		Relationships: relationships,
		CreatedAt:     time.Now(),
	}
	s.chars = append(s.chars, c)
	s.version[id] = next + 1
	return &c, nil
}

func (s *charService) List() ([]canon.Character, error) {
	latest := make(map[uuid.UUID]canon.Character)
	for _, c := range s.chars {
		if existing, ok := latest[c.ID]; !ok || c.Version > existing.Version {
			latest[c.ID] = c
		}
	}
	result := make([]canon.Character, 0, len(latest))
	for _, c := range latest {
		result = append(result, c)
	}
	return result, nil
}

type locService struct {
	locs    []canon.Location
	version map[uuid.UUID]int
}

func NewLocService() *locService {
	return &locService{version: make(map[uuid.UUID]int)}
}

func (s *locService) Create(name, description string, props []string) (*canon.Location, error) {
	l := canon.Location{
		ID:          uuid.New(),
		Version:     1,
		Name:        name,
		Description: description,
		Props:       props,
		CreatedAt:   time.Now(),
	}
	s.locs = append(s.locs, l)
	s.version[l.ID] = 2
	return &l, nil
}

func (s *locService) Get(id uuid.UUID, version int) (*canon.Location, error) {
	var latest *canon.Location
	for i := range s.locs {
		if s.locs[i].ID == id {
			if version > 0 && s.locs[i].Version == version {
				return &s.locs[i], nil
			}
			if latest == nil || s.locs[i].Version > latest.Version {
				latest = &s.locs[i]
			}
		}
	}
	if latest == nil {
		return nil, fmt.Errorf("location %s not found", id)
	}
	return latest, nil
}

func (s *locService) Update(id uuid.UUID, description string, props []string) (*canon.Location, error) {
	next := s.version[id]
	if next == 0 {
		return nil, fmt.Errorf("location %s not found", id)
	}
	l := canon.Location{
		ID:          id,
		Version:     next,
		Name:        s.locs[0].Name,
		Description: description,
		Props:       props,
		CreatedAt:   time.Now(),
	}
	s.locs = append(s.locs, l)
	s.version[id] = next + 1
	return &l, nil
}

func (s *locService) List() ([]canon.Location, error) {
	latest := make(map[uuid.UUID]canon.Location)
	for _, l := range s.locs {
		if existing, ok := latest[l.ID]; !ok || l.Version > existing.Version {
			latest[l.ID] = l
		}
	}
	result := make([]canon.Location, 0, len(latest))
	for _, l := range latest {
		result = append(result, l)
	}
	return result, nil
}

type loreService struct {
	items []canon.Lore
}

func NewLoreService() *loreService {
	return &loreService{}
}

func (s *loreService) Create(tags []string, content string) (*canon.Lore, error) {
	l := canon.Lore{
		ID:        uuid.New(),
		Tags:      tags,
		Content:   content,
		CreatedAt: time.Now(),
	}
	s.items = append(s.items, l)
	return &l, nil
}

func (s *loreService) List() ([]canon.Lore, error) {
	r := make([]canon.Lore, len(s.items))
	copy(r, s.items)
	return r, nil
}

func (s *loreService) SearchByTags(tags []string) ([]canon.Lore, error) {
	tagSet := make(map[string]bool, len(tags))
	for _, t := range tags {
		tagSet[t] = true
	}
	var result []canon.Lore
	for _, l := range s.items {
		for _, t := range l.Tags {
			if tagSet[t] {
				result = append(result, l)
				break
			}
		}
	}
	return result, nil
}

func (s *loreService) SearchSimilar(embedding []float32, limit int) ([]canon.Lore, error) {
	if limit > len(s.items) {
		limit = len(s.items)
	}
	return s.items[:limit], nil
}

type graphStoryService struct {
	graph *graph.MemoryStore
}

func NewGraphStoryService(gs *graph.MemoryStore) *graphStoryService {
	return &graphStoryService{graph: gs}
}

func (s *graphStoryService) Create(title string) (*graph.Story, error) { return s.graph.CreateStory(title) }
func (s *graphStoryService) Get(id uuid.UUID) (*graph.Story, error)   { return s.graph.GetStory(id) }
func (s *graphStoryService) List() ([]graph.Story, error)              { return s.graph.ListStories() }

func (s *graphStoryService) CreateEdge(storyID, fromNode, toNode uuid.UUID, edgeType string) error {
	et := graph.EdgeType(edgeType)
	if !et.Valid() {
		et = graph.EdgeTypeSeq
	}
	return s.graph.CreateEdge(storyID, fromNode, toNode, et)
}

func (s *graphStoryService) ListEdges(storyID uuid.UUID) ([]graph.Edge, error) {
	return s.graph.ListEdges(storyID)
}

func (s *graphStoryService) GetNode(id uuid.UUID) (*graph.Node, error) {
	return s.graph.GetNode(id)
}

func (s *graphStoryService) ListNodes(storyID uuid.UUID) ([]graph.Node, error) {
	return s.graph.ListNodes(storyID)
}

func (s *graphStoryService) TopologicalSort(storyID uuid.UUID) ([]graph.Node, error) {
	return s.graph.TopologicalSort(storyID)
}

type graphNodeService struct {
	graph *graph.MemoryStore
}

func NewGraphNodeService(gs *graph.MemoryStore) *graphNodeService {
	return &graphNodeService{graph: gs}
}

func (s *graphNodeService) Create(storyID uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int) (*graph.Node, error) {
	return s.graph.CreateNode(storyID, beatIntent, characterRefs, locationRef, pov, tone, targetWords)
}

func (s *graphNodeService) Get(id uuid.UUID) (*graph.Node, error) {
	return s.graph.GetNode(id)
}

func (s *graphNodeService) Update(id uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int) (*graph.Node, error) {
	return s.graph.UpdateNode(id, beatIntent, characterRefs, locationRef, pov, tone, targetWords)
}

func (s *graphNodeService) List(storyID uuid.UUID) ([]graph.Node, error) {
	return s.graph.ListNodes(storyID)
}

type generationService struct {
	gens []compiler.Generation
}

func NewGenerationService() *generationService {
	return &generationService{}
}

func (s *generationService) Generate(nodeID uuid.UUID) (*compiler.Generation, error) {
	return nil, fmt.Errorf("generation requires LLM integration — not implemented in memory mode")
}

func (s *generationService) AcceptGeneration(nodeID, genID uuid.UUID) error {
	for i := range s.gens {
		if s.gens[i].ID == genID.String() && s.gens[i].NodeID == nodeID.String() {
			s.gens[i].Accepted = true
			return nil
		}
	}
	return fmt.Errorf("generation %s not found for node %s", genID, nodeID)
}

func (s *generationService) ListGenerations(nodeID uuid.UUID) ([]compiler.Generation, error) {
	var result []compiler.Generation
	for _, g := range s.gens {
		if g.NodeID == nodeID.String() {
			result = append(result, g)
		}
	}
	return result, nil
}
