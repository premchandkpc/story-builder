package event

import (
	"context"

	"github.com/premchand/story-builder/internal/domain"
)

type EventViolation struct {
	RuleName string
	Severity string
	Reason   string
}

type EventValidationRule interface {
	Name() string
	Validate(ctx context.Context, event *domain.NarrativeEvent, state *StoryState) *EventViolation
}

type StoryState struct {
	Characters map[string]*domain.CharacterState
	Timeline   []domain.TimelineEvent
	Scene      *domain.Scene
}

type EventValidator struct {
	rules []EventValidationRule
}

func NewEventValidator(rules []EventValidationRule) *EventValidator {
	return &EventValidator{rules: rules}
}

func (v *EventValidator) Rules() []EventValidationRule {
	return v.rules
}

func (v *EventValidator) Filter(ctx context.Context, candidates []domain.NarrativeEvent, state *StoryState) (accepted, rejected []domain.NarrativeEvent) {
	for _, event := range candidates {
		var hasViolation bool
		for _, rule := range v.rules {
			if violation := rule.Validate(ctx, &event, state); violation != nil {
				if violation.Severity == "reject" {
					hasViolation = true
				}
			}
		}
		if hasViolation {
			rejected = append(rejected, event)
		} else {
			accepted = append(accepted, event)
		}
	}
	return
}
