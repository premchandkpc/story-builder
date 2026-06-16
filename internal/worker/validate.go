package worker

import (
	"context"
	"log/slog"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
)

type ValidationWorker struct {
	validate llm.ValidationService
	genRepo  ValidationWriter
}

type ValidationWriter interface {
	Update(ctx context.Context, g *domain.Generation) error
}

func NewValidationWorker(validate llm.ValidationService, genRepo ValidationWriter) *ValidationWorker {
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

	gen := &domain.Generation{
		ID:               args.GenerationID,
		ValidationResult: result,
	}

	return w.genRepo.Update(ctx, gen)
}
