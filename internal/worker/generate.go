package worker

import (
	"context"
	"log/slog"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
)

type GenerateSceneWorker struct {
	prose     llm.ProseService
	genRepo   GenerationWriter
	sceneRepo SceneUpdater
}

type GenerationWriter interface {
	Create(ctx context.Context, g *domain.Generation) error
	Get(ctx context.Context, id string) (*domain.Generation, error)
	Update(ctx context.Context, g *domain.Generation) error
}

type SceneUpdater interface {
	Get(ctx context.Context, id string) (*domain.Scene, error)
	Update(ctx context.Context, s *domain.Scene) error
}

func NewGenerateSceneWorker(prose llm.ProseService, genRepo GenerationWriter, sceneRepo SceneUpdater) *GenerateSceneWorker {
	return &GenerateSceneWorker{prose: prose, genRepo: genRepo, sceneRepo: sceneRepo}
}

type GenerateSceneArgs struct {
	SceneID    string
	GenID      string
	Context    llm.PromptParams
}

func (w *GenerateSceneWorker) Work(ctx context.Context, args GenerateSceneArgs) (string, error) {
	slog.Info("generating scene", "sceneId", args.SceneID, "genId", args.GenID)

	resp, err := w.prose.GenerateScene(ctx, args.Context)
	if err != nil {
		return "", err
	}

	scene, err := w.sceneRepo.Get(ctx, args.SceneID)
	if err != nil {
		return "", err
	}
	scene.GeneratedContent = resp.Content
	if err := w.sceneRepo.Update(ctx, scene); err != nil {
		return "", err
	}

	gen, err := w.genRepo.Get(ctx, args.GenID)
	if err != nil {
		return "", err
	}
	gen.Output = resp.Content

	return resp.Content, w.genRepo.Update(ctx, gen)
}
