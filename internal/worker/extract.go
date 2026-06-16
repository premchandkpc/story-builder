package worker

import (
	"context"
	"log/slog"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
)

type ExtractStateWorker struct {
	extract   llm.ExtractionService
	stateRepo StateWriter
}

type StateWriter interface {
	Append(ctx context.Context, s *domain.CharacterState) error
}

func NewExtractStateWorker(extract llm.ExtractionService, stateRepo StateWriter) *ExtractStateWorker {
	return &ExtractStateWorker{extract: extract, stateRepo: stateRepo}
}

type ExtractStateArgs struct {
	StoryID       string
	SceneID       string
	SceneText     string
	CharacterRefs []string
}

func (w *ExtractStateWorker) Work(ctx context.Context, args ExtractStateArgs) error {
	slog.Info("extracting state", "sceneId", args.SceneID)

	roster := make(map[string]string)
	for _, ref := range args.CharacterRefs {
		roster[ref] = ref
	}

	deltas, err := w.extract.ExtractState(ctx, args.SceneText, roster)
	if err != nil {
		return err
	}

	for _, delta := range deltas.Deltas {
		changes := map[string]any{
			"new_location":          delta.NewLocation,
			"learned":               delta.Learned,
			"mood":                  delta.Mood,
			"relationship_changes":  delta.RelationshipChanges,
			"items_gained":          delta.ItemsGained,
			"items_lost":            delta.ItemsLost,
		}
		state := &domain.CharacterState{
			CharacterID: delta.Character,
			StoryID:     args.StoryID,
			SceneID:     args.SceneID,
			Changes:     changes,
		}
		if err := w.stateRepo.Append(ctx, state); err != nil {
			return err
		}
	}

	return nil
}
