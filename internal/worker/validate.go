package worker

import (
	"context"
	"log/slog"

	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/repository"
)

type ValidationWorker struct {
	validate llm.ValidationService
	genRepo  repository.GenerationRepository
}

func NewValidationWorker(validate llm.ValidationService, genRepo repository.GenerationRepository) *ValidationWorker {
	return &ValidationWorker{validate: validate, genRepo: genRepo}
}

type ValidateArgs struct {
	GenerationID string
	CanonXML     string
	CharState    string
	SceneText    string
}

func (w *ValidationWorker) Work(ctx context.Context, args ValidateArgs) error {
	slog.Info("validating scene", "generationId", args.GenerationID)

	result, err := w.validate.ValidateAgainstCanon(ctx, args.CanonXML, args.CharState, args.SceneText)
	if err != nil {
		return err
	}

	gen, err := w.genRepo.Get(ctx, args.GenerationID)
	if err != nil {
		return err
	}
	if gen == nil {
		return nil
	}
	gen.ValidationResult = result

	return w.genRepo.Update(ctx, gen)
}
