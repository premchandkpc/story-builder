package story

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo  Repository
	now   func() time.Time
	newID func() string
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now, newID: uuid.NewString}
}

func (s *Service) CreateStory(ctx context.Context, title string) (Story, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Story{}, ErrInvalidTitle
	}
	story := Story{ID: s.newID(), Title: title, CreatedAt: s.now().UTC()}
	if err := s.repo.CreateStory(ctx, story); err != nil {
		return Story{}, fmt.Errorf("create story: %w", err)
	}
	return story, nil
}

func (s *Service) ListStories(ctx context.Context) ([]Story, error) {
	stories, err := s.repo.ListStories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list stories: %w", err)
	}
	return stories, nil
}

func (s *Service) Graph(ctx context.Context, storyID string) (Graph, error) {
	story, err := s.repo.GetStory(ctx, storyID)
	if err != nil {
		return Graph{}, fmt.Errorf("get story: %w", err)
	}
	scenes, err := s.repo.ListScenes(ctx, storyID)
	if err != nil {
		return Graph{}, fmt.Errorf("list scenes: %w", err)
	}
	edges, err := s.repo.ListEdges(ctx, storyID)
	if err != nil {
		return Graph{}, fmt.Errorf("list edges: %w", err)
	}
	return Graph{Story: story, Scenes: scenes, Edges: edges}, nil
}

func (s *Service) CreateScene(ctx context.Context, storyID, title, brief string, position Position) (Scene, error) {
	if _, err := s.repo.GetStory(ctx, storyID); err != nil {
		return Scene{}, fmt.Errorf("get story: %w", err)
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return Scene{}, ErrInvalidTitle
	}
	now := s.now().UTC()
	scene := Scene{ID: s.newID(), StoryID: storyID, Title: title, Brief: strings.TrimSpace(brief), Position: position, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.CreateScene(ctx, scene); err != nil {
		return Scene{}, fmt.Errorf("create scene: %w", err)
	}
	return scene, nil
}

func (s *Service) UpdateScene(ctx context.Context, sceneID string, title, brief *string, position *Position) (Scene, error) {
	scene, err := s.repo.GetScene(ctx, sceneID)
	if err != nil {
		return Scene{}, fmt.Errorf("get scene: %w", err)
	}
	if title != nil {
		trimmed := strings.TrimSpace(*title)
		if trimmed == "" {
			return Scene{}, ErrInvalidTitle
		}
		scene.Title = trimmed
	}
	if brief != nil {
		scene.Brief = strings.TrimSpace(*brief)
	}
	if position != nil {
		scene.Position = *position
	}
	scene.UpdatedAt = s.now().UTC()
	if err := s.repo.UpdateScene(ctx, scene); err != nil {
		return Scene{}, fmt.Errorf("update scene: %w", err)
	}
	return scene, nil
}

func (s *Service) CreateEdge(ctx context.Context, storyID, fromID, toID string) (Edge, error) {
	if fromID == "" || toID == "" || fromID == toID {
		return Edge{}, ErrInvalidEdge
	}
	from, err := s.repo.GetScene(ctx, fromID)
	if err != nil {
		return Edge{}, fmt.Errorf("get source scene: %w", err)
	}
	to, err := s.repo.GetScene(ctx, toID)
	if err != nil {
		return Edge{}, fmt.Errorf("get target scene: %w", err)
	}
	if from.StoryID != storyID || to.StoryID != storyID {
		return Edge{}, ErrCrossStoryEdge
	}
	edges, err := s.repo.ListEdges(ctx, storyID)
	if err != nil {
		return Edge{}, fmt.Errorf("list edges: %w", err)
	}
	for _, edge := range edges {
		if edge.FromSceneID == fromID && edge.ToSceneID == toID {
			return Edge{}, ErrDuplicateEdge
		}
	}
	candidate := Edge{ID: s.newID(), StoryID: storyID, FromSceneID: fromID, ToSceneID: toID, CreatedAt: s.now().UTC()}
	if HasCycle(append(edges, candidate)) {
		return Edge{}, ErrCycle
	}
	if err := s.repo.CreateEdge(ctx, candidate); err != nil {
		return Edge{}, fmt.Errorf("create edge: %w", err)
	}
	return candidate, nil
}

func HasCycle(edges []Edge) bool {
	next := make(map[string][]string)
	nodes := make(map[string]struct{})
	for _, edge := range edges {
		next[edge.FromSceneID] = append(next[edge.FromSceneID], edge.ToSceneID)
		nodes[edge.FromSceneID] = struct{}{}
		nodes[edge.ToSceneID] = struct{}{}
	}
	visiting := make(map[string]bool)
	visited := make(map[string]bool)
	var visit func(string) bool
	visit = func(node string) bool {
		if visiting[node] {
			return true
		}
		if visited[node] {
			return false
		}
		visiting[node] = true
		for _, child := range next[node] {
			if visit(child) {
				return true
			}
		}
		visiting[node] = false
		visited[node] = true
		return false
	}
	for node := range nodes {
		if visit(node) {
			return true
		}
	}
	return false
}
