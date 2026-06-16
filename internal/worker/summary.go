package worker

import (
	"context"
	"log/slog"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
)

type SummaryWorker struct {
	summary  llm.SummaryService
	sumRepo  SummaryWriter
}

type SummaryWriter interface {
	Upsert(ctx context.Context, s *domain.Summary) error
}

func NewSummaryWorker(summary llm.SummaryService, sumRepo SummaryWriter) *SummaryWorker {
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
		Level:   domain.SummaryLevelScene,
		Content: content,
	}

	return w.sumRepo.Upsert(ctx, sum)
}
