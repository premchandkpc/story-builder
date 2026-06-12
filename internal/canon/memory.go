package canon

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MemoryStore struct {
	mu          sync.RWMutex
	characters  []Character
	locations   []Location
	lore        []Lore
	charNextVer map[uuid.UUID]int
	locNextVer  map[uuid.UUID]int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		charNextVer: make(map[uuid.UUID]int),
		locNextVer:  make(map[uuid.UUID]int),
	}
}

func (s *MemoryStore) Create(name string, traits, voiceSamples []string, relationships map[string]string) (*Character, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := Character{
		ID:            uuid.New(),
		Version:       1,
		Name:          name,
		Traits:        traits,
		VoiceSamples:  voiceSamples,
		Relationships: relationships,
		CreatedAt:     time.Now(),
	}
	s.characters = append(s.characters, c)
	s.charNextVer[c.ID] = 2
	return &c, nil
}

func (s *MemoryStore) GetLatest(id uuid.UUID) (*Character, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var latest *Character
	for i := range s.characters {
		if s.characters[i].ID == id {
			if latest == nil || s.characters[i].Version > latest.Version {
				latest = &s.characters[i]
			}
		}
	}
	if latest == nil {
		return nil, fmt.Errorf("character %s not found", id)
	}
	return latest, nil
}

func (s *MemoryStore) Get(id uuid.UUID, version int) (*Character, error) {
	if version == 0 {
		return s.GetLatest(id)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.characters {
		if s.characters[i].ID == id && s.characters[i].Version == version {
			return &s.characters[i], nil
		}
	}
	return nil, fmt.Errorf("character %s v%d not found", id, version)
}

func (s *MemoryStore) Update(id uuid.UUID, traits, voiceSamples []string, relationships map[string]string) (*Character, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	latest := 0
	for i := range s.characters {
		if s.characters[i].ID == id && s.characters[i].Version > latest {
			latest = s.characters[i].Version
		}
	}
	if latest == 0 {
		return nil, fmt.Errorf("character %s not found", id)
	}
	c := Character{
		ID:            id,
		Version:       latest + 1,
		Name:          s.characters[0].Name,
		Traits:        traits,
		VoiceSamples:  voiceSamples,
		Relationships: relationships,
		CreatedAt:     time.Now(),
	}
	s.characters = append(s.characters, c)
	return &c, nil
}

func (s *MemoryStore) List() ([]Character, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	latest := make(map[uuid.UUID]Character)
	for _, c := range s.characters {
		if existing, ok := latest[c.ID]; !ok || c.Version > existing.Version {
			latest[c.ID] = c
		}
	}
	result := make([]Character, 0, len(latest))
	for _, c := range latest {
		result = append(result, c)
	}
	return result, nil
}

func (s *MemoryStore) CreateLocation(name, description string, props []string) (*Location, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	l := Location{
		ID:          uuid.New(),
		Version:     1,
		Name:        name,
		Description: description,
		Props:       props,
		CreatedAt:   time.Now(),
	}
	s.locations = append(s.locations, l)
	s.locNextVer[l.ID] = 2
	return &l, nil
}

func (s *MemoryStore) GetLocationLatest(id uuid.UUID) (*Location, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var latest *Location
	for i := range s.locations {
		if s.locations[i].ID == id {
			if latest == nil || s.locations[i].Version > latest.Version {
				latest = &s.locations[i]
			}
		}
	}
	if latest == nil {
		return nil, fmt.Errorf("location %s not found", id)
	}
	return latest, nil
}

func (s *MemoryStore) GetLocation(id uuid.UUID, version int) (*Location, error) {
	if version == 0 {
		return s.GetLocationLatest(id)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.locations {
		if s.locations[i].ID == id && s.locations[i].Version == version {
			return &s.locations[i], nil
		}
	}
	return nil, fmt.Errorf("location %s v%d not found", id, version)
}

func (s *MemoryStore) UpdateLocation(id uuid.UUID, description string, props []string) (*Location, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	latest := 0
	for i := range s.locations {
		if s.locations[i].ID == id && s.locations[i].Version > latest {
			latest = s.locations[i].Version
		}
	}
	if latest == 0 {
		return nil, fmt.Errorf("location %s not found", id)
	}
	var currentName string
	for _, loc := range s.locations {
		if loc.ID == id {
			currentName = loc.Name
		}
	}
	l := Location{
		ID:          id,
		Version:     latest + 1,
		Name:        currentName,
		Description: description,
		Props:       props,
		CreatedAt:   time.Now(),
	}
	s.locations = append(s.locations, l)
	return &l, nil
}

func (s *MemoryStore) ListLocations() ([]Location, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	latest := make(map[uuid.UUID]Location)
	for _, l := range s.locations {
		if existing, ok := latest[l.ID]; !ok || l.Version > existing.Version {
			latest[l.ID] = l
		}
	}
	result := make([]Location, 0, len(latest))
	for _, l := range latest {
		result = append(result, l)
	}
	return result, nil
}

func (s *MemoryStore) CreateLore(tags []string, content string) (*Lore, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	l := Lore{
		ID:        uuid.New(),
		Tags:      tags,
		Content:   content,
		CreatedAt: time.Now(),
	}
	s.lore = append(s.lore, l)
	return &l, nil
}

func (s *MemoryStore) ListLore() ([]Lore, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r := make([]Lore, len(s.lore))
	copy(r, s.lore)
	return r, nil
}

func (s *MemoryStore) SearchByTags(tags []string) ([]Lore, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tagSet := make(map[string]bool, len(tags))
	for _, t := range tags {
		tagSet[t] = true
	}
	var result []Lore
	for _, l := range s.lore {
		for _, t := range l.Tags {
			if tagSet[t] {
				result = append(result, l)
				break
			}
		}
	}
	return result, nil
}

func (s *MemoryStore) SearchSimilar(_ []float32, limit int) ([]Lore, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit > len(s.lore) {
		limit = len(s.lore)
	}
	return s.lore[:limit], nil
}
