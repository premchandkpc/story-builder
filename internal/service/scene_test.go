package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/premchand/story-builder/internal/domain"
)

type mockSceneRepo struct {
	scenes map[string]*domain.Scene
	byStory map[string][]*domain.Scene
	err    error
}

func newMockSceneRepo() *mockSceneRepo {
	return &mockSceneRepo{
		scenes:  make(map[string]*domain.Scene),
		byStory: make(map[string][]*domain.Scene),
	}
}

func (m *mockSceneRepo) Create(ctx context.Context, s *domain.Scene) error {
	if m.err != nil {
		return m.err
	}
	if s.ID == "" {
		s.ID = "scene-id"
	}
	m.scenes[s.ID] = s
	m.byStory[s.StoryID] = append(m.byStory[s.StoryID], s)
	return nil
}

func (m *mockSceneRepo) Get(ctx context.Context, id string) (*domain.Scene, error) {
	if m.err != nil {
		return nil, m.err
	}
	s, ok := m.scenes[id]
	if !ok {
		return nil, nil
	}
	return s, nil
}

func (m *mockSceneRepo) Update(ctx context.Context, s *domain.Scene) error {
	if m.err != nil {
		return m.err
	}
	m.scenes[s.ID] = s
	return nil
}

func (m *mockSceneRepo) ListByStory(ctx context.Context, storyID string) ([]*domain.Scene, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.byStory[storyID], nil
}

func (m *mockSceneRepo) Delete(ctx context.Context, id string) error {
	if m.err != nil {
		return m.err
	}
	delete(m.scenes, id)
	return nil
}

func (m *mockSceneRepo) DeleteByStory(ctx context.Context, storyID string) error {
	return nil
}

type mockEdgeRepo struct {
	edges   map[string]*domain.SceneEdge
	byStory map[string][]*domain.SceneEdge
	err     error
}

func newMockEdgeRepo() *mockEdgeRepo {
	return &mockEdgeRepo{
		edges:   make(map[string]*domain.SceneEdge),
		byStory: make(map[string][]*domain.SceneEdge),
	}
}

func (m *mockEdgeRepo) Create(ctx context.Context, e *domain.SceneEdge) error {
	if m.err != nil {
		return m.err
	}
	if e.ID == "" {
		e.ID = "edge-id"
	}
	m.edges[e.ID] = e
	m.byStory[e.StoryID] = append(m.byStory[e.StoryID], e)
	return nil
}

func (m *mockEdgeRepo) ListByStory(ctx context.Context, storyID string) ([]*domain.SceneEdge, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.byStory[storyID], nil
}

func (m *mockEdgeRepo) ListFrom(ctx context.Context, sceneID string) ([]*domain.SceneEdge, error) {
	return nil, errors.New("not implemented")
}

func (m *mockEdgeRepo) ListTo(ctx context.Context, sceneID string) ([]*domain.SceneEdge, error) {
	return nil, errors.New("not implemented")
}

func (m *mockEdgeRepo) Delete(ctx context.Context, storyID, fromSceneID, toSceneID string) error {
	if m.err != nil {
		return m.err
	}
	for id, e := range m.edges {
		if e.StoryID == storyID && e.FromSceneID == fromSceneID && e.ToSceneID == toSceneID {
			delete(m.edges, id)
			return nil
		}
	}
	return nil
}

func (m *mockEdgeRepo) DeleteByStory(ctx context.Context, storyID string) error {
	return nil
}

func (m *mockEdgeRepo) DeleteByID(ctx context.Context, edgeID string) error {
	return nil
}

type mockGenRepo struct {
	mu    sync.Mutex
	gens  map[string]*domain.Generation
}

func newMockGenRepo() *mockGenRepo {
	return &mockGenRepo{gens: make(map[string]*domain.Generation)}
}

func (m *mockGenRepo) Create(_ context.Context, g *domain.Generation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.gens == nil {
		m.gens = make(map[string]*domain.Generation)
	}
	if g.ID == "" {
		g.ID = "gen-" + g.SceneID
	}
	g.CreatedAt = time.Now()
	m.gens[g.ID] = g
	return nil
}

func (m *mockGenRepo) Get(_ context.Context, id string) (*domain.Generation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.gens == nil {
		return nil, nil
	}
	g, ok := m.gens[id]
	if !ok {
		return nil, nil
	}
	return g, nil
}

func (m *mockGenRepo) Update(_ context.Context, g *domain.Generation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gens[g.ID] = g
	return nil
}

func (m *mockGenRepo) ListByScene(_ context.Context, sceneID string) ([]*domain.Generation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var gens []*domain.Generation
	if m.gens != nil {
		for _, g := range m.gens {
			if g.SceneID == sceneID {
				gens = append(gens, g)
			}
		}
	}
	return gens, nil
}

func (m *mockGenRepo) ListByStory(_ context.Context, storyID string) ([]*domain.Generation, error) {
	return nil, nil
}

func (m *mockGenRepo) DeleteByScene(_ context.Context, sceneID string) error {
	return nil
}

func (m *mockGenRepo) DeleteByStory(_ context.Context, storyID string) error {
	return nil
}

func (m *mockGenRepo) SetStepStatus(_ context.Context, genID, step, status string) error {
	return nil
}

func (m *mockGenRepo) FindByContextHash(_ context.Context, storyID, hash string) (*domain.Generation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, g := range m.gens {
		if g.ContextHash == hash && g.Accepted {
			return g, nil
		}
	}
	return nil, nil
}

