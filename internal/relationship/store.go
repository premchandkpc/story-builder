package relationship

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type relKey struct {
	storyID uuid.UUID
	charA   uuid.UUID
	charB   uuid.UUID
}

type MemoryStore struct {
	mu    sync.RWMutex
	rels  map[relKey]*Relationship
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		rels: make(map[relKey]*Relationship),
	}
}

func (m *MemoryStore) Get(storyID, charA, charB uuid.UUID) (*Relationship, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := relKey{storyID, minUUID(charA, charB), maxUUID(charA, charB)}
	rel, ok := m.rels[key]
	if !ok {
		return nil, ErrNotFound
	}
	return rel, nil
}

func (m *MemoryStore) Upsert(rel *Relationship) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	rel.UpdatedAt = time.Now()
	key := relKey{rel.StoryID, minUUID(rel.CharA, rel.CharB), maxUUID(rel.CharA, rel.CharB)}
	m.rels[key] = rel
	return nil
}

func (m *MemoryStore) GetAllForCharacter(storyID, charID uuid.UUID) ([]Relationship, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []Relationship
	for key, rel := range m.rels {
		if key.storyID == storyID && (key.charA == charID || key.charB == charID) {
			result = append(result, *rel)
		}
	}
	return result, nil
}

func (m *MemoryStore) GetHistory(storyID, charA, charB uuid.UUID) ([]Delta, error) {
	rel, err := m.Get(storyID, charA, charB)
	if err != nil {
		return nil, err
	}
	return rel.History, nil
}

func minUUID(a, b uuid.UUID) uuid.UUID {
	for i := 0; i < 16; i++ {
		if a[i] < b[i] {
			return a
		}
		if a[i] > b[i] {
			return b
		}
	}
	return a
}

func maxUUID(a, b uuid.UUID) uuid.UUID {
	for i := 0; i < 16; i++ {
		if a[i] > b[i] {
			return a
		}
		if a[i] < b[i] {
			return b
		}
	}
	return a
}
