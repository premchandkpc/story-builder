package events

import (
	"context"
	"sync"
	"time"
)

type Event struct {
	Type      string
	StoryID   string
	SceneID   string
	GenID     string
	Data      map[string]any
	Timestamp time.Time
}

type Handler func(ctx context.Context, event Event) error

type Bus interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(eventType string, handler Handler) func()
}

type InMemoryBus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

func NewInMemoryBus() *InMemoryBus {
	return &InMemoryBus{
		handlers: make(map[string][]Handler),
	}
}

func (b *InMemoryBus) Publish(ctx context.Context, event Event) error {
	event.Timestamp = time.Now()
	b.mu.RLock()
	hh := b.handlers[event.Type]
	wildcard := b.handlers["*"]
	combined := make([]Handler, 0, len(hh)+len(wildcard))
	combined = append(combined, hh...)
	combined = append(combined, wildcard...)
	b.mu.RUnlock()

	for _, h := range combined {
		if err := h(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (b *InMemoryBus) Subscribe(eventType string, handler Handler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		hh := b.handlers[eventType]
		for i, h := range hh {
			if &h == &handler {
				b.handlers[eventType] = append(hh[:i], hh[i+1:]...)
				break
			}
		}
	}
}
