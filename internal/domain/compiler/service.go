package compiler

import (
	"context"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/compiler"
)

type CompilerService interface {
	CompileSceneContext(ctx context.Context, nodeID uuid.UUID) (*compiler.CompiledContext, error)
	ComputeHash(ctx *compiler.CompiledContext) (string, error)
}
