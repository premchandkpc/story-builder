package rules

import (
	"context"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/event"
)

type DuplicateDetector struct{}

func (r *DuplicateDetector) Name() string { return "duplicate_detector" }

func (r *DuplicateDetector) Validate(ctx context.Context, evt *domain.NarrativeEvent, state *event.StoryState) *event.EventViolation {
	if evt.EventType != domain.NarrativeEventTypeCharKnowledge {
		return nil
	}
	knowledge, ok := evt.Payload["knowledge"].(string)
	if !ok || knowledge == "" {
		return nil
	}
	charState, ok := state.Characters[evt.SubjectID]
	if !ok {
		return nil
	}
	for _, existing := range charState.Knowledge {
		if existing == knowledge {
			return &event.EventViolation{
				RuleName: r.Name(),
				Severity: "warn",
				Reason:   "duplicate knowledge event, character already knows this",
			}
		}
	}
	return nil
}
