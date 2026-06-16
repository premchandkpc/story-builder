package worker

import (
	"context"
	"log/slog"

	"github.com/premchand/story-builder/internal/domain"
)

type MemoryUpdateWorker struct {
	memRepo MemoryWriter
}

type MemoryWriter interface {
	Create(ctx context.Context, m *domain.CharacterMemory) error
}

func NewMemoryUpdateWorker(memRepo MemoryWriter) *MemoryUpdateWorker {
	return &MemoryUpdateWorker{memRepo: memRepo}
}

type MemoryUpdateArgs struct {
	StoryID     string
	CharacterID string
	SceneID     string
	Content     string
	Importance  float64
}

func (w *MemoryUpdateWorker) Work(ctx context.Context, args MemoryUpdateArgs) error {
	slog.Info("creating memory", "characterId", args.CharacterID, "sceneId", args.SceneID)

	mem := &domain.CharacterMemory{
		StoryID:     args.StoryID,
		CharacterID: args.CharacterID,
		SceneID:     args.SceneID,
		Content:     args.Content,
		Type:        domain.MemoryTypeEvent,
		Importance:  args.Importance,
	}

	return w.memRepo.Create(ctx, mem)
}
