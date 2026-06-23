package rules

import (
	"context"
	"fmt"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/event"
)

type ValueBounds struct{}

func (r *ValueBounds) Name() string { return "value_bounds" }

func (r *ValueBounds) Validate(ctx context.Context, evt *domain.NarrativeEvent, state *event.StoryState) *event.EventViolation {
	if evt.EventType == domain.NarrativeEventTypeRelTrust {
		trust, ok := evt.Payload["trust"].(float64)
		if !ok {
			return nil
		}
		if trust < -1.0 || trust > 1.0 {
			return &event.EventViolation{
				RuleName: r.Name(),
				Severity: "warn",
				Reason:   fmt.Sprintf("trust value %.2f out of range [-1.0, 1.0]", trust),
			}
		}
	}
	return nil
}
