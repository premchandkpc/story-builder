//go:build test

package event_test

import (
	"math/rand"
	"testing"

	"github.com/premchand/story-builder/internal/domain"
)

func randomEventID() string {
	return "evt_" + string(rune('A'+rand.Intn(26))) + string(rune('0'+rand.Intn(10)))
}

func generateEvents(count int) []domain.NarrativeEvent {
	events := make([]domain.NarrativeEvent, count)
	for i := 0; i < count; i++ {
		events[i] = domain.NarrativeEvent{
			ID:      randomEventID(),
			EventType: []string{"dialogue", "action", "movement", "observation"}[rand.Intn(4)],
			SubjectType: "character",
			Version:  int64(i + 1),
		}
	}
	return events
}

func TestNarrative_EventIDUniqueness(t *testing.T) {
	for n := 0; n < 50; n++ {
		events := generateEvents(20)
		seen := make(map[string]bool)
		for _, e := range events {
			if seen[e.ID] {
				t.Fatalf("iteration %d: duplicate event ID %s", n, e.ID)
			}
			seen[e.ID] = true
		}
	}
}

func TestNarrative_EventVersionMonotonic(t *testing.T) {
	for n := 0; n < 50; n++ {
		events := generateEvents(30)
		for i := 1; i < len(events); i++ {
			if events[i].Version <= events[i-1].Version {
				t.Fatalf("iteration %d: version %d <= %d at index %d", n, events[i].Version, events[i-1].Version, i)
			}
		}
	}
}
