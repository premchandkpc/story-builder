package worker

import (
	"context"
	"log/slog"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/repository"
)

type ExtractStateWorker struct {
	extract   llm.ExtractionService
	stateRepo repository.CharacterStateRepository
}

func NewExtractStateWorker(extract llm.ExtractionService, stateRepo repository.CharacterStateRepository) *ExtractStateWorker {
	return &ExtractStateWorker{extract: extract, stateRepo: stateRepo}
}

type ExtractStateArgs struct {
	StoryID       string
	SceneID       string
	SceneText     string
	CharacterRefs []string
	CharNameToID  map[string]string
}

func (w *ExtractStateWorker) Work(ctx context.Context, args ExtractStateArgs) error {
	slog.Info("extracting state", "sceneId", args.SceneID)

	roster := make(map[string]string, len(args.CharNameToID))
	for name := range args.CharNameToID {
		roster[name] = name
	}

	deltas, err := w.extract.ExtractState(ctx, args.SceneText, roster)
	if err != nil {
		return err
	}

	for _, delta := range deltas.Deltas {
		charID, ok := args.CharNameToID[delta.Character]
		if !ok {
			slog.Warn("extract: unknown character name, skipping", "name", delta.Character)
			continue
		}
		changes := map[string]any{
			"new_location":          delta.NewLocation,
			"learned":               delta.Learned,
			"mood":                  delta.Mood,
			"relationship_changes":  delta.RelationshipChanges,
			"items_gained":          delta.ItemsGained,
			"items_lost":            delta.ItemsLost,
		}
		state := &domain.CharacterState{
			CharacterID: charID,
			StoryID:     args.StoryID,
			SceneID:     args.SceneID,
			Location:    delta.NewLocation,
			Mood:        delta.Mood,
			Changes:     changes,
		}
		if err := w.stateRepo.Append(ctx, state); err != nil {
			return err
		}
	}

	return nil
}
