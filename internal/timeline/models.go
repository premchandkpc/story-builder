package timeline

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID          uuid.UUID `json:"id"`
	StoryID     uuid.UUID `json:"story_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Order       int       `json:"order"`
	Timestamp   string    `json:"timestamp,omitempty"`
	Location    string    `json:"location,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (e *Event) Validate() error {
	if e == nil {
		return fmt.Errorf("timeline event is required")
	}
	if strings.TrimSpace(e.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if strings.TrimSpace(e.Description) == "" {
		return fmt.Errorf("description is required")
	}
	return nil
}

type MemoryStore struct {
	mu     sync.RWMutex
	events map[string][]*Event
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{events: make(map[string][]*Event)}
}

func (s *MemoryStore) Save(storyID uuid.UUID, event *Event) error {
	if event == nil {
		return fmt.Errorf("timeline event is required")
	}
	if err := event.Validate(); err != nil {
		return err
	}
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	event.StoryID = storyID
	now := time.Now()
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}
	event.UpdatedAt = now

	s.mu.Lock()
	defer s.mu.Unlock()
	s.events[storyID.String()] = append(s.events[storyID.String()], event)
	return nil
}

func (s *MemoryStore) List(storyID uuid.UUID) ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Event, 0, len(s.events[storyID.String()]))
	for _, event := range s.events[storyID.String()] {
		if event == nil {
			continue
		}
		clone := *event
		items = append(items, clone)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Order == items[j].Order {
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		}
		return items[i].Order < items[j].Order
	})
	return items, nil
}
