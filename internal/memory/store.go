package memory

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MemoryStore struct {
	mu      sync.RWMutex
	memories map[uuid.UUID]*Memory
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		memories: make(map[uuid.UUID]*Memory),
	}
}

func (s *MemoryStore) Create(m *Memory) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m.ID = uuid.New()
	m.CreatedAt = time.Now()
	s.memories[m.ID] = m
	return nil
}

func (s *MemoryStore) Get(id uuid.UUID) (*Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.memories[id]
	if !ok {
		return nil, ErrNotFound
	}
	return m, nil
}

func (s *MemoryStore) Search(query RetrievalQuery) ([]RankedMemory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	var candidates []Memory

	for _, m := range s.memories {
		if m.StoryID != query.StoryID {
			continue
		}
		if query.CharacterID != uuid.Nil && m.CharacterID != query.CharacterID {
			continue
		}
		if len(query.Types) > 0 {
			matched := false
			for _, t := range query.Types {
				if m.Type == t {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		candidates = append(candidates, *m)
	}

	limit := query.MaxResults
	if limit <= 0 {
		limit = 10
	}

	ranked := make([]RankedMemory, 0, len(candidates))
	for _, m := range candidates {
		sim := cosineSimilarityField(m.Summary, query.QueryText)
		score := m.Rank(sim, 1.0, 0.5, 0.3, now)
		if score < query.MinScore {
			continue
		}
		ranked = append(ranked, RankedMemory{Memory: m, Score: score})
	}

	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].Score > ranked[j].Score
	})

	if len(ranked) > limit {
		ranked = ranked[:limit]
	}

	return ranked, nil
}

func (s *MemoryStore) RetrieveRecent(storyID, characterID uuid.UUID, limit int) ([]Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var candidates []Memory
	for _, m := range s.memories {
		if m.StoryID != storyID {
			continue
		}
		if characterID != uuid.Nil && m.CharacterID != characterID {
			continue
		}
		candidates = append(candidates, *m)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].CreatedAt.After(candidates[j].CreatedAt)
	})

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	return candidates, nil
}

func (s *MemoryStore) RetrieveByType(storyID uuid.UUID, typ MemType, limit int) ([]Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Memory
	for _, m := range s.memories {
		if m.StoryID == storyID && m.Type == typ {
			result = append(result, *m)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Importance > result[j].Importance
	})

	if len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

func (s *MemoryStore) Delete(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.memories, id)
	return nil
}

func cosineSimilarityField(a, b string) float64 {
	if a == "" || b == "" {
		return 0
	}
	runes := func(s string) map[rune]float64 {
		m := make(map[rune]float64)
		for _, r := range s {
			m[r]++
		}
		return m
	}
	va := runes(a)
	vb := runes(b)

	var dot, na2, nb2 float64
	for r, c := range va {
		dot += c * vb[r]
		na2 += c * c
	}
	for _, c := range vb {
		nb2 += c * c
	}
	if na2 == 0 || nb2 == 0 {
		return 0
	}
	return dot / (math.Sqrt(na2) * math.Sqrt(nb2))
}


