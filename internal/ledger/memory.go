package ledger

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MemoryStore struct {
	mu     sync.RWMutex
	states []CharacterState
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (s *MemoryStore) GetState(storyID, characterID, asOfNode uuid.UUID) (*CharacterState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, cs := range s.states {
		if cs.StoryID == storyID && cs.CharacterID == characterID && cs.AsOfNode == asOfNode {
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
		if cs.StoryID == storyID && cs.AsOfNode == asOfNode {
			c := cs
			result[cs.CharacterID] = &c
		}
	}
	return result, nil
}

func (s *MemoryStore) ApplyDelta(storyID, nodeID uuid.UUID, delta StateDelta) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, cs := range s.states {
		if cs.StoryID == storyID && cs.CharacterID == delta.Character && cs.AsOfNode == nodeID {
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
		CharacterID:   delta.Character,
		AsOfNode:      nodeID,
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
