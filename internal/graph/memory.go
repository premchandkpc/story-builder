package graph

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MemoryStore struct {
	mu      sync.RWMutex
	stories []Story
	nodes   []Node
	edges   []Edge
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (s *MemoryStore) CreateStory(title string) (*Story, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	story := Story{
		ID:        uuid.New(),
		Title:     title,
		CanonPins: make(map[string]interface{}),
		CreatedAt: time.Now(),
	}
	s.stories = append(s.stories, story)
	return &story, nil
}

func (s *MemoryStore) GetStory(id uuid.UUID) (*Story, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.stories {
		if s.stories[i].ID == id {
			return &s.stories[i], nil
		}
	}
	return nil, fmt.Errorf("story %s not found", id)
}

func (s *MemoryStore) ListStories() ([]Story, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r := make([]Story, len(s.stories))
	copy(r, s.stories)
	return r, nil
}

func (s *MemoryStore) CreateNode(storyID uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int) (*Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := Node{
		ID:            uuid.New(),
		StoryID:       storyID,
		BeatIntent:    beatIntent,
		CharacterRefs: characterRefs,
		LocationRef:   locationRef,
		POV:           pov,
		Tone:          tone,
		TargetWords:   targetWords,
		Status:        NodeStatusDraft,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	s.nodes = append(s.nodes, n)
	return &n, nil
}

func (s *MemoryStore) GetNode(id uuid.UUID) (*Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.nodes {
		if s.nodes[i].ID == id {
			return &s.nodes[i], nil
		}
	}
	return nil, fmt.Errorf("node %s not found", id)
}

func (s *MemoryStore) UpdateNode(id uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int, sceneStructure *SceneStructure) (*Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.nodes {
		if s.nodes[i].ID == id {
			s.nodes[i].BeatIntent = beatIntent
			s.nodes[i].CharacterRefs = characterRefs
			s.nodes[i].LocationRef = locationRef
			s.nodes[i].POV = pov
			s.nodes[i].Tone = tone
			s.nodes[i].TargetWords = targetWords
			if sceneStructure != nil {
				s.nodes[i].SceneStructure = sceneStructure
			}
			s.nodes[i].UpdatedAt = time.Now()
			return &s.nodes[i], nil
		}
	}
	return nil, fmt.Errorf("node %s not found", id)
}

func (s *MemoryStore) SetSceneStructure(id uuid.UUID, ss SceneStructure) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.nodes {
		if s.nodes[i].ID == id {
			s.nodes[i].SceneStructure = &ss
			s.nodes[i].UpdatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("node %s not found", id)
}

func (s *MemoryStore) SetNodeStatus(id uuid.UUID, status NodeStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.nodes {
		if s.nodes[i].ID == id {
			s.nodes[i].Status = status
			s.nodes[i].UpdatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("node %s not found", id)
}

func (s *MemoryStore) ListNodes(storyID uuid.UUID) ([]Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []Node
	for _, n := range s.nodes {
		if n.StoryID == storyID {
			result = append(result, n)
		}
	}
	return result, nil
}

func (s *MemoryStore) CreateEdge(storyID, fromNode, toNode uuid.UUID, edgeType EdgeType) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := Edge{
		StoryID:  storyID,
		FromNode: fromNode,
		ToNode:   toNode,
		EdgeType: edgeType,
	}
	s.edges = append(s.edges, e)
	return nil
}

func (s *MemoryStore) ListEdges(storyID uuid.UUID) ([]Edge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []Edge
	for _, e := range s.edges {
		if e.StoryID == storyID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (s *MemoryStore) GetOutgoingEdges(nodeID uuid.UUID) ([]Edge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []Edge
	for _, e := range s.edges {
		if e.FromNode == nodeID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (s *MemoryStore) GetIncomingEdges(nodeID uuid.UUID) ([]Edge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []Edge
	for _, e := range s.edges {
		if e.ToNode == nodeID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (s *MemoryStore) TopologicalSort(storyID uuid.UUID) ([]Node, error) {
	nodes, err := s.ListNodes(storyID)
	if err != nil {
		return nil, err
	}
	edges, err := s.ListEdges(storyID)
	if err != nil {
		return nil, err
	}
	return TopologicalSort(nodes, edges)
}

func (s *MemoryStore) Predecessors(nodeID uuid.UUID) ([]Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var preds []Node
	for _, e := range s.edges {
		if e.ToNode == nodeID {
			for _, n := range s.nodes {
				if n.ID == e.FromNode {
					preds = append(preds, n)
				}
			}
		}
	}
	return preds, nil
}

func (s *MemoryStore) IsForkJoin(storyID uuid.UUID) ([]Edge, error) {
	edges, err := s.ListEdges(storyID)
	if err != nil {
		return nil, err
	}
	var result []Edge
	for _, e := range edges {
		if e.EdgeType == EdgeTypeFork || e.EdgeType == EdgeTypeJoin {
			result = append(result, e)
		}
	}
	return result, nil
}

func (s *MemoryStore) BranchNodes(storyID uuid.UUID, forkNode uuid.UUID) ([]Node, error) {
	nodes, err := s.ListNodes(storyID)
	if err != nil {
		return nil, err
	}
	edges, err := s.ListEdges(storyID)
	if err != nil {
		return nil, err
	}
	adj := make(map[uuid.UUID][]Edge)
	for _, e := range edges {
		adj[e.FromNode] = append(adj[e.FromNode], e)
	}
	nodeMap := make(map[uuid.UUID]Node)
	for _, n := range nodes {
		nodeMap[n.ID] = n
	}

	var result []Node
	var walk func(id uuid.UUID)
	walked := make(map[uuid.UUID]bool)
	walk = func(id uuid.UUID) {
		if walked[id] {
			return
		}
		walked[id] = true
		if n, ok := nodeMap[id]; ok {
			result = append(result, n)
		}
		outEdges := adj[id]
		for _, e := range outEdges {
			if e.EdgeType == EdgeTypeJoin {
				if n, ok := nodeMap[e.ToNode]; ok {
					result = append(result, n)
				}
				return
			}
			walk(e.ToNode)
		}
	}
	outEdges := adj[forkNode]
	for _, e := range outEdges {
		if e.EdgeType == EdgeTypeFork || e.EdgeType == EdgeTypeChoice {
			walk(e.ToNode)
		}
	}
	return result, nil
}

func (s *MemoryStore) ForkCharacterSets(storyID uuid.UUID, forkNode uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	nodes, err := s.ListNodes(storyID)
	if err != nil {
		return nil, err
	}
	edges, err := s.ListEdges(storyID)
	if err != nil {
		return nil, err
	}
	return BranchCharacterSets(nodes, edges, forkNode)
}
