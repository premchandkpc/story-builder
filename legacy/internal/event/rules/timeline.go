package rules

import (
	"context"
	"fmt"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/event"
)

type TimelineMonotonicity struct{}

func (r *TimelineMonotonicity) Name() string { return "timeline_monotonicity" }

func (r *TimelineMonotonicity) Validate(ctx context.Context, evt *domain.NarrativeEvent, state *event.StoryState) *event.EventViolation {
	if evt.EventType != domain.NarrativeEventTypeTimeline {
		return nil
	}
	sceneOrder := state.Scene.TimelinePosition
	if len(state.Timeline) > 0 {
		lastOrder := state.Timeline[len(state.Timeline)-1].Order
		if sceneOrder < lastOrder {
			return &event.EventViolation{
				RuleName: r.Name(),
				Severity: "reject",
				Reason:   fmt.Sprintf("scene order %d precedes last timeline event order %d", sceneOrder, lastOrder),
			}
		}
	}
	return nil
}
