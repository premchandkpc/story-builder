package rules

import (
	"context"
	"fmt"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/event"
)

type LocationConsistency struct {
	validLocations map[string]bool
}

func NewLocationConsistency(validLocations map[string]bool) *LocationConsistency {
	return &LocationConsistency{validLocations: validLocations}
}

func (r *LocationConsistency) Name() string { return "location_consistency" }

func (r *LocationConsistency) Validate(ctx context.Context, evt *domain.NarrativeEvent, state *event.StoryState) *event.EventViolation {
	if evt.EventType != domain.NarrativeEventTypeCharLocation {
		return nil
	}
	loc, ok := evt.Payload["location"].(string)
	if !ok || loc == "" {
		return nil
	}
	if len(r.validLocations) > 0 && !r.validLocations[loc] {
		return &event.EventViolation{
			RuleName: r.Name(),
			Severity: "warn",
			Reason:   fmt.Sprintf("location %q is not a known location", loc),
		}
	}
	return nil
}
