package event

import (
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MemoryStore struct {
	mu     sync.RWMutex
	events []Event
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (m *MemoryStore) Append(evt *Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	evt.ID = uuid.New()
	evt.Timestamp = time.Now()
	m.events = append(m.events, *evt)
	return nil
}

func (m *MemoryStore) GetByAggregate(aggregateID uuid.UUID, evtType EventType) ([]Event, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []Event
	for _, e := range m.events {
		if e.AggregateID == aggregateID && (evtType == "" || e.Type == evtType) {
			result = append(result, e)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})
	return result, nil
}

func (m *MemoryStore) GetByStory(storyID uuid.UUID, evtType EventType, limit int) ([]Event, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []Event
	for _, e := range m.events {
		if e.StoryID == storyID && (evtType == "" || e.Type == evtType) {
			result = append(result, e)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})
	if limit > 0 && len(result) > limit {
		result = result[len(result)-limit:]
	}
	return result, nil
}

func (m *MemoryStore) GetByType(evtType EventType, since time.Time, limit int) ([]Event, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []Event
	for _, e := range m.events {
		if e.Type == evtType && (since.IsZero() || e.Timestamp.After(since)) {
			result = append(result, e)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})
	if limit > 0 && len(result) > limit {
		result = result[len(result)-limit:]
	}
	return result, nil
}

func (m *MemoryStore) Replay(aggregateID uuid.UUID) ([]Event, error) {
	return m.GetByAggregate(aggregateID, "")
}

type MemoryBus struct {
	mu       sync.RWMutex
	handlers map[EventType][]func(*Event) error
	events   chan *Event
	stop     chan struct{}
}

func NewMemoryBus() *MemoryBus {
	return &MemoryBus{
		handlers: make(map[EventType][]func(*Event) error),
		events:   make(chan *Event, 100),
		stop:     make(chan struct{}),
	}
}

func (b *MemoryBus) Publish(evt *Event) error {
	b.events <- evt
	return nil
}

func (b *MemoryBus) Subscribe(evtType EventType, handler func(*Event) error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[evtType] = append(b.handlers[evtType], handler)
}

func (b *MemoryBus) Start() {
	go func() {
		for {
			select {
			case evt := <-b.events:
				b.mu.RLock()
				handlers := b.handlers[evt.Type]
				b.mu.RUnlock()
				for _, h := range handlers {
					h(evt)
				}
			case <-b.stop:
				return
			}
		}
	}()
}

func (b *MemoryBus) Stop() {
	close(b.stop)
}
