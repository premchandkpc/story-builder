package timeline

import (
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/event"
)

type BranchID string

type Branch struct {
	ID          BranchID  `json:"id"`
	StoryID     uuid.UUID `json:"story_id"`
	Name        string    `json:"name"`
	ParentID    BranchID  `json:"parent_id,omitempty"`
	ForkPoint   int       `json:"fork_point,omitempty"`
	MergedInto  BranchID  `json:"merged_into,omitempty"`
	IsAlternate bool      `json:"is_alternate"`
	CreatedAt   time.Time `json:"created_at"`
}

type SceneRef struct {
	StoryID   uuid.UUID `json:"story_id"`
	SceneID   uuid.UUID `json:"scene_id"`
	Title     string    `json:"title"`
	Order     int       `json:"order"`
	BranchID  BranchID  `json:"branch_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Engine struct {
	store      event.Store
	bus        event.Bus
	branches   map[BranchID]*Branch
	assignments map[uuid.UUID]BranchID
}

func NewEngine(store event.Store, bus event.Bus) *Engine {
	return &Engine{
		store:       store,
		bus:         bus,
		branches:    make(map[BranchID]*Branch),
		assignments: make(map[uuid.UUID]BranchID),
	}
}

func (e *Engine) RecordScene(sceneID, storyID uuid.UUID, branchID BranchID, title string, order int) error {
	e.bus.Publish(&event.Event{
		Type:        event.EvTimelineUpdated,
		AggregateID: storyID,
		StoryID:     storyID,
		SceneID:     sceneID,
		Payload: map[string]any{
			"branch_id": string(branchID),
			"title":     title,
			"order":     order,
		},
	})
	return nil
}

func (e *Engine) CreateBranch(storyID uuid.UUID, name string, forkPoint int, isAlternate bool) (BranchID, error) {
	id := BranchID(uuid.New().String())
	e.branches[id] = &Branch{
		ID:          id,
		StoryID:     storyID,
		Name:        name,
		ForkPoint:   forkPoint,
		IsAlternate: isAlternate,
		CreatedAt:   time.Now(),
	}

	e.bus.Publish(&event.Event{
		Type:        event.EvTimelineUpdated,
		AggregateID: storyID,
		Payload: map[string]any{
			"action":       "branch_created",
			"branch_id":    string(id),
			"name":         name,
			"fork_point":   forkPoint,
			"is_alternate": isAlternate,
		},
	})

	return id, nil
}

func (e *Engine) ForkFrom(parentID BranchID, name string) (BranchID, error) {
	parent, ok := e.branches[parentID]
	if !ok {
		return "", fmt.Errorf("timeline: parent branch %q not found", parentID)
	}

	childID := BranchID(uuid.New().String())
	e.branches[childID] = &Branch{
		ID:        childID,
		StoryID:   parent.StoryID,
		Name:      name,
		ParentID:  parentID,
		ForkPoint: parent.ForkPoint,
		CreatedAt: time.Now(),
	}

	return childID, nil
}

func (e *Engine) MergeBranch(sourceID, targetID BranchID) error {
	source, ok := e.branches[sourceID]
	if !ok {
		return fmt.Errorf("timeline: source branch %q not found", sourceID)
	}
	if _, ok := e.branches[targetID]; !ok {
		return fmt.Errorf("timeline: target branch %q not found", targetID)
	}

	source.MergedInto = targetID
	return nil
}

func (e *Engine) Past(storyID uuid.UUID, upToOrder int) ([]SceneRef, error) {
	events, err := e.store.GetByStory(storyID, event.EvTimelineUpdated, 0)
	if err != nil {
		return nil, err
	}

	var past []SceneRef
	for _, evt := range events {
		if evt.Type != event.EvTimelineUpdated {
			continue
		}
		order, _ := evt.Payload["order"].(int)
		if order > 0 && order <= upToOrder {
			past = append(past, e.eventToSceneRef(&evt))
		}
	}

	sort.Slice(past, func(i, j int) bool {
		return past[i].Order < past[j].Order
	})
	return past, nil
}

func (e *Engine) Future(storyID uuid.UUID, afterOrder int) ([]SceneRef, error) {
	events, err := e.store.GetByStory(storyID, event.EvTimelineUpdated, 0)
	if err != nil {
		return nil, err
	}

	var future []SceneRef
	for _, evt := range events {
		if evt.Type != event.EvTimelineUpdated {
			continue
		}
		order, _ := evt.Payload["order"].(int)
		if order > afterOrder {
			future = append(future, e.eventToSceneRef(&evt))
		}
	}

	sort.Slice(future, func(i, j int) bool {
		return future[i].Order < future[j].Order
	})
	return future, nil
}

func (e *Engine) BranchScenes(storyID uuid.UUID, branchID BranchID) ([]SceneRef, error) {
	events, err := e.store.GetByStory(storyID, event.EvTimelineUpdated, 0)
	if err != nil {
		return nil, err
	}

	var scenes []SceneRef
	for _, evt := range events {
		if evt.Type != event.EvTimelineUpdated {
			continue
		}
		bID, _ := evt.Payload["branch_id"].(string)
		if bID == string(branchID) {
			scenes = append(scenes, e.eventToSceneRef(&evt))
		}
	}

	sort.Slice(scenes, func(i, j int) bool {
		return scenes[i].Order < scenes[j].Order
	})
	return scenes, nil
}

func (e *Engine) eventToSceneRef(evt *event.Event) SceneRef {
	order, _ := evt.Payload["order"].(int)
	title, _ := evt.Payload["title"].(string)
	branchID, _ := evt.Payload["branch_id"].(string)

	return SceneRef{
		StoryID:   evt.StoryID,
		SceneID:   evt.SceneID,
		Title:     title,
		Order:     order,
		BranchID:  BranchID(branchID),
		CreatedAt: evt.Timestamp,
	}
}
