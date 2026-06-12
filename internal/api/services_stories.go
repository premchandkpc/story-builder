package api

import (
	"context"
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

func (s *graphStoryService) Create(ctx context.Context, title string) (*graph.Story, error) { return s.graph.CreateStory(title) }
func (s *graphStoryService) Get(ctx context.Context, id uuid.UUID) (*graph.Story, error)   { return s.graph.GetStory(id) }
func (s *graphStoryService) List(ctx context.Context) ([]graph.Story, error)              { return s.graph.ListStories() }

func (s *graphStoryService) CreateEdge(ctx context.Context, storyID, fromNode, toNode uuid.UUID, edgeType string) error {
	et := graph.EdgeType(edgeType)
	if !et.Valid() {
		et = graph.EdgeTypeSeq
	}
	return s.graph.CreateEdge(storyID, fromNode, toNode, et)
}

func (s *graphStoryService) ListEdges(ctx context.Context, storyID uuid.UUID) ([]graph.Edge, error) {
	return s.graph.ListEdges(storyID)
}

func (s *graphStoryService) GetNode(ctx context.Context, id uuid.UUID) (*graph.Node, error) {
	return s.graph.GetNode(id)
}

func (s *graphStoryService) ListNodes(ctx context.Context, storyID uuid.UUID) ([]graph.Node, error) {
	return s.graph.ListNodes(storyID)
}

func (s *graphStoryService) TopologicalSort(ctx context.Context, storyID uuid.UUID) ([]graph.Node, error) {
	return s.graph.TopologicalSort(storyID)
}

type graphNodeService struct {
	graph *graph.MemoryStore
}

func NewGraphNodeService(gs *graph.MemoryStore) *graphNodeService {
	return &graphNodeService{graph: gs}
}

func (s *graphNodeService) Create(ctx context.Context, storyID uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int) (*graph.Node, error) {
	return s.graph.CreateNode(storyID, beatIntent, characterRefs, locationRef, pov, tone, targetWords)
}

func (s *graphNodeService) Get(ctx context.Context, id uuid.UUID) (*graph.Node, error) {
	return s.graph.GetNode(id)
}

func (s *graphNodeService) Update(ctx context.Context, id uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int, sceneStructure *graph.SceneStructure) (*graph.Node, error) {
	return s.graph.UpdateNode(id, beatIntent, characterRefs, locationRef, pov, tone, targetWords, sceneStructure)
}

func (s *graphNodeService) SetSceneStructure(ctx context.Context, id uuid.UUID, ss graph.SceneStructure) error {
	return s.graph.SetSceneStructure(id, ss)
}

func (s *graphNodeService) List(ctx context.Context, storyID uuid.UUID) ([]graph.Node, error) {
	return s.graph.ListNodes(storyID)
}

type generationService struct {
	gens []compiler.Generation
}

func NewGenerationService() *generationService {
	return &generationService{}
}

func (s *generationService) Generate(ctx context.Context, nodeID uuid.UUID) (*compiler.Generation, error) {
	return nil, fmt.Errorf("generation requires LLM integration -- not implemented in memory mode")
}

func (s *generationService) AcceptGeneration(ctx context.Context, nodeID, genID uuid.UUID) error {
	for i := range s.gens {
		if s.gens[i].ID == genID.String() && s.gens[i].NodeID == nodeID.String() {
			s.gens[i].Accepted = true
			return nil
		}
	}
	return fmt.Errorf("generation %s not found for node %s", genID, nodeID)
}

func (s *generationService) ListGenerations(ctx context.Context, nodeID uuid.UUID) ([]compiler.Generation, error) {
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

func (s *memoryStoryGeneratorService) GenerateStory(ctx context.Context, synopsis string) (*StoryGenerateResult, error) {
	return nil, fmt.Errorf("story generation requires LLM integration -- not implemented in memory mode")
}
