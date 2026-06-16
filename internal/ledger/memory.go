package ledger

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/event"
)

type MemoryStore struct {
	mu      sync.RWMutex
	states  []CharacterState
	estore  event.Store
	bus     event.Bus
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func NewEventSourcedStore(estore event.Store, bus event.Bus) *MemoryStore {
	s := &MemoryStore{
		estore: estore,
		bus:    bus,
	}
	bus.Subscribe(event.EvStateDeltaApplied, func(evt *event.Event) error {
		return s.applyEvent(evt)
	})
	return s
}

func (s *MemoryStore) ReplayOnStartup() error {
	if s.estore == nil {
		return nil
	}
	evts, err := s.estore.GetByType(event.EvStateDeltaApplied, time.Time{}, 0)
	if err != nil {
		return fmt.Errorf("replay events: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states = s.states[:0]
	for _, e := range evts {
		if err := s.applyEventLocked(&e); err != nil {
			return fmt.Errorf("replay event %s: %w", e.ID, err)
		}
	}
	return nil
}

func (s *MemoryStore) GetState(storyID, characterID, asOfNode uuid.UUID) (*CharacterState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, cs := range s.states {
		if cs.StoryID == storyID && cs.CharacterID == characterID && cs.AsOfScene == asOfNode {
			return &cs, nil
		}
	}
	return nil, fmt.Errorf("character_state not found")
}

func (s *MemoryStore) GetAllStates(storyID, asOfNode uuid.UUID) (map[uuid.UUID]*CharacterState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[uuid.UUID]*CharacterState)
	for _, cs := range s.states {
		if cs.StoryID == storyID && cs.AsOfScene == asOfNode {
			c := cs
			result[cs.CharacterID] = &c
		}
	}
	return result, nil
}

func (s *MemoryStore) ApplyDelta(storyID, nodeID uuid.UUID, delta StateDelta) error {
	evt := deltaToEvent(storyID, nodeID, delta)

	if s.estore != nil {
		if err := s.estore.Append(evt); err != nil {
			return fmt.Errorf("append event: %w", err)
		}
	}

	s.mu.Lock()
	if err := s.applyEventLocked(evt); err != nil {
		s.mu.Unlock()
		return err
	}
	s.mu.Unlock()

	if s.bus != nil {
		s.bus.Publish(evt)
	}

	return nil
}

func (s *MemoryStore) ApplyDeltas(storyID, nodeID uuid.UUID, deltas StateDeltas) error {
	for _, d := range deltas.Deltas {
		if err := s.ApplyDelta(storyID, nodeID, d); err != nil {
			return err
		}
	}
	return nil
}

func (s *MemoryStore) GetStateAtBranch(storyID, forkNode, branchNode uuid.UUID) (map[uuid.UUID]*CharacterState, error) {
	return s.GetAllStates(storyID, branchNode)
}

func (s *MemoryStore) applyEvent(evt *event.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.applyEventLocked(evt)
}

func (s *MemoryStore) applyEventLocked(evt *event.Event) error {
	storyID, ok := evt.StoryID, true
	if !ok {
		return nil
	}
	charID, ok := evt.CharID, true
	if !ok {
		return nil
	}
	sceneID := evt.SceneID

	delta, ok := payloadToDelta(evt.Payload)
	if !ok {
		return nil
	}

	for i, cs := range s.states {
		if cs.StoryID == storyID && cs.CharacterID == charID && cs.AsOfScene == sceneID {
			if delta.NewLocation != "" {
				cs.Location = delta.NewLocation
			}
			if delta.Mood != "" {
				cs.Mood = delta.Mood
			}
			cs.Knows = append(cs.Knows, delta.Learned...)
			cs.Items = append(cs.Items, delta.ItemsGained...)
			for _, rel := range delta.RelationshipChanges {
				cs.Relationships[rel.With.String()] = rel.Change
			}
			s.states[i] = cs
			return nil
		}
	}

	cs := CharacterState{
		StoryID:       storyID,
		CharacterID:   charID,
		AsOfScene:      sceneID,
		Location:      delta.NewLocation,
		Mood:          delta.Mood,
		Knows:         delta.Learned,
		DoesNotKnow:   nil,
		Relationships: make(map[string]string),
		Items:         delta.ItemsGained,
		UpdatedAt:     time.Now(),
	}
	for _, rel := range delta.RelationshipChanges {
		cs.Relationships[rel.With.String()] = rel.Change
	}
	s.states = append(s.states, cs)
	return nil
}

func deltaToEvent(storyID, sceneID uuid.UUID, delta StateDelta) *event.Event {
	payload := map[string]any{
		"character":    delta.Character.String(),
		"new_location": delta.NewLocation,
		"learned":      delta.Learned,
		"mood":         delta.Mood,
		"items_gained": delta.ItemsGained,
		"items_lost":   delta.ItemsLost,
	}
	if len(delta.RelationshipChanges) > 0 {
		rels := make([]map[string]string, len(delta.RelationshipChanges))
		for i, r := range delta.RelationshipChanges {
			rels[i] = map[string]string{
				"with":   r.With.String(),
				"change": r.Change,
			}
		}
		payload["relationship_changes"] = rels
	}
	return &event.Event{
		Type:        event.EvStateDeltaApplied,
		AggregateID: delta.Character,
		StoryID:     storyID,
		SceneID:     sceneID,
		CharID:      delta.Character,
		Payload:     payload,
	}
}

func payloadToDelta(payload map[string]any) (StateDelta, bool) {
	if payload == nil {
		return StateDelta{}, false
	}
	delta := StateDelta{}

	if v, ok := payload["character"].(string); ok {
		delta.Character, _ = uuid.Parse(v)
	}
	if delta.Character == uuid.Nil {
		return StateDelta{}, false
	}

	if v, ok := payload["new_location"].(string); ok {
		delta.NewLocation = v
	}
	if v, ok := payload["mood"].(string); ok {
		delta.Mood = v
	}
	if v, ok := payload["learned"].([]string); ok {
		delta.Learned = v
	} else if v, ok := payload["learned"].([]any); ok {
		for _, x := range v {
			if s, ok := x.(string); ok {
				delta.Learned = append(delta.Learned, s)
			}
		}
	}
	if v, ok := payload["items_gained"].([]string); ok {
		delta.ItemsGained = v
	} else if v, ok := payload["items_gained"].([]any); ok {
		for _, x := range v {
			if s, ok := x.(string); ok {
				delta.ItemsGained = append(delta.ItemsGained, s)
			}
		}
	}
	if v, ok := payload["items_lost"].([]string); ok {
		delta.ItemsLost = v
	} else if v, ok := payload["items_lost"].([]any); ok {
		for _, x := range v {
			if s, ok := x.(string); ok {
				delta.ItemsLost = append(delta.ItemsLost, s)
			}
		}
	}
	if v, ok := payload["relationship_changes"].([]any); ok {
		for _, x := range v {
			if m, ok := x.(map[string]any); ok {
				rc := RelationshipChange{}
				if w, ok := m["with"].(string); ok {
					rc.With, _ = uuid.Parse(w)
				}
				if c, ok := m["change"].(string); ok {
					rc.Change = c
				}
				if rc.With != uuid.Nil {
					delta.RelationshipChanges = append(delta.RelationshipChanges, rc)
				}
			}
		}
	}

	return delta, true
}
