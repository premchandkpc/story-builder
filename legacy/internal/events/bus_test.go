package events

import (
	"context"
	"errors"
	"sync"
	"testing"
)

var errBang = errors.New("handler error")

func TestPublishSubscribe(t *testing.T) {
	bus := NewInMemoryBus()
	ctx := context.Background()

	var mu sync.Mutex
	var received []Event

	bus.Subscribe("test.event", func(_ context.Context, evt Event) error {
		mu.Lock()
		received = append(received, evt)
		mu.Unlock()
		return nil
	})

	err := bus.Publish(ctx, Event{Type: "test.event", StoryID: "s1"})
	if err != nil {
		t.Fatal(err)
	}

	if len(received) != 1 {
		t.Fatalf("expected 1 event, got %d", len(received))
	}
	if received[0].Type != "test.event" {
		t.Fatalf("expected test.event, got %s", received[0].Type)
	}
	if received[0].StoryID != "s1" {
		t.Fatalf("expected s1, got %s", received[0].StoryID)
	}
}

func TestPublishNoSubscribers(t *testing.T) {
	bus := NewInMemoryBus()
	err := bus.Publish(context.Background(), Event{Type: "nonexistent"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := NewInMemoryBus()
	ctx := context.Background()

	var count int
	var mu sync.Mutex

	for i := 0; i < 3; i++ {
		bus.Subscribe("multi", func(_ context.Context, evt Event) error {
			mu.Lock()
			count++
			mu.Unlock()
			return nil
		})
	}

	bus.Publish(ctx, Event{Type: "multi"})

	if count != 3 {
		t.Fatalf("expected 3 handler calls, got %d", count)
	}
}

func TestWildcardHandler(t *testing.T) {
	bus := NewInMemoryBus()
	ctx := context.Background()

	var events []string
	var mu sync.Mutex

	bus.Subscribe("*", func(_ context.Context, evt Event) error {
		mu.Lock()
		events = append(events, evt.Type)
		mu.Unlock()
		return nil
	})

	bus.Publish(ctx, Event{Type: "a"})
	bus.Publish(ctx, Event{Type: "b"})

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := NewInMemoryBus()
	ctx := context.Background()

	var count int
	var mu sync.Mutex

	unsub := bus.Subscribe("sub", func(_ context.Context, evt Event) error {
		mu.Lock()
		count++
		mu.Unlock()
		return nil
	})

	bus.Publish(ctx, Event{Type: "sub"})
	unsub()
	bus.Publish(ctx, Event{Type: "sub"})

	if count != 1 {
		t.Fatalf("expected 1 event after unsubscribe, got %d", count)
	}
}

func TestHandlerError(t *testing.T) {
	bus := NewInMemoryBus()
	bus.Subscribe("err", func(_ context.Context, evt Event) error {
		return errBang
	})

	err := bus.Publish(context.Background(), Event{Type: "err"})
	if err == nil {
		t.Fatal("expected error")
	}
}
