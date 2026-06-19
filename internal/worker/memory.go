package worker

import (
	"context"
	"log/slog"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/repository"
)

type MemoryUpdateWorker struct {
	memRepo     repository.MemoryRepository
	embedSvc    llm.EmbeddingService
}

func NewMemoryUpdateWorker(memRepo repository.MemoryRepository, embedSvc llm.EmbeddingService) *MemoryUpdateWorker {
	return &MemoryUpdateWorker{memRepo: memRepo, embedSvc: embedSvc}
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

	if w.embedSvc != nil {
		embedding, err := w.embedSvc.GenerateEmbedding(ctx, args.Content)
		if err != nil {
			slog.Warn("embedding generation failed, storing without vector", "characterId", args.CharacterID, "error", err)
		} else {
			mem.Embedding = embedding
		}
	}

	return w.memRepo.Create(ctx, mem)
}
