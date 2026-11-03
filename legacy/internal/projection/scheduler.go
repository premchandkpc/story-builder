package projection

import (
	"context"
	"log/slog"
	"sync"
)

type Scheduler struct {
	charProjection *CharacterProjection
	tlProjection   *TimelineProjection
	mu             sync.Mutex
	notify         chan string
}

func NewScheduler(charProjection *CharacterProjection, tlProjection *TimelineProjection) *Scheduler {
	return &Scheduler{
		charProjection: charProjection,
		tlProjection:   tlProjection,
		notify:         make(chan string, 100),
	}
}

func (s *Scheduler) Notify(storyID string) {
	select {
	case s.notify <- storyID:
	default:
	}
}

func (s *Scheduler) TriggerRebuild(ctx context.Context, storyID string) error {
	slog.Info("rebuilding projections", "storyId", storyID)
	if s.charProjection != nil {
		if err := s.charProjection.RebuildAll(ctx, storyID); err != nil {
			slog.Error("character projection rebuild failed", "storyId", storyID, "error", err)
			return err
		}
	}
	if s.tlProjection != nil {
		if _, err := s.tlProjection.EnsureLatest(ctx, storyID); err != nil {
			slog.Error("timeline projection rebuild failed", "storyId", storyID, "error", err)
			return err
		}
	}
	return nil
}
