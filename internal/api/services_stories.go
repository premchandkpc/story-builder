package api

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/compiler"
	"github.com/premchand/story-builder/internal/graph"
)

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

func (s *graphNodeService) Update(id uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int, sceneStructure *graph.SceneStructure) (*graph.Node, error) {
	return s.graph.UpdateNode(id, beatIntent, characterRefs, locationRef, pov, tone, targetWords, sceneStructure)
}

func (s *graphNodeService) SetSceneStructure(id uuid.UUID, ss graph.SceneStructure) error {
	return s.graph.SetSceneStructure(id, ss)
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
	return nil, fmt.Errorf("generation requires LLM integration -- not implemented in memory mode")
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

type memoryStoryGeneratorService struct{}

func NewMemoryStoryGeneratorService() *memoryStoryGeneratorService {
	return &memoryStoryGeneratorService{}
}

func (s *memoryStoryGeneratorService) GenerateStory(synopsis string) (*StoryGenerateResult, error) {
	return nil, fmt.Errorf("story generation requires LLM integration -- not implemented in memory mode")
}
