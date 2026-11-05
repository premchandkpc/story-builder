package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/premchand/story-builder/internal/collection"
	"github.com/premchand/story-builder/internal/story"
)

// Repository makes the graph workspace runnable without external services.
// It is intentionally only a local-development adapter; restarting clears it.
type Repository struct {
	mu                sync.RWMutex
	stories           map[string]story.Story
	scenes            map[string]story.Scene
	edges             map[string]story.Edge
	drafts            map[string]story.Draft
	characters        map[string]story.Character
	characterVersions map[string]story.Character
	events            []story.CanonEvent
}

func NewRepository() *Repository {
	return &Repository{stories: map[string]story.Story{}, scenes: map[string]story.Scene{}, edges: map[string]story.Edge{}, drafts: map[string]story.Draft{}, characters: map[string]story.Character{}, characterVersions: map[string]story.Character{}}
}

func (r *Repository) CreateStory(_ context.Context, value story.Story) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stories[value.ID] = value
	return nil
}
func (r *Repository) GetStory(_ context.Context, id string) (story.Story, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return get(r.stories, id)
}
func (r *Repository) ListStories(_ context.Context) ([]story.Story, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := collection.MapValues(r.stories, nil)
	sort.Slice(values, func(i, j int) bool { return values[i].CreatedAt.After(values[j].CreatedAt) })
	return values, nil
}
func (r *Repository) CreateScene(_ context.Context, value story.Scene) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scenes[value.ID] = value
	return nil
}
func (r *Repository) GetScene(_ context.Context, id string) (story.Scene, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return get(r.scenes, id)
}
func (r *Repository) UpdateScene(_ context.Context, value story.Scene) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.scenes[value.ID]; !ok {
		return story.ErrNotFound
	}
	old := r.scenes[value.ID]
	if value.Revision != old.Revision+1 {
		return story.ErrConflict
	}
	if old.AcceptedDraftID != "" && (value.Title != old.Title || value.Brief != old.Brief) {
		return story.ErrSceneLocked
	}
	// Update only author-owned fields, preserving canonical acceptance fields.
	old.Title = value.Title
	old.Brief = value.Brief
	old.Position = value.Position
	old.UpdatedAt = value.UpdatedAt
	old.Revision = value.Revision
	old.ContentRevision = value.ContentRevision
	r.scenes[value.ID] = old
	return nil
}
func (r *Repository) ListScenes(_ context.Context, storyID string) ([]story.Scene, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := collection.MapValues(r.scenes, func(value story.Scene) bool { return value.StoryID == storyID })
	sort.Slice(values, func(i, j int) bool { return values[i].CreatedAt.Before(values[j].CreatedAt) })
	return values, nil
}
func (r *Repository) CreateEdge(_ context.Context, value story.Edge) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, edge := range r.edges {
		if edge.StoryID == value.StoryID && edge.FromSceneID == value.FromSceneID && edge.ToSceneID == value.ToSceneID {
			return story.ErrDuplicateEdge
		}
	}
	target, ok := r.scenes[value.ToSceneID]
	if !ok {
		return story.ErrNotFound
	}
	if target.AcceptedDraftID != "" {
		return story.ErrSceneLocked
	}
	edges := collection.MapValues(r.edges, func(edge story.Edge) bool { return edge.StoryID == value.StoryID })
	if story.HasCycle(append(edges, value)) {
		return story.ErrCycle
	}
	current := r.stories[value.StoryID]
	current.GraphVersion++
	r.stories[value.StoryID] = current
	r.edges[value.ID] = value
	return nil
}
func (r *Repository) ListEdges(_ context.Context, storyID string) ([]story.Edge, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := collection.MapValues(r.edges, func(value story.Edge) bool { return value.StoryID == storyID })
	sort.Slice(values, func(i, j int) bool { return values[i].CreatedAt.Before(values[j].CreatedAt) })
	return values, nil
}
func (r *Repository) CreateDraft(_ context.Context, value story.Draft) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	for id, draft := range r.drafts {
		if draft.Status != story.DraftAccepted && !draft.ExpiresAt.After(now) {
			delete(r.drafts, id)
		}
	}
	for id, draft := range r.drafts {
		if draft.Status == story.DraftGenerating && ((!draft.LeaseUntil.IsZero() && !draft.LeaseUntil.After(now)) || (draft.LeaseUntil.IsZero() && !draft.CreatedAt.After(now.Add(-3*time.Minute)))) {
			draft.Status = story.DraftFailed
			r.drafts[id] = draft
		}
	}
	for _, draft := range r.drafts {
		if draft.SceneID == value.SceneID && draft.ContextHash == value.ContextHash && (draft.Status == story.DraftGenerating || draft.Status == story.DraftReady) {
			return story.ErrDraftInProgress
		}
	}
	r.drafts[value.ID] = cloneDraft(value)
	return nil
}
func (r *Repository) FindDraft(_ context.Context, sceneID, hash string) (story.Draft, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	now := time.Now().UTC()
	for _, value := range r.drafts {
		if value.SceneID == sceneID && value.ContextHash == hash && value.Status == story.DraftReady && value.ExpiresAt.After(now) {
			return cloneDraft(value), nil
		}
	}
	return story.Draft{}, story.ErrNotFound
}
func (r *Repository) GetDraft(_ context.Context, id string) (story.Draft, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, err := get(r.drafts, id)
	return cloneDraft(value), err
}
func (r *Repository) UpdateDraft(_ context.Context, value story.Draft) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.drafts[value.ID]; !ok {
		return story.ErrNotFound
	}
	if r.drafts[value.ID].Status != story.DraftGenerating {
		return story.ErrDraftNotReady
	}
	r.drafts[value.ID] = cloneDraft(value)
	return nil
}
func (r *Repository) DeleteDraft(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.drafts[id]; !ok {
		return story.ErrNotFound
	}
	if r.drafts[id].Status != story.DraftReady && r.drafts[id].Status != story.DraftFailed {
		return story.ErrDraftNotReady
	}
	delete(r.drafts, id)
	return nil
}
func (r *Repository) AcceptDraft(_ context.Context, id string) (story.Scene, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	draft, ok := r.drafts[id]
	if !ok {
		return story.Scene{}, story.ErrNotFound
	}
	scene, ok := r.scenes[draft.SceneID]
	if !ok {
		return story.Scene{}, story.ErrNotFound
	}
	current := r.stories[draft.StoryID]
	if draft.Status == story.DraftAccepted {
		return scene, nil
	}
	if draft.Status != story.DraftReady || !draft.ExpiresAt.After(time.Now().UTC()) {
		return story.Scene{}, story.ErrDraftNotReady
	}
	for _, edge := range r.edges {
		if edge.StoryID == draft.StoryID && edge.ToSceneID == draft.SceneID && r.scenes[edge.FromSceneID].AcceptedDraftID == "" {
			return story.Scene{}, story.ErrParentsPending
		}
	}
	if scene.AcceptedDraftID != "" {
		return story.Scene{}, story.ErrSceneLocked
	}
	if scene.ContentRevision != draft.BaseSceneRevision || current.CharacterRevision != draft.BaseCharacterRevision || current.GraphVersion != draft.BaseGraphVersion || current.CanonVersion != draft.BaseCanonVersion {
		return story.Scene{}, story.ErrStaleDraft
	}
	scene.AcceptedDraftID = draft.ID
	scene.Prose = draft.Prose
	scene.Revision++
	scene.UpdatedAt = time.Now().UTC()
	r.scenes[scene.ID] = scene
	events := draft.ProposedEvents
	if len(events) == 0 {
		events = []story.ProposedEvent{{Type: "scene_accepted"}}
	}
	for i, proposal := range events {
		current.CanonVersion++
		r.events = append(r.events, story.CanonEvent{ID: fmt.Sprintf("%s:%d", draft.ID, i), StoryID: draft.StoryID, Seq: current.CanonVersion, SceneID: draft.SceneID, DraftID: draft.ID, ProposedEvent: proposal, At: time.Now().UTC()})
	}
	r.stories[current.ID] = current
	draft.Status = story.DraftAccepted
	draft.ExpiresAt = time.Time{}
	r.drafts[id] = draft
	return scene, nil
}
func (r *Repository) CreateCharacter(ctx context.Context, value story.Character) error {
	return r.SaveCharacter(ctx, value, 0)
}
func (r *Repository) SaveCharacter(_ context.Context, value story.Character, expected int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.stories[value.StoryID]
	if !ok {
		return story.ErrNotFound
	}
	old, exists := r.characters[value.CharacterID]
	if (exists && (old.StoryID != value.StoryID || old.Version != expected)) || (!exists && expected != 0) || value.Version != expected+1 {
		return story.ErrConflict
	}
	r.characters[value.CharacterID] = cloneCharacter(value)
	r.characterVersions[fmt.Sprintf("%s:%d", value.CharacterID, value.Version)] = cloneCharacter(value)
	current.CharacterRevision++
	r.stories[current.ID] = current
	return nil
}
func (r *Repository) ListCharacters(_ context.Context, storyID string) ([]story.Character, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := collection.MapValues(r.characters, func(value story.Character) bool { return value.StoryID == storyID })
	for i := range values {
		values[i] = cloneCharacter(values[i])
	}
	return values, nil
}
func (r *Repository) ListCanonEvents(_ context.Context, storyID string) ([]story.CanonEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return collection.Filter(r.events, func(event story.CanonEvent) bool { return event.StoryID == storyID }), nil
}
func (r *Repository) ListCharacterStates(ctx context.Context, storyID string) ([]story.CharacterState, error) {
	events, _ := r.ListCanonEvents(ctx, storyID)
	return story.ProjectCharacterStates(events), nil
}

func get[T any](values map[string]T, id string) (T, error) {
	value, ok := collection.Get(values, id)
	if !ok {
		var zero T
		return zero, story.ErrNotFound
	}
	return value, nil
}
