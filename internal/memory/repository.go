package memory

import (
	"context"
	"sort"
	"sync"

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
	value, ok := r.stories[id]
	if !ok {
		return story.Story{}, story.ErrNotFound
	}
	return value, nil
}
func (r *Repository) ListStories(_ context.Context) ([]story.Story, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]story.Story, 0, len(r.stories))
	for _, value := range r.stories {
		values = append(values, value)
	}
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
	value, ok := r.scenes[id]
	if !ok {
		return story.Scene{}, story.ErrNotFound
	}
	return value, nil
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
	var values []story.Scene
	for _, value := range r.scenes {
		if value.StoryID == storyID {
			values = append(values, value)
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i].CreatedAt.Before(values[j].CreatedAt) })
	return values, nil
}
func (r *Repository) CreateEdge(_ context.Context, value story.Edge) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.edges[value.ID] = value
	return nil
}
func (r *Repository) ListEdges(_ context.Context, storyID string) ([]story.Edge, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var values []story.Edge
	for _, value := range r.edges {
		if value.StoryID == storyID {
			values = append(values, value)
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i].CreatedAt.Before(values[j].CreatedAt) })
	return values, nil
}
func (r *Repository) CreateDraft(_ context.Context, value story.Draft) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.drafts[value.ID] = value
	return nil
}
func (r *Repository) FindDraft(_ context.Context, sceneID, hash string) (story.Draft, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, value := range r.drafts {
		if value.SceneID == sceneID && value.ContextHash == hash {
			return value, nil
		}
	}
	return story.Draft{}, story.ErrNotFound
}
func (r *Repository) GetDraft(_ context.Context, id string) (story.Draft, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.drafts[id]
	if !ok {
		return story.Draft{}, story.ErrNotFound
	}
	return value, nil
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
	if len(draft.ProposedEvents) == 0 {
		current.CanonVersion++
	}
	for _, proposal := range draft.ProposedEvents {
		current.CanonVersion++
		r.events = append(r.events, story.CanonEvent{ID: draft.ID + proposal.CharacterID, StoryID: draft.StoryID, Seq: current.CanonVersion, SceneID: draft.SceneID, DraftID: draft.ID, ProposedEvent: proposal, At: draft.CreatedAt})
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
	var out []story.Character
	for _, value := range r.characters {
		if value.StoryID == storyID {
			out = append(out, value)
		}
	}
	return out, nil
}
func (r *Repository) ListCanonEvents(_ context.Context, storyID string) ([]story.CanonEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []story.CanonEvent
	for _, event := range r.events {
		if event.StoryID == storyID {
			out = append(out, event)
		}
	}
	return out, nil
}
func (r *Repository) ListCharacterStates(ctx context.Context, storyID string) ([]story.CharacterState, error) {
	events, _ := r.ListCanonEvents(ctx, storyID)
	return story.ProjectCharacterStates(events), nil
}
