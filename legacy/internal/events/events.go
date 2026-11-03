package events

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
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

type handlerEntry struct {
	id   int64
	fn   Handler
}

type InMemoryBus struct {
	mu        sync.RWMutex
	handlers  map[string][]handlerEntry
	nextID    atomic.Int64
}

func NewInMemoryBus() *InMemoryBus {
	return &InMemoryBus{
		handlers: make(map[string][]handlerEntry),
	}
}

func (b *InMemoryBus) Publish(ctx context.Context, event Event) error {
	event.Timestamp = time.Now()
	b.mu.RLock()
	hh := b.handlers[event.Type]
	wildcard := b.handlers["*"]
	combined := make([]handlerEntry, 0, len(hh)+len(wildcard))
	combined = append(combined, hh...)
	combined = append(combined, wildcard...)
	b.mu.RUnlock()

	var mu sync.Mutex
	var firstErr error
	var wg sync.WaitGroup
	for _, entry := range combined {
		entry := entry
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("event handler panic: %v", r)
					}
					mu.Unlock()
				}
			}()
			if err := entry.fn(ctx, event); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return firstErr
}

func (b *InMemoryBus) Subscribe(eventType string, handler Handler) func() {
	id := b.nextID.Add(1)
	b.mu.Lock()
	b.handlers[eventType] = append(b.handlers[eventType], handlerEntry{id: id, fn: handler})
	b.mu.Unlock()

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		hh := b.handlers[eventType]
		for i, entry := range hh {
			if entry.id == id {
				b.handlers[eventType] = append(hh[:i], hh[i+1:]...)
				break
			}
		}
	}
}
