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
	drafts  map[string]Draft
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{stories: map[string]Story{}, scenes: map[string]Scene{}, edges: map[string]Edge{}, drafts: map[string]Draft{}}
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
func (r *memoryRepo) CreateDraft(_ context.Context, d Draft) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.drafts[d.ID] = d
	return nil
}
func (r *memoryRepo) FindDraft(_ context.Context, sceneID, hash string) (Draft, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, d := range r.drafts {
		if d.SceneID == sceneID && d.ContextHash == hash {
			return d, nil
		}
	}
	return Draft{}, ErrNotFound
}
func (r *memoryRepo) GetDraft(_ context.Context, id string) (Draft, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.drafts[id]
	if !ok {
		return Draft{}, ErrNotFound
	}
	return d, nil
}
func (r *memoryRepo) UpdateDraft(_ context.Context, d Draft) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.drafts[d.ID] = d
	return nil
}
func (r *memoryRepo) DeleteDraft(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.drafts[id]; !ok {
		return ErrNotFound
	}
	delete(r.drafts, id)
	return nil
}
func (r *memoryRepo) AcceptDraft(_ context.Context, id string) (Scene, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.drafts[id]
	if !ok {
		return Scene{}, ErrNotFound
	}
	scene := r.scenes[d.SceneID]
	current := r.stories[d.StoryID]
	if d.Status == DraftAccepted {
		return scene, nil
	}
	if d.Status != DraftReady {
		return Scene{}, ErrDraftNotReady
	}
	if current.CanonVersion != d.BaseCanonVersion {
		return Scene{}, ErrStaleDraft
	}
	scene.AcceptedDraftID, scene.Prose = d.ID, d.Prose
	r.scenes[scene.ID] = scene
	current.CanonVersion++
	r.stories[current.ID] = current
	d.Status = DraftAccepted
	r.drafts[id] = d
	return scene, nil
}
func (r *memoryRepo) CreateCharacter(_ context.Context, _ Character) error { return nil }
func (r *memoryRepo) ListCharacters(_ context.Context, _ string) ([]Character, error) {
	return nil, nil
}
func (r *memoryRepo) ListCanonEvents(_ context.Context, _ string) ([]CanonEvent, error) {
	return nil, nil
}
func (r *memoryRepo) ListCharacterStates(_ context.Context, _ string) ([]CharacterState, error) {
	return nil, nil
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

func TestRejectDraftLeavesStoryAndSceneUntouched(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryRepo()
	svc := NewService(repo)
	st, _ := svc.CreateStory(ctx, "Map")
	sc, _ := svc.CreateScene(ctx, st.ID, "Start", "", Position{})
	d, err := svc.GenerateDraft(ctx, sc.ID, TemplateGenerator{}, func(string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.RejectDraft(ctx, d.ID); err != nil {
		t.Fatal(err)
	}
	after, _ := repo.GetScene(ctx, sc.ID)
	latest, _ := repo.GetStory(ctx, st.ID)
	if after.Prose != "" || after.AcceptedDraftID != "" || latest.CanonVersion != 0 {
		t.Fatal("reject changed canon")
	}
}
func TestAcceptDraftIsIdempotent(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryRepo()
	svc := NewService(repo)
	st, _ := svc.CreateStory(ctx, "Map")
	sc, _ := svc.CreateScene(ctx, st.ID, "Start", "", Position{})
	d, _ := svc.GenerateDraft(ctx, sc.ID, TemplateGenerator{}, func(string) error { return nil })
	if _, err := svc.AcceptDraft(ctx, d.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AcceptDraft(ctx, d.ID); err != nil {
		t.Fatal(err)
	}
	latest, _ := repo.GetStory(ctx, st.ID)
	if latest.CanonVersion != 1 {
		t.Fatalf("got version %d", latest.CanonVersion)
	}
}

func TestAcceptRequiresAcceptedParents(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryRepo()
	svc := NewService(repo)
	st, _ := svc.CreateStory(ctx, "Map")
	parent, _ := svc.CreateScene(ctx, st.ID, "Parent", "", Position{})
	child, _ := svc.CreateScene(ctx, st.ID, "Child", "", Position{})
	if _, err := svc.CreateEdge(ctx, st.ID, parent.ID, child.ID); err != nil {
		t.Fatal(err)
	}
	childDraft, _ := svc.GenerateDraft(ctx, child.ID, TemplateGenerator{}, func(string) error { return nil })
	if _, err := svc.AcceptDraft(ctx, childDraft.ID); !errors.Is(err, ErrParentsPending) {
		t.Fatalf("expected pending-parent error, got %v", err)
	}
	parentDraft, _ := svc.GenerateDraft(ctx, parent.ID, TemplateGenerator{}, func(string) error { return nil })
	if _, err := svc.AcceptDraft(ctx, parentDraft.ID); err != nil {
		t.Fatal(err)
	}
	childDraft, _ = svc.GenerateDraft(ctx, child.ID, TemplateGenerator{}, func(string) error { return nil })
	if _, err := svc.AcceptDraft(ctx, childDraft.ID); err != nil {
		t.Fatal(err)
	}
}

func TestAcceptedSceneCannotRegenerate(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryRepo()
	svc := NewService(repo)
	st, _ := svc.CreateStory(ctx, "Map")
	scene, _ := svc.CreateScene(ctx, st.ID, "Scene", "", Position{})
	draft, _ := svc.GenerateDraft(ctx, scene.ID, TemplateGenerator{}, func(string) error { return nil })
	if _, err := svc.AcceptDraft(ctx, draft.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GenerateDraft(ctx, scene.ID, TemplateGenerator{}, func(string) error { return nil }); !errors.Is(err, ErrSceneLocked) {
		t.Fatalf("expected scene locked error, got %v", err)
	}
}

func TestProjectCharacterStatesReplaysCanonEvents(t *testing.T) {
	events := []CanonEvent{{StoryID: "story", Seq: 2, ProposedEvent: ProposedEvent{Type: "character_state", CharacterID: "c1", Emotion: "calm", Learned: []string{"the key"}}}, {StoryID: "story", Seq: 1, ProposedEvent: ProposedEvent{Type: "character_state", CharacterID: "c1", Emotion: "alert", Learned: []string{"the door"}}}}
	states := ProjectCharacterStates(events)
	if len(states) != 1 || states[0].Emotion != "calm" || states[0].AsOfCanonVersion != 2 || len(states[0].Knowledge) != 2 {
		t.Fatalf("unexpected replay: %#v", states)
	}
}
