package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
)

type MemoryStore struct {
	mu          sync.RWMutex
	generations []Generation
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (s *MemoryStore) Create(g *Generation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.generations = append(s.generations, *g)
	return nil
}

func (s *MemoryStore) Accept(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.generations {
		if s.generations[i].ID == id {
			s.generations[i].Accepted = true
			return nil
		}
	}
	return fmt.Errorf("generation %s not found", id)
}

func (s *MemoryStore) Reject(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.generations {
		if s.generations[i].ID == id {
			s.generations[i].Accepted = false
			return nil
		}
	}
	return fmt.Errorf("generation %s not found", id)
}

func (s *MemoryStore) GetByNode(nodeID string) ([]Generation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []Generation
	for _, g := range s.generations {
		if g.NodeID == nodeID {
			result = append(result, g)
		}
	}
	return result, nil
}

func (s *MemoryStore) IsStale(nodeID, currentHash string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hasAccepted := false
	for _, g := range s.generations {
		if g.NodeID == nodeID && g.Accepted {
			hasAccepted = true
			if g.ContextHash != currentHash {
				return true, nil
			}
		}
	}
	return !hasAccepted, nil
}

func ComputeHash(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("compiler: hash marshal: %v", err))
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
