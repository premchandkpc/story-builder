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
	mu         sync.RWMutex
	stories    map[string]story.Story
	scenes     map[string]story.Scene
	edges      map[string]story.Edge
	drafts     map[string]story.Draft
	characters map[string]story.Character
	events     []story.CanonEvent
}

func NewRepository() *Repository {
	return &Repository{stories: map[string]story.Story{}, scenes: map[string]story.Scene{}, edges: map[string]story.Edge{}, drafts: map[string]story.Draft{}, characters: map[string]story.Character{}}
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
	current, ok := r.scenes[value.ID]
	if !ok {
		return story.ErrNotFound
	}
	current.Title, current.Brief, current.Position, current.UpdatedAt = value.Title, value.Brief, value.Position, value.UpdatedAt
	r.scenes[value.ID] = current
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
	edges := collection.MapValues(r.edges, func(edge story.Edge) bool { return edge.StoryID == value.StoryID })
	if story.HasCycle(append(edges, value)) {
		return story.ErrCycle
	}
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
		if !draft.ExpiresAt.After(now) {
			delete(r.drafts, id)
		}
	}
	for _, draft := range r.drafts {
		if draft.SceneID == value.SceneID && draft.ContextHash == value.ContextHash && (draft.Status == story.DraftGenerating || draft.Status == story.DraftReady) {
			return story.ErrDraftInProgress
		}
	}
	r.drafts[value.ID] = value
	return nil
}
func (r *Repository) FindDraft(_ context.Context, sceneID, hash string) (story.Draft, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	now := time.Now().UTC()
	for _, value := range r.drafts {
		if value.SceneID == sceneID && value.ContextHash == hash && value.Status == story.DraftReady && value.ExpiresAt.After(now) {
			return value, nil
		}
	}
	return story.Draft{}, story.ErrNotFound
}
func (r *Repository) GetDraft(_ context.Context, id string) (story.Draft, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return get(r.drafts, id)
}
func (r *Repository) UpdateDraft(_ context.Context, value story.Draft) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.drafts[value.ID]; !ok {
		return story.ErrNotFound
	}
	r.drafts[value.ID] = value
	return nil
}
func (r *Repository) DeleteDraft(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	draft, ok := r.drafts[id]
	if !ok {
		return story.ErrNotFound
	}
	if draft.Status == story.DraftAccepted {
		return story.ErrSceneLocked
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
	if draft.Status != story.DraftReady {
		return story.Scene{}, story.ErrDraftNotReady
	}
	for _, edge := range r.edges {
		if edge.StoryID == draft.StoryID && edge.ToSceneID == draft.SceneID && r.scenes[edge.FromSceneID].AcceptedDraftID == "" {
			return story.Scene{}, story.ErrParentsPending
		}
	}
	characters := collection.MapValues(r.characters, func(character story.Character) bool { return character.StoryID == draft.StoryID })
	if current.CanonVersion != draft.BaseCanonVersion || !draft.ExpiresAt.After(time.Now()) || draft.ContextHash != story.DraftContextHashForGenerator(scene, current.CanonVersion, draft.GeneratorKey, characters...) {
		return story.Scene{}, story.ErrStaleDraft
	}
	scene.AcceptedDraftID = draft.ID
	scene.Prose = draft.Prose
	r.scenes[scene.ID] = scene
	events := draft.ProposedEvents
	if len(events) == 0 {
		events = []story.ProposedEvent{{Type: "scene_accepted"}}
	}
	for i, proposal := range events {
		current.CanonVersion++
		r.events = append(r.events, story.CanonEvent{ID: fmt.Sprintf("%s:%d", draft.ID, i), StoryID: draft.StoryID, Seq: current.CanonVersion, SceneID: draft.SceneID, DraftID: draft.ID, ProposedEvent: proposal, At: draft.CreatedAt})
	}
	r.stories[current.ID] = current
	draft.Status = story.DraftAccepted
	r.drafts[id] = draft
	return scene, nil
}
func (r *Repository) CreateCharacter(_ context.Context, value story.Character) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.characters[value.CharacterID] = value
	return nil
}
func (r *Repository) ListCharacters(_ context.Context, storyID string) ([]story.Character, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return collection.MapValues(r.characters, func(value story.Character) bool { return value.StoryID == storyID }), nil
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
