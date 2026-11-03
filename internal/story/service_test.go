package story

import (
	"context"
	"errors"
	"sync"
	"testing"
)

type memoryRepo struct {
	mu      sync.RWMutex
	stories map[string]Story
	scenes  map[string]Scene
	edges   map[string]Edge
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{stories: map[string]Story{}, scenes: map[string]Scene{}, edges: map[string]Edge{}}
}
func (r *memoryRepo) CreateStory(_ context.Context, s Story) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stories[s.ID] = s
	return nil
}
func (r *memoryRepo) GetStory(_ context.Context, id string) (Story, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.stories[id]
	if !ok {
		return Story{}, ErrNotFound
	}
	return s, nil
}
func (r *memoryRepo) ListStories(_ context.Context) ([]Story, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Story, 0, len(r.stories))
	for _, s := range r.stories {
		out = append(out, s)
	}
	return out, nil
}
func (r *memoryRepo) CreateScene(_ context.Context, s Scene) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scenes[s.ID] = s
	return nil
}
func (r *memoryRepo) GetScene(_ context.Context, id string) (Scene, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.scenes[id]
	if !ok {
		return Scene{}, ErrNotFound
	}
	return s, nil
}
func (r *memoryRepo) UpdateScene(_ context.Context, s Scene) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.scenes[s.ID]; !ok {
		return ErrNotFound
	}
	r.scenes[s.ID] = s
	return nil
}
func (r *memoryRepo) ListScenes(_ context.Context, storyID string) ([]Scene, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Scene
	for _, s := range r.scenes {
		if s.StoryID == storyID {
			out = append(out, s)
		}
	}
	return out, nil
}
func (r *memoryRepo) CreateEdge(_ context.Context, e Edge) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.edges[e.ID] = e
	return nil
}
func (r *memoryRepo) ListEdges(_ context.Context, storyID string) ([]Edge, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Edge
	for _, e := range r.edges {
		if e.StoryID == storyID {
			out = append(out, e)
		}
	}
	return out, nil
}

func TestCreateEdgeRejectsCycle(t *testing.T) {
	ctx := context.Background()
	svc := NewService(newMemoryRepo())
	story, _ := svc.CreateStory(ctx, "A map")
	a, _ := svc.CreateScene(ctx, story.ID, "A", "", Position{})
	b, _ := svc.CreateScene(ctx, story.ID, "B", "", Position{})
	c, _ := svc.CreateScene(ctx, story.ID, "C", "", Position{})
	for _, pair := range [][2]string{{a.ID, b.ID}, {b.ID, c.ID}} {
		if _, err := svc.CreateEdge(ctx, story.ID, pair[0], pair[1]); err != nil {
			t.Fatal(err)
		}
	}
	_, err := svc.CreateEdge(ctx, story.ID, c.ID, a.ID)
	if !errors.Is(err, ErrCycle) {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestUpdateScenePersistsPosition(t *testing.T) {
	ctx := context.Background()
	svc := NewService(newMemoryRepo())
	story, _ := svc.CreateStory(ctx, "A map")
	scene, _ := svc.CreateScene(ctx, story.ID, "Opening", "", Position{})
	position := Position{X: 240, Y: 120}
	updated, err := svc.UpdateScene(ctx, scene.ID, nil, nil, &position)
	if err != nil || updated.Position != position {
		t.Fatalf("position was not persisted: %#v, %v", updated.Position, err)
	}
}
