package validation

import (
	"context"

	"github.com/google/uuid"
)

type ValidationResult struct {
	Passed    bool     `json:"passed"`
	Violations []string `json:"violations,omitempty"`
}

type ValidationService interface {
	ValidateScene(ctx context.Context, generationID uuid.UUID, compiledCanon, charState, sceneText string) (*ValidationResult, error)
	ValidateCanonConsistency(ctx context.Context, storyID uuid.UUID) ([]string, error)
}
