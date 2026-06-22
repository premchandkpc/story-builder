package worker

import (
	"context"
	"log/slog"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type TimelineWorker struct {
	tlRepo    repository.TimelineRepository
	edgeRepo  repository.SceneEdgeRepository
	bibleRepo repository.BibleRepository
}

func NewTimelineWorker(tlRepo repository.TimelineRepository, edgeRepo repository.SceneEdgeRepository, bibleRepo repository.BibleRepository) *TimelineWorker {
	return &TimelineWorker{tlRepo: tlRepo, edgeRepo: edgeRepo, bibleRepo: bibleRepo}
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

	// Auto-populate RelatedStoryIDs from bible sharing: if this story's bible
	// is shared with other stories, its timeline events propagate to them.
	var relatedIDs []string
	if w.bibleRepo != nil {
		if bibles, err := w.bibleRepo.ListByReferencingStory(ctx, args.StoryID); err == nil {
			for _, b := range bibles {
				relatedIDs = append(relatedIDs, b.StoryID)
			}
		}
	}

	event := &domain.TimelineEvent{
		StoryID:         args.StoryID,
		SceneID:         args.SceneID,
		Title:           args.Title,
		EventType:       eventType,
		Description:     args.Description,
		Dependencies:    deps,
		RelatedStoryIDs: relatedIDs,
		Order:           args.Order,
	}

	return w.tlRepo.Create(ctx, event)
}
