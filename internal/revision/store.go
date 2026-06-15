package revision

import (
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MemoryStore struct {
	mu        sync.RWMutex
	revisions map[uuid.UUID][]Revision
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		revisions: make(map[uuid.UUID][]Revision),
	}
}

func (m *MemoryStore) Create(rev *Revision) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	rev.ID = uuid.New()
	rev.CreatedAt = time.Now()
	existing := m.revisions[rev.SceneID]
	rev.Version = len(existing) + 1
	m.revisions[rev.SceneID] = append(existing, *rev)
	return nil
}

func (m *MemoryStore) GetLatest(sceneID uuid.UUID) (*Revision, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	revs, ok := m.revisions[sceneID]
	if !ok || len(revs) == 0 {
		return nil, ErrNotFound
	}
	return &revs[len(revs)-1], nil
}

func (m *MemoryStore) GetVersion(sceneID uuid.UUID, version int) (*Revision, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	revs, ok := m.revisions[sceneID]
	if !ok {
		return nil, ErrNotFound
	}
	for _, r := range revs {
		if r.Version == version {
			return &r, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) List(sceneID uuid.UUID) ([]Revision, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	revs, ok := m.revisions[sceneID]
	if !ok {
		return nil, nil
	}
	result := make([]Revision, len(revs))
	copy(result, revs)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Version < result[j].Version
	})
	return result, nil
}
