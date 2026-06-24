package projection

import (
	"context"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type TimelineView struct {
	StoryID   string          `bson:"_id"`
	Events    []TimelineEntry `bson:"events"`
	LastOrder int             `bson:"lastOrder"`
	Version   int64           `bson:"version"`
	UpdatedAt time.Time       `bson:"updatedAt"`
}

type TimelineEntry struct {
	EventID   string    `bson:"eventId"`
	SceneID   string    `bson:"sceneId"`
	Order     int       `bson:"order"`
	Summary   string    `bson:"summary"`
	CreatedAt time.Time `bson:"createdAt"`
}

type TimelineProjection struct {
	EventRepo    repository.NarrativeEventRepository
	TimelineRepo repository.TimelineRepository
}

func NewTimelineProjection(eventRepo repository.NarrativeEventRepository, tlRepo repository.TimelineRepository) *TimelineProjection {
	return &TimelineProjection{EventRepo: eventRepo, TimelineRepo: tlRepo}
}

func (p *TimelineProjection) EnsureLatest(ctx context.Context, storyID string) (*TimelineView, error) {
	latestVersion, err := p.EventRepo.LatestVersion(ctx, storyID)
	if err != nil {
		return nil, err
	}

	tlEvents, err := p.TimelineRepo.ListByStory(ctx, storyID)
	if err != nil {
		return nil, err
	}

	view := &TimelineView{
		StoryID:   storyID,
		Events:    make([]TimelineEntry, 0, len(tlEvents)),
		Version:   latestVersion,
		UpdatedAt: time.Now(),
	}

	for _, e := range tlEvents {
		if e == nil {
			continue
		}
		view.Events = append(view.Events, TimelineEntry{
			EventID:   e.ID,
			SceneID:   e.SceneID,
			Order:     e.Order,
			Summary:   e.Title,
			CreatedAt: e.CreatedAt,
		})
		if e.Order > view.LastOrder {
			view.LastOrder = e.Order
		}
	}

	return view, nil
}
