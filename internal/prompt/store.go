package prompt

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type MemoryStore struct {
	mu       sync.RWMutex
	prompts  map[uuid.UUID]*Prompt
	byName   map[string]*Prompt
	versions map[uuid.UUID][]PromptVersion
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		prompts:  make(map[uuid.UUID]*Prompt),
		byName:   make(map[string]*Prompt),
		versions: make(map[uuid.UUID][]PromptVersion),
	}
}

func (m *MemoryStore) Create(p *Prompt) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p.ID = uuid.New()
	p.CurrentVer = 1
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	m.prompts[p.ID] = p
	m.byName[p.Name] = p
	return nil
}

func (m *MemoryStore) Get(id uuid.UUID) (*Prompt, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.prompts[id]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (m *MemoryStore) GetByName(name string) (*Prompt, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.byName[name]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (m *MemoryStore) List() ([]Prompt, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []Prompt
	for _, p := range m.prompts {
		result = append(result, *p)
	}
	return result, nil
}

func (m *MemoryStore) Update(p *Prompt) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p.UpdatedAt = time.Now()
	m.prompts[p.ID] = p
	m.byName[p.Name] = p
	return nil
}

func (m *MemoryStore) CreateVersion(pv *PromptVersion) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	pv.ID = uuid.New()
	pv.CreatedAt = time.Now()
	p, ok := m.prompts[pv.PromptID]
	if !ok {
		return ErrNotFound
	}
	pv.Version = p.CurrentVer + 1
	p.CurrentVer = pv.Version
	m.versions[pv.PromptID] = append(m.versions[pv.PromptID], *pv)
	return nil
}

func (m *MemoryStore) ListVersions(promptID uuid.UUID) ([]PromptVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	versions, ok := m.versions[promptID]
	if !ok {
		return nil, nil
	}
	result := make([]PromptVersion, len(versions))
	copy(result, versions)
	return result, nil
}

func (m *MemoryStore) GetVersion(promptID uuid.UUID, version int) (*PromptVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	versions, ok := m.versions[promptID]
	if !ok {
		return nil, ErrNotFound
	}
	for _, v := range versions {
		if v.Version == version {
			return &v, nil
		}
	}
	return nil, ErrNotFound
}
