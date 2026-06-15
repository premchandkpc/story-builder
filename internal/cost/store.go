package cost

import (
	"sync"

	"github.com/google/uuid"
)

type MemoryStore struct {
	mu   sync.RWMutex
	costs []GenerationCost
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (m *MemoryStore) Create(c *GenerationCost) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.costs = append(m.costs, *c)
	return nil
}

func (m *MemoryStore) GetByGeneration(generationID uuid.UUID) (*GenerationCost, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, c := range m.costs {
		if c.GenerationID == generationID {
			return &c, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) GetByStory(storyID uuid.UUID) ([]GenerationCost, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []GenerationCost
	for _, c := range m.costs {
		if c.StoryID == storyID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *MemoryStore) GetTotalByStory(storyID uuid.UUID) (int, float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var totalTokens int
	var totalCost float64
	for _, c := range m.costs {
		if c.StoryID == storyID {
			totalTokens += c.TotalTokens
			totalCost += c.CostUSD
		}
	}
	return totalTokens, totalCost, nil
}
