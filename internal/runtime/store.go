package runtime

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type Store interface {
	CreateSceneRuntime(sr *SceneRuntime) error
	GetSceneRuntime(sceneID uuid.UUID) (*SceneRuntime, error)
	UpdateSceneRuntime(sr *SceneRuntime) error

	CreateSnapshot(snap *RuntimeSnapshot) error
	GetSnapshot(snapshotID uuid.UUID) (*RuntimeSnapshot, error)
	ListSnapshots(storyID uuid.UUID) ([]RuntimeSnapshot, error)

	UpsertCharRuntime(storyID uuid.UUID, cr *CharRuntime) error
	GetCharRuntime(storyID, charID uuid.UUID) (*CharRuntime, error)
	ListCharRuntimes(storyID uuid.UUID) (map[uuid.UUID]CharRuntime, error)
}

type MemoryStore struct {
	mu        sync.RWMutex
	sceneRTs  map[uuid.UUID]*SceneRuntime
	snapshots map[uuid.UUID]*RuntimeSnapshot
	charRTs   map[uuid.UUID]map[uuid.UUID]*CharRuntime
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sceneRTs:  make(map[uuid.UUID]*SceneRuntime),
		snapshots: make(map[uuid.UUID]*RuntimeSnapshot),
		charRTs:   make(map[uuid.UUID]map[uuid.UUID]*CharRuntime),
	}
}

func (m *MemoryStore) CreateSceneRuntime(sr *SceneRuntime) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	sr.UpdatedAt = time.Now()
	m.sceneRTs[sr.SceneID] = sr
	return nil
}

func (m *MemoryStore) GetSceneRuntime(sceneID uuid.UUID) (*SceneRuntime, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sr, ok := m.sceneRTs[sceneID]
	if !ok {
		return nil, ErrNotFound
	}
	return sr, nil
}

func (m *MemoryStore) UpdateSceneRuntime(sr *SceneRuntime) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	sr.UpdatedAt = time.Now()
	m.sceneRTs[sr.SceneID] = sr
	return nil
}

func (m *MemoryStore) CreateSnapshot(snap *RuntimeSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	snap.CreatedAt = time.Now()
	m.snapshots[snap.SnapshotID] = snap
	return nil
}

func (m *MemoryStore) GetSnapshot(snapshotID uuid.UUID) (*RuntimeSnapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	snap, ok := m.snapshots[snapshotID]
	if !ok {
		return nil, ErrNotFound
	}
	return snap, nil
}

func (m *MemoryStore) ListSnapshots(storyID uuid.UUID) ([]RuntimeSnapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []RuntimeSnapshot
	for _, snap := range m.snapshots {
		if snap.StoryID == storyID {
			result = append(result, *snap)
		}
	}
	return result, nil
}

func (m *MemoryStore) UpsertCharRuntime(storyID uuid.UUID, cr *CharRuntime) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.charRTs[storyID] == nil {
		m.charRTs[storyID] = make(map[uuid.UUID]*CharRuntime)
	}
	m.charRTs[storyID][cr.CharacterID] = cr
	return nil
}

func (m *MemoryStore) GetCharRuntime(storyID, charID uuid.UUID) (*CharRuntime, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	storyChars, ok := m.charRTs[storyID]
	if !ok {
		return nil, ErrNotFound
	}
	cr, ok := storyChars[charID]
	if !ok {
		return nil, ErrNotFound
	}
	return cr, nil
}

func (m *MemoryStore) ListCharRuntimes(storyID uuid.UUID) (map[uuid.UUID]CharRuntime, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[uuid.UUID]CharRuntime)
	storyChars, ok := m.charRTs[storyID]
	if !ok {
		return result, nil
	}
	for id, cr := range storyChars {
		result[id] = *cr
	}
	return result, nil
}
