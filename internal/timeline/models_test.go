package timeline

import (
	"testing"

	"github.com/google/uuid"
)

func TestEventValidate(t *testing.T) {
	event := &Event{StoryID: uuid.New(), Title: "Arrival", Description: "The hero enters the city", Order: 1}
	if err := event.Validate(); err != nil {
		t.Fatalf("expected valid event, got %v", err)
	}
}

func TestEventValidateRequiresTitle(t *testing.T) {
	event := &Event{StoryID: uuid.New(), Description: "Missing title", Order: 1}
	if err := event.Validate(); err == nil {
		t.Fatal("expected validation error for missing title")
	}
}

func TestMemoryStoreSaveAndList(t *testing.T) {
	store := NewMemoryStore()
	storyID := uuid.New()
	if err := store.Save(storyID, &Event{Title: "Opening", Description: "The story begins", Order: 1}); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	list, err := store.List(storyID)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 event, got %d", len(list))
	}
}
