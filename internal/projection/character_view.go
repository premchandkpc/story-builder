package projection

import (
	"context"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type CharacterProjection struct {
	EventRepo repository.NarrativeEventRepository
	ViewRepo  repository.CharacterViewRepository
}

func NewCharacterProjection(eventRepo repository.NarrativeEventRepository, viewRepo repository.CharacterViewRepository) *CharacterProjection {
	return &CharacterProjection{EventRepo: eventRepo, ViewRepo: viewRepo}
}

func (p *CharacterProjection) EnsureLatest(ctx context.Context, storyID, charID string) (*domain.CharacterView, error) {
	view, err := p.ViewRepo.Get(ctx, charID)
	if err != nil {
		return nil, err
	}

	latestVersion, err := p.EventRepo.LatestVersion(ctx, storyID)
	if err != nil {
		return nil, err
	}

	if view != nil && view.Version >= latestVersion {
		return view, nil
	}

	return p.rebuild(ctx, storyID, charID)
}

func (p *CharacterProjection) RebuildAll(ctx context.Context, storyID string) error {
	charIDs := collectCharIDsFromEvents(ctx, p.EventRepo, storyID)
	for _, charID := range charIDs {
		if _, err := p.rebuild(ctx, storyID, charID); err != nil {
			return err
		}
	}
	return nil
}

func (p *CharacterProjection) rebuild(ctx context.Context, storyID, charID string) (*domain.CharacterView, error) {
	events, err := p.EventRepo.ListBySubject(ctx, storyID, charID, 0)
	if err != nil {
		return nil, err
	}

	view := &domain.CharacterView{
		CharacterID: charID,
		StoryID:     storyID,
		CurrentState: domain.CharacterStateSnapshot{
			Relationships: []domain.RelSnapshot{},
			Knowledge:     []string{},
		},
		EventIDs:  make([]string, 0, len(events)),
		UpdatedAt: time.Now(),
	}

	for _, e := range events {
		if e == nil {
			continue
		}
		view.EventIDs = append(view.EventIDs, e.ID)
		applyEvent(view, e)
	}

	if len(events) > 0 {
		view.Version = events[len(events)-1].Version
	}

	if err := p.ViewRepo.Upsert(ctx, view); err != nil {
		return nil, err
	}
	return view, nil
}

func applyEvent(view *domain.CharacterView, event *domain.NarrativeEvent) {
	payload := event.Payload
	if payload == nil {
		return
	}

	switch event.EventType {
	case domain.NarrativeEventTypeCharLocation:
		if loc, ok := payload["location"].(string); ok {
			view.CurrentState.Location = loc
		}
	case domain.NarrativeEventTypeCharEmotion:
		if mood, ok := payload["mood"].(string); ok {
			view.CurrentState.EmotionalState = mood
			view.CurrentState.Mood = mood
		}
	case domain.NarrativeEventTypeCharGoal:
		if goal, ok := payload["goal"].(string); ok {
			view.CurrentState.ActiveGoal = goal
		}
	case domain.NarrativeEventTypeCharKnowledge:
		if knowledge, ok := payload["knowledge"].(string); ok && knowledge != "" {
			seen := false
			for _, k := range view.CurrentState.Knowledge {
				if k == knowledge {
					seen = true
					break
				}
			}
			if !seen {
				view.CurrentState.Knowledge = append(view.CurrentState.Knowledge, knowledge)
			}
		}
	case domain.NarrativeEventTypeRelTrust:
		if targetID, ok := payload["targetId"].(string); ok {
			trust, _ := payload["trust"].(float64)
			found := false
			for i, rel := range view.CurrentState.Relationships {
				if rel.TargetID == targetID {
					view.CurrentState.Relationships[i].Trust = trust
					found = true
					break
				}
			}
			if !found {
				view.CurrentState.Relationships = append(view.CurrentState.Relationships, domain.RelSnapshot{
					TargetID: targetID,
					Trust:    trust,
				})
			}
		}
	}
}

func collectCharIDsFromEvents(ctx context.Context, eventRepo repository.NarrativeEventRepository, storyID string) []string {
	events, err := eventRepo.ListByStory(ctx, storyID, 10000)
	if err != nil {
		return nil
	}
	seen := make(map[string]bool)
	var ids []string
	for _, e := range events {
		if e.SubjectType == domain.NarrativeSubjectChar && !seen[e.SubjectID] {
			seen[e.SubjectID] = true
			ids = append(ids, e.SubjectID)
		}
	}
	return ids
}
