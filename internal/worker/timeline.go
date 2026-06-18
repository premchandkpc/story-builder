package worker

import (
	"context"
	"log/slog"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type TimelineWorker struct {
	tlRepo repository.TimelineRepository
}

func NewTimelineWorker(tlRepo repository.TimelineRepository) *TimelineWorker {
	return &TimelineWorker{tlRepo: tlRepo}
}

type TimelineArgs struct {
	StoryID     string
	SceneID     string
	Title       string
	Description string
	Order       int
}

func (w *TimelineWorker) Work(ctx context.Context, args TimelineArgs) error {
	slog.Info("recording timeline event", "sceneId", args.SceneID, "order", args.Order)

	event := &domain.TimelineEvent{
		StoryID:     args.StoryID,
		SceneID:     args.SceneID,
		Title:       args.Title,
		Description: args.Description,
		Order:       args.Order,
	}

	return w.tlRepo.Create(ctx, event)
}
