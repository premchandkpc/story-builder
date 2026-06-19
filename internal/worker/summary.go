package worker

import (
	"context"
	"log/slog"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/repository"
)

type SummaryWorker struct {
	summary llm.SummaryService
	sumRepo repository.SummaryRepository
}

func NewSummaryWorker(summary llm.SummaryService, sumRepo repository.SummaryRepository) *SummaryWorker {
	return &SummaryWorker{summary: summary, sumRepo: sumRepo}
}

type SummaryArgs struct {
	StoryID         string
	SceneID         string
	PreviousSummary string
	AcceptedScene   string
}

func (w *SummaryWorker) Work(ctx context.Context, args SummaryArgs) error {
	slog.Info("updating summary", "sceneId", args.SceneID)

	content, err := w.summary.UpdateSummary(ctx, args.PreviousSummary, args.AcceptedScene)
	if err != nil {
		return err
	}

	sum := &domain.Summary{
		StoryID: args.StoryID,
		SceneID: args.SceneID,
		Level:   domain.SummaryLevelStory,
		Content: content,
	}

	return w.sumRepo.Upsert(ctx, sum)
}
