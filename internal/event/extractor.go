package event

import (
	"context"

	"github.com/premchand/story-builder/internal/domain"
)

type EventExtractor struct{}

func NewEventExtractor() *EventExtractor {
	return &EventExtractor{}
}

type ExtractorConfig struct {
	StoryID string
	SceneID string
	RunID   string
	GenID   string
}

func (e *EventExtractor) ExtractFromStates(ctx context.Context, cfg ExtractorConfig, states []domain.CharacterState) []domain.NarrativeEvent {
	var events []domain.NarrativeEvent
	for _, s := range states {
		events = append(events, e.extractCharacterEvents(cfg, s)...)
	}
	return events
}

func (e *EventExtractor) extractCharacterEvents(cfg ExtractorConfig, s domain.CharacterState) []domain.NarrativeEvent {
	var events []domain.NarrativeEvent

	if location, ok := s.Changes["location"]; ok {
		if locStr, ok := location.(string); ok && locStr != "" {
			events = append(events, domain.NarrativeEvent{
				StoryID:     cfg.StoryID,
				SceneID:     cfg.SceneID,
				SourceRunID: cfg.RunID,
				EventType:   domain.NarrativeEventTypeCharLocation,
				SubjectType: domain.NarrativeSubjectChar,
				SubjectID:   s.CharacterID,
				Payload:     map[string]any{"location": locStr, "previous": s.Location},
				Confidence:  0.9,
			})
		}
	}

	if mood, ok := s.Changes["mood"]; ok {
		if moodStr, ok := mood.(string); ok && moodStr != "" {
			events = append(events, domain.NarrativeEvent{
				StoryID:     cfg.StoryID,
				SceneID:     cfg.SceneID,
				SourceRunID: cfg.RunID,
				EventType:   domain.NarrativeEventTypeCharEmotion,
				SubjectType: domain.NarrativeSubjectChar,
				SubjectID:   s.CharacterID,
				Payload:     map[string]any{"mood": moodStr},
				Confidence:  0.8,
			})
		}
	}

	if goal, ok := s.Changes["activeGoal"]; ok {
		if goalStr, ok := goal.(string); ok && goalStr != "" {
			events = append(events, domain.NarrativeEvent{
				StoryID:     cfg.StoryID,
				SceneID:     cfg.SceneID,
				SourceRunID: cfg.RunID,
				EventType:   domain.NarrativeEventTypeCharGoal,
				SubjectType: domain.NarrativeSubjectChar,
				SubjectID:   s.CharacterID,
				Payload:     map[string]any{"goal": goalStr},
				Confidence:  0.7,
			})
		}
	}

	if learned, ok := s.Changes["learned"]; ok {
		learnedList := toStringSlice(learned)
		for _, k := range learnedList {
			events = append(events, domain.NarrativeEvent{
				StoryID:     cfg.StoryID,
				SceneID:     cfg.SceneID,
				SourceRunID: cfg.RunID,
				EventType:   domain.NarrativeEventTypeCharKnowledge,
				SubjectType: domain.NarrativeSubjectChar,
				SubjectID:   s.CharacterID,
				Payload:     map[string]any{"knowledge": k},
				Confidence:  0.6,
			})
		}
	}

	if rel, ok := s.Changes["relationships"]; ok {
		if relMap, ok := rel.(map[string]any); ok {
			for targetID, change := range relMap {
				if changeMap, ok := change.(map[string]any); ok {
					if trust, ok := changeMap["trust"].(float64); ok {
						events = append(events, domain.NarrativeEvent{
							StoryID:     cfg.StoryID,
							SceneID:     cfg.SceneID,
							SourceRunID: cfg.RunID,
							EventType:   domain.NarrativeEventTypeRelTrust,
							SubjectType: domain.NarrativeSubjectRel,
							SubjectID:   s.CharacterID + ":" + targetID,
							Payload:     map[string]any{"targetId": targetID, "trust": trust},
							Confidence:  0.5,
						})
					}
				}
			}
		}
	}

	return events
}

func toStringSlice(v any) []string {
	switch s := v.(type) {
	case []string:
		return s
	case []any:
		out := make([]string, 0, len(s))
		for _, item := range s {
			if str, ok := item.(string); ok {
				out = append(out, str)
			}
		}
		return out
	default:
		return nil
	}
}
