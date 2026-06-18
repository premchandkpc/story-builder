package service

import (
	"context"
	"testing"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type mockEdgeRepo2 struct {
	repository.SceneEdgeRepository
	edges map[string]*domain.SceneEdge
	err   error
}

func newMockEdgeRepo2() *mockEdgeRepo2 {
	return &mockEdgeRepo2{
		edges: make(map[string]*domain.SceneEdge),
	}
}

func (m *mockEdgeRepo2) Create(ctx context.Context, e *domain.SceneEdge) error {
	if m.err != nil {
		return m.err
	}
	e.ID = "edge-" + e.FromSceneID + "-" + e.ToSceneID
	m.edges[e.ID] = e
	return nil
}

func (m *mockEdgeRepo2) ListByStory(ctx context.Context, storyID string) ([]*domain.SceneEdge, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*domain.SceneEdge
	for _, e := range m.edges {
		if e.StoryID == storyID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *mockEdgeRepo2) Delete(ctx context.Context, storyID, fromSceneID, toSceneID string) error {
	if m.err != nil {
		return m.err
	}
	id := "edge-" + fromSceneID + "-" + toSceneID
	delete(m.edges, id)
	return nil
}

func TestEdgeService_Create(t *testing.T) {
	mock := newMockEdgeRepo2()
	svc := NewEdgeService(mock)

	e := &domain.SceneEdge{StoryID: "s1", FromSceneID: "a", ToSceneID: "b", Type: "seq"}
	got, err := svc.Create(context.Background(), e)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.FromSceneID != "a" {
		t.Fatalf("expected from=a, got %q", got.FromSceneID)
	}
}

func TestEdgeService_List(t *testing.T) {
	mock := newMockEdgeRepo2()
	svc := NewEdgeService(mock)
	svc.Create(context.Background(), &domain.SceneEdge{StoryID: "s1", FromSceneID: "a", ToSceneID: "b"})
	svc.Create(context.Background(), &domain.SceneEdge{StoryID: "s1", FromSceneID: "b", ToSceneID: "c"})

	edges, err := svc.List(context.Background(), "s1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(edges))
	}
}

func TestEdgeService_Delete(t *testing.T) {
	mock := newMockEdgeRepo2()
	svc := NewEdgeService(mock)
	svc.Create(context.Background(), &domain.SceneEdge{StoryID: "s1", FromSceneID: "a", ToSceneID: "b"})
	svc.Create(context.Background(), &domain.SceneEdge{StoryID: "s1", FromSceneID: "b", ToSceneID: "c"})

	err := svc.Delete(context.Background(), "s1", "a", "b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	edges, _ := svc.List(context.Background(), "s1")
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge after delete, got %d", len(edges))
	}
}

type mockTimelineRepo struct {
	repository.TimelineRepository
	events  map[string]*domain.TimelineEvent
	byStory map[string][]*domain.TimelineEvent
}

func newMockTimelineRepo() *mockTimelineRepo {
	return &mockTimelineRepo{
		events:  make(map[string]*domain.TimelineEvent),
		byStory: make(map[string][]*domain.TimelineEvent),
	}
}

func (m *mockTimelineRepo) Create(ctx context.Context, e *domain.TimelineEvent) error {
	e.ID = "tl-" + e.SceneID
	m.events[e.ID] = e
	m.byStory[e.StoryID] = append(m.byStory[e.StoryID], e)
	return nil
}

func (m *mockTimelineRepo) ListByStory(ctx context.Context, storyID string) ([]*domain.TimelineEvent, error) {
	return m.byStory[storyID], nil
}

func TestTimelineService_Create(t *testing.T) {
	mock := newMockTimelineRepo()
	svc := NewTimelineService(mock)

	e := &domain.TimelineEvent{StoryID: "s1", SceneID: "sc1", Title: "Chapter 1", Order: 1}
	got, err := svc.Create(context.Background(), e)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != "Chapter 1" {
		t.Fatalf("expected 'Chapter 1', got %q", got.Title)
	}
}

func TestTimelineService_List(t *testing.T) {
	mock := newMockTimelineRepo()
	svc := NewTimelineService(mock)
	svc.Create(context.Background(), &domain.TimelineEvent{StoryID: "s1", SceneID: "sc1", Title: "A", Order: 1})
	svc.Create(context.Background(), &domain.TimelineEvent{StoryID: "s1", SceneID: "sc2", Title: "B", Order: 2})

	events, err := svc.List(context.Background(), "s1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestTimelineService_List_Empty(t *testing.T) {
	mock := newMockTimelineRepo()
	svc := NewTimelineService(mock)

	events, err := svc.List(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

type mockSummaryRepo struct {
	repository.SummaryRepository
	summaries map[string]*domain.Summary
}

func newMockSummaryRepo() *mockSummaryRepo {
	return &mockSummaryRepo{summaries: make(map[string]*domain.Summary)}
}

func (m *mockSummaryRepo) GetByLevel(ctx context.Context, storyID, level string) (*domain.Summary, error) {
	for _, s := range m.summaries {
		if s.StoryID == storyID && s.Level == level {
			return s, nil
		}
	}
	return nil, nil
}

func (m *mockSummaryRepo) GetSceneSummary(ctx context.Context, storyID, sceneID string) (*domain.Summary, error) {
	for _, s := range m.summaries {
		if s.StoryID == storyID && s.SceneID == sceneID {
			return s, nil
		}
	}
	return nil, nil
}

func TestSummaryService_GetByLevel(t *testing.T) {
	mock := newMockSummaryRepo()
	svc := NewSummaryService(mock)

	mock.summaries["1"] = &domain.Summary{StoryID: "s1", Level: "story", Content: "The whole story"}

	got, err := svc.GetByLevel(context.Background(), "s1", "story")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected summary, got nil")
	}
	if got.Content != "The whole story" {
		t.Fatalf("expected 'The whole story', got %q", got.Content)
	}
}

func TestSummaryService_GetByLevel_Missing(t *testing.T) {
	mock := newMockSummaryRepo()
	svc := NewSummaryService(mock)

	got, err := svc.GetByLevel(context.Background(), "s1", "story")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for missing summary")
	}
}

func TestSummaryService_GetSceneSummary(t *testing.T) {
	mock := newMockSummaryRepo()
	svc := NewSummaryService(mock)

	mock.summaries["1"] = &domain.Summary{StoryID: "s1", SceneID: "sc1", Level: "scene", Content: "Scene details"}

	got, err := svc.GetSceneSummary(context.Background(), "s1", "sc1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected summary, got nil")
	}
	if got.Content != "Scene details" {
		t.Fatalf("expected 'Scene details', got %q", got.Content)
	}
}

type mockMemoryRepo struct {
	repository.MemoryRepository
	memories map[string][]*domain.CharacterMemory
}

func newMockMemoryRepo() *mockMemoryRepo {
	return &mockMemoryRepo{memories: make(map[string][]*domain.CharacterMemory)}
}

func (m *mockMemoryRepo) ListByCharacter(ctx context.Context, charID string) ([]*domain.CharacterMemory, error) {
	return m.memories[charID], nil
}

func (m *mockMemoryRepo) Create(ctx context.Context, mem *domain.CharacterMemory) error {
	m.memories[mem.CharacterID] = append(m.memories[mem.CharacterID], mem)
	return nil
}

func TestMemoryService_ListByCharacter(t *testing.T) {
	mock := newMockMemoryRepo()
	svc := NewMemoryService(mock)

	mock.Create(context.Background(), &domain.CharacterMemory{
		CharacterID: "char1", Content: "Remember this", Importance: 0.8,
	})
	mock.Create(context.Background(), &domain.CharacterMemory{
		CharacterID: "char1", Content: "And this", Importance: 0.5,
	})

	mems, err := svc.ListByCharacter(context.Background(), "char1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mems) != 2 {
		t.Fatalf("expected 2 memories, got %d", len(mems))
	}
}

func TestMemoryService_ListByCharacter_Empty(t *testing.T) {
	mock := newMockMemoryRepo()
	svc := NewMemoryService(mock)

	mems, err := svc.ListByCharacter(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mems) != 0 {
		t.Fatalf("expected 0 memories, got %d", len(mems))
	}
}