func (m *mockGenRepo) SetAccepted(_ context.Context, sceneID, genID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, g := range m.gens {
		g.Accepted = g.ID == genID
	}
	return nil
}

func TestSceneService_Topology(t *testing.T) {
	sceneMock := newMockSceneRepo()
	edgeMock := newMockEdgeRepo()
	svc := NewSceneService(sceneMock, edgeMock, &mockGenRepo{})

	s1 := &domain.Scene{StoryID: "story-1", Title: "Scene 1"}
	s2 := &domain.Scene{StoryID: "story-1", Title: "Scene 2"}
	svc.Create(context.Background(), s1)
	svc.Create(context.Background(), s2)
	e := &domain.SceneEdge{StoryID: "story-1", FromSceneID: s1.ID, ToSceneID: s2.ID}
	edgeMock.Create(context.Background(), e)

	scenes, edges, err := svc.Topology(context.Background(), "story-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scenes) != 2 {
		t.Fatalf("expected 2 scenes, got %d", len(scenes))
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
}

func TestSceneService_Create(t *testing.T) {
	sceneMock := newMockSceneRepo()
	svc := NewSceneService(sceneMock, newMockEdgeRepo(), &mockGenRepo{})

	s, err := svc.Create(context.Background(), &domain.Scene{StoryID: "s1", Title: "New Scene"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Title != "New Scene" {
		t.Fatalf("expected 'New Scene', got %q", s.Title)
	}
}

func TestCharacterService(t *testing.T) {
	charMock := newMockCharacterRepo()
	svc := NewCharacterService(charMock)

	t.Run("create and get", func(t *testing.T) {
		c, err := svc.Create(context.Background(), &domain.Character{StoryID: "s1", Name: "Alice"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.Name != "Alice" {
			t.Fatalf("expected 'Alice', got %q", c.Name)
		}

		got, err := svc.Get(context.Background(), c.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "Alice" {
			t.Fatalf("expected 'Alice', got %q", got.Name)
		}
	})

	t.Run("list by story", func(t *testing.T) {
		svc.Create(context.Background(), &domain.Character{StoryID: "s1", Name: "Bob"})
		chars, err := svc.List(context.Background(), "s1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(chars) != 2 {
			t.Fatalf("expected 2 characters, got %d", len(chars))
		}
	})
}

// ── Mock JobRepository ────────────────────────────────────────────────

type mockJobRepo struct {
	mu   sync.Mutex
	jobs map[string]*domain.Job
}

func newMockJobRepo() *mockJobRepo {
	return &mockJobRepo{jobs: make(map[string]*domain.Job)}
}

func (m *mockJobRepo) Create(_ context.Context, j *domain.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	j.ID = "job-" + j.SceneID
	j.CreatedAt = time.Now()
	m.jobs[j.ID] = j
	return nil
}

func (m *mockJobRepo) Get(_ context.Context, id string) (*domain.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[id]
	if !ok {
		return nil, nil
	}
	return j, nil
}

func (m *mockJobRepo) Update(_ context.Context, j *domain.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[j.ID] = j
	return nil
}

func (m *mockJobRepo) PickPending(_ context.Context, _ string, _ time.Duration) (*domain.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, j := range m.jobs {
		if j.Status == domain.JobStatusPending {
			j.Status = domain.JobStatusRunning
			return j, nil
		}
	}
	return nil, nil
}

func (m *mockJobRepo) ListPending(_ context.Context) ([]*domain.Job, error) {
	return nil, nil
}

func (m *mockJobRepo) ListStuck(_ context.Context, _ time.Duration) ([]*domain.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var stuck []*domain.Job
	for _, j := range m.jobs {
		if j.Status == domain.JobStatusRunning {
			stuck = append(stuck, j)
		}
	}
	return stuck, nil
}

// ── Mock CharacterRepository ──────────────────────────────────────────

type mockCharacterRepo struct {
	chars   map[string]*domain.Character
	byStory map[string][]*domain.Character
	err     error
}

func newMockCharacterRepo() *mockCharacterRepo {
	return &mockCharacterRepo{
		chars:   make(map[string]*domain.Character),
		byStory: make(map[string][]*domain.Character),
	}
}

func (m *mockCharacterRepo) Create(ctx context.Context, c *domain.Character) error {
	if m.err != nil {
		return m.err
	}
	if c.ID == "" {
		c.ID = "char-" + c.Name
	}
	m.chars[c.ID] = c
	m.byStory[c.StoryID] = append(m.byStory[c.StoryID], c)
	return nil
}

func (m *mockCharacterRepo) Get(ctx context.Context, id string) (*domain.Character, error) {
	if m.err != nil {
		return nil, m.err
	}
	c, ok := m.chars[id]
	if !ok {
		return nil, nil
	}
	return c, nil
}

func (m *mockCharacterRepo) ListByStory(ctx context.Context, storyID string) ([]*domain.Character, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.byStory[storyID], nil
}

func (m *mockCharacterRepo) Update(ctx context.Context, c *domain.Character) error {
	if m.err != nil {
		return m.err
	}
	m.chars[c.ID] = c
	return nil
}

func (m *mockCharacterRepo) GetLatest(ctx context.Context, charID string) (*domain.Character, error) {
	return m.Get(ctx, charID)
}

func (m *mockCharacterRepo) DeleteByStory(ctx context.Context, storyID string) error {
	return nil
}
