package rules

import (
	"context"
	"fmt"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/event"
)

type DeadCharacterCannotAct struct{}

func (r *DeadCharacterCannotAct) Name() string { return "dead_character_cannot_act" }

func (r *DeadCharacterCannotAct) Validate(ctx context.Context, evt *domain.NarrativeEvent, state *event.StoryState) *event.EventViolation {
	if evt.SubjectType != domain.NarrativeSubjectChar {
		return nil
	}
	if evt.EventType != domain.NarrativeEventTypeCharLocation && evt.EventType != domain.NarrativeEventTypeCharGoal {
		return nil
	}
	charState, ok := state.Characters[evt.SubjectID]
	if !ok {
		return &event.EventViolation{
			RuleName: r.Name(),
			Severity: "reject",
			Reason:   fmt.Sprintf("character %s not found in story state", evt.SubjectID),
		}
	}
	if charState.Health <= 0 {
		return &event.EventViolation{
			RuleName: r.Name(),
			Severity: "reject",
			Reason:   fmt.Sprintf("character %s is dead (health=%d)", evt.SubjectID, charState.Health),
		}
	}
	return nil
}
