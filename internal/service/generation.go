package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/repository"
	"github.com/premchand/story-builder/internal/worker"
)

type GenerationService struct {
	genRepo     repository.GenerationRepository
	sceneRepo   repository.SceneRepository
	charRepo    repository.CharacterRepository
	stateRepo   repository.CharacterStateRepository
	memRepo     repository.MemoryRepository
	tlRepo      repository.TimelineRepository
	sumRepo     repository.SummaryRepository

	proseSvc    llm.ProseService
	extractSvc  llm.ExtractionService
	summarySvc  llm.SummaryService
	validateSvc llm.ValidationService

	inFlight sync.Map
}

func NewGenerationService(
	genRepo repository.GenerationRepository,
	sceneRepo repository.SceneRepository,
	charRepo repository.CharacterRepository,
	stateRepo repository.CharacterStateRepository,
	memRepo repository.MemoryRepository,
	tlRepo repository.TimelineRepository,
	sumRepo repository.SummaryRepository,
	proseSvc llm.ProseService,
	extractSvc llm.ExtractionService,
	summarySvc llm.SummaryService,
	validateSvc llm.ValidationService,
) *GenerationService {
	return &GenerationService{
		genRepo: genRepo, sceneRepo: sceneRepo, charRepo: charRepo,
		stateRepo: stateRepo, memRepo: memRepo, tlRepo: tlRepo, sumRepo: sumRepo,
		proseSvc: proseSvc, extractSvc: extractSvc, summarySvc: summarySvc, validateSvc: validateSvc,
	}
}

func (s *GenerationService) Generate(ctx context.Context, sceneID string) (*domain.Generation, error) {
	if _, loaded := s.inFlight.LoadOrStore(sceneID, true); loaded {
		return nil, fmt.Errorf("generation already in progress for scene %s", sceneID)
	}

	scene, err := s.sceneRepo.Get(ctx, sceneID)
	if err != nil {
		s.inFlight.Delete(sceneID)
		return nil, fmt.Errorf("get scene: %w", err)
	}
	if scene == nil {
		s.inFlight.Delete(sceneID)
		return nil, fmt.Errorf("scene not found")
	}

	gen := &domain.Generation{
		StoryID: scene.StoryID,
		SceneID: sceneID,
		Model:   string(llm.ModelSonnet),
	}
	if err := s.genRepo.Create(ctx, gen); err != nil {
		s.inFlight.Delete(sceneID)
		return nil, fmt.Errorf("create generation: %w", err)
	}

	params := llm.PromptParams{
		BeatIntent:  scene.BeatIntent,
		POV:         scene.POV,
		Tone:        scene.Tone,
		TargetWords: scene.TargetWords,
	}

	go func() {
		s.runPipeline(ctx, gen.ID, scene, params)
		s.inFlight.Delete(sceneID)
	}()

	return gen, nil
}

func (s *GenerationService) AcceptGeneration(ctx context.Context, sceneID, genID string) error {
	gen, err := s.genRepo.Get(ctx, genID)
	if err != nil {
		return err
	}
	if gen == nil {
		return fmt.Errorf("generation not found")
	}

	gen.Accepted = true
	if err := s.genRepo.Update(ctx, gen); err != nil {
		return fmt.Errorf("accept gen: %w", err)
	}

	gens, err := s.genRepo.ListByScene(ctx, sceneID)
	if err != nil {
		return err
	}
	for _, g := range gens {
		if g.ID != genID {
			g.Accepted = false
			if err := s.genRepo.Update(ctx, g); err != nil {
				slog.Error("accept gen: marking stale failed", "genId", g.ID, "error", err)
			}
		}
	}

	scene, err := s.sceneRepo.Get(ctx, sceneID)
	if err != nil {
		return fmt.Errorf("accept gen: get scene: %w", err)
	}
	if scene != nil {
		scene.Status = domain.SceneStatusAccepted
		if err := s.sceneRepo.Update(ctx, scene); err != nil {
			return fmt.Errorf("accept gen: update scene status: %w", err)
		}
	}

	return nil
}

func (s *GenerationService) ListGenerations(ctx context.Context, sceneID string) ([]*domain.Generation, error) {
	return s.genRepo.ListByScene(ctx, sceneID)
}

func (s *GenerationService) runPipeline(ctx context.Context, genID string, scene *domain.Scene, params llm.PromptParams) {
	genWorker := worker.NewGenerateSceneWorker(s.proseSvc, s.genRepo, s.sceneRepo)
	extractWorker := worker.NewExtractStateWorker(s.extractSvc, s.stateRepo)
	memWorker := worker.NewMemoryUpdateWorker(s.memRepo)
	tlWorker := worker.NewTimelineWorker(s.tlRepo)
	sumWorker := worker.NewSummaryWorker(s.summarySvc, s.sumRepo)
	valWorker := worker.NewValidationWorker(s.validateSvc, s.genRepo)

	slog.Info("generation pipeline starting", "genId", genID)

	sceneText, err := genWorker.Work(ctx, worker.GenerateSceneArgs{
		SceneID: scene.ID,
		GenID:   genID,
		Context: params,
	})
	if err != nil {
		slog.Error("generate step failed", "genId", genID, "error", err)
		return
	}

	if err := extractWorker.Work(ctx, worker.ExtractStateArgs{
		StoryID: scene.StoryID, SceneID: scene.ID, SceneText: sceneText, CharacterRefs: scene.Participants,
	}); err != nil {
		slog.Error("extract step failed", "genId", genID, "error", err)
		return
	}

	for _, charID := range scene.Participants {
		if err := memWorker.Work(ctx, worker.MemoryUpdateArgs{
			StoryID: scene.StoryID, CharacterID: charID, SceneID: scene.ID,
			Content: sceneText, Importance: 0.5,
		}); err != nil {
			slog.Error("memory step failed", "genId", genID, "error", err)
		}
	}

	if err := tlWorker.Work(ctx, worker.TimelineArgs{
		StoryID: scene.StoryID, SceneID: scene.ID,
		Title: scene.Title, Order: scene.TimelinePosition,
	}); err != nil {
		slog.Error("timeline step failed", "genId", genID, "error", err)
	}

	if err := sumWorker.Work(ctx, worker.SummaryArgs{
		StoryID: scene.StoryID, SceneID: scene.ID, AcceptedScene: sceneText,
	}); err != nil {
		slog.Error("summary step failed", "genId", genID, "error", err)
	}

	if err := valWorker.Work(ctx, worker.ValidateArgs{
		GenerationID: genID, SceneText: sceneText,
	}); err != nil {
		slog.Error("validation step failed", "genId", genID, "error", err)
	}

	slog.Info("generation pipeline complete", "genId", genID)
}
