package worker

import (
	"context"
	"log/slog"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type TimelineWorker struct {
	tlRepo   repository.TimelineRepository
	edgeRepo repository.SceneEdgeRepository
}

func NewTimelineWorker(tlRepo repository.TimelineRepository, edgeRepo repository.SceneEdgeRepository) *TimelineWorker {
	return &TimelineWorker{tlRepo: tlRepo, edgeRepo: edgeRepo}
}

type TimelineArgs struct {
	StoryID     string
	SceneID     string
	Title       string
	Description string
	EventType   string
	Order       int
}

func (w *TimelineWorker) Work(ctx context.Context, args TimelineArgs) error {
	slog.Info("recording timeline event", "sceneId", args.SceneID, "order", args.Order)

	// Find incoming edges (predecessor scenes) and wire as dependencies.
	deps := []string{}
	if w.edgeRepo != nil && args.SceneID != "" {
		inEdges, err := w.edgeRepo.ListTo(ctx, args.SceneID)
		if err == nil {
			for _, e := range inEdges {
				deps = append(deps, e.FromSceneID)
			}
		}
	}

	eventType := args.EventType
	if eventType == "" {
		eventType = domain.TimelineEventScene
	}

	event := &domain.TimelineEvent{
		StoryID:      args.StoryID,
		SceneID:      args.SceneID,
		Title:        args.Title,
		EventType:    eventType,
		Description:  args.Description,
		Dependencies: deps,
		Order:        args.Order,
	}

	return w.tlRepo.Create(ctx, event)
}
