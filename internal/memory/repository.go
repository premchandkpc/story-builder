package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

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
	values := mapValues(r.stories, nil)
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
	r.scenes[value.ID] = value
	return nil
}
func (r *Repository) ListScenes(_ context.Context, storyID string) ([]story.Scene, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := mapValues(r.scenes, func(value story.Scene) bool { return value.StoryID == storyID })
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
	edges := mapValues(r.edges, func(edge story.Edge) bool { return edge.StoryID == value.StoryID })
	if story.HasCycle(append(edges, value)) {
		return story.ErrCycle
	}
	r.edges[value.ID] = value
	return nil
}
func (r *Repository) ListEdges(_ context.Context, storyID string) ([]story.Edge, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := mapValues(r.edges, func(value story.Edge) bool { return value.StoryID == storyID })
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
	if _, ok := r.drafts[id]; !ok {
		return story.ErrNotFound
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
	if current.CanonVersion != draft.BaseCanonVersion {
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
	return mapValues(r.characters, func(value story.Character) bool { return value.StoryID == storyID }), nil
}
func (r *Repository) ListCanonEvents(_ context.Context, storyID string) ([]story.CanonEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return filter(r.events, func(event story.CanonEvent) bool { return event.StoryID == storyID }), nil
}
func (r *Repository) ListCharacterStates(ctx context.Context, storyID string) ([]story.CharacterState, error) {
	events, _ := r.ListCanonEvents(ctx, storyID)
	return story.ProjectCharacterStates(events), nil
}

func get[T any](values map[string]T, id string) (T, error) {
	value, ok := values[id]
	if !ok {
		var zero T
		return zero, story.ErrNotFound
	}
	return value, nil
}

func mapValues[T any](items map[string]T, include func(T) bool) []T {
	result := make([]T, 0, len(items))
	for _, item := range items {
		if include == nil || include(item) {
			result = append(result, item)
		}
	}
	return result
}

func filter[T any](items []T, include func(T) bool) []T {
	result := make([]T, 0, len(items))
	for _, item := range items {
		if include(item) {
			result = append(result, item)
		}
	}
	return result
}
