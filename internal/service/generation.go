package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/events"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/repository"
	"github.com/premchand/story-builder/internal/validation"
	"github.com/premchand/story-builder/internal/worker"
)

type GenerationServiceConfig struct {
	GenRepo       repository.GenerationRepository
	SceneRepo     repository.SceneRepository
	StoryRepo     repository.StoryRepository
	CharRepo      repository.CharacterRepository
	StateRepo     repository.CharacterStateRepository
	EdgeRepo      repository.SceneEdgeRepository
	MemRepo       repository.MemoryRepository
	TlRepo        repository.TimelineRepository
	SumRepo       repository.SummaryRepository
	LocRepo       repository.LocationRepository
	ProseSvc      llm.ProseService
	ExtractSvc    llm.ExtractionService
	SummarySvc    llm.SummaryService
	ValidateSvc   llm.ValidationService
	ContextBldr   *ContextBuilder
	EventBus      events.Bus
	EmbeddingSvc  llm.EmbeddingService
	SceneValidator *validation.SceneValidator
}

type GenerationService struct {
	genRepo       repository.GenerationRepository
	sceneRepo     repository.SceneRepository
	storyRepo     repository.StoryRepository
	charRepo      repository.CharacterRepository
	stateRepo     repository.CharacterStateRepository
	edgeRepo      repository.SceneEdgeRepository
	memRepo       repository.MemoryRepository
	tlRepo        repository.TimelineRepository
	sumRepo       repository.SummaryRepository
	locRepo       repository.LocationRepository

	proseSvc      llm.ProseService
	extractSvc    llm.ExtractionService
	summarySvc    llm.SummaryService
	validateSvc   llm.ValidationService
	contextBldr   *ContextBuilder
	embeddingSvc  llm.EmbeddingService
	sceneValidator *validation.SceneValidator

	genInFlight    sync.Map
	acceptInFlight sync.Map
	progress       ProgressPublisher
	eventBus       events.Bus
}

func NewGenerationService(cfg GenerationServiceConfig) *GenerationService {
	return &GenerationService{
		genRepo: cfg.GenRepo, sceneRepo: cfg.SceneRepo, storyRepo: cfg.StoryRepo,
		charRepo: cfg.CharRepo, stateRepo: cfg.StateRepo, edgeRepo: cfg.EdgeRepo,
		memRepo: cfg.MemRepo, tlRepo: cfg.TlRepo, sumRepo: cfg.SumRepo, locRepo: cfg.LocRepo,
		proseSvc: cfg.ProseSvc, extractSvc: cfg.ExtractSvc, summarySvc: cfg.SummarySvc, validateSvc: cfg.ValidateSvc,
		contextBldr: cfg.ContextBldr, eventBus: cfg.EventBus, embeddingSvc: cfg.EmbeddingSvc,
		sceneValidator: cfg.SceneValidator,
	}
}

func (s *GenerationService) SetProgressPublisher(p ProgressPublisher) {
	s.progress = p
}

func (s *GenerationService) Generate(ctx context.Context, sceneID string) (*domain.Generation, error) {
	if _, loaded := s.genInFlight.LoadOrStore(sceneID, true); loaded {
		return nil, fmt.Errorf("generation already in progress for scene %s", sceneID)
	}

	scene, err := s.sceneRepo.Get(ctx, sceneID)
	if err != nil {
		s.genInFlight.Delete(sceneID)
		return nil, fmt.Errorf("get scene: %w", err)
	}
	if scene == nil {
		s.genInFlight.Delete(sceneID)
		return nil, fmt.Errorf("scene not found")
	}

	gen := &domain.Generation{
		StoryID:    scene.StoryID,
		SceneID:    sceneID,
		Model:      string(llm.ModelSonnet),
		StepStatus: map[string]string{},
		Status:     domain.GenStatusPending,
	}
	if err := s.genRepo.Create(ctx, gen); err != nil {
		s.genInFlight.Delete(sceneID)
		return nil, fmt.Errorf("create generation: %w", err)
	}

	gen.Status = domain.GenStatusRunning
	_ = s.genRepo.Update(ctx, gen)

	go func() {
		defer s.genInFlight.Delete(sceneID)
		start := time.Now()
		defer func() {
			if r := recover(); r != nil {
				slog.Error("pipeline panic", "genId", gen.ID, "recover", r)
				bgCtx := context.Background()
				s.setStepStatus(bgCtx, gen.ID, "pipeline", domain.StepFailed)
				s.setGenError(bgCtx, gen.ID, fmt.Sprintf("pipeline panic: %v", r))
			}
		}()

		bgCtx := context.Background()
		pCtx, pCancel := context.WithTimeout(bgCtx, 5*time.Minute)
		defer pCancel()

		s.runPipeline(pCtx, gen.ID, scene)

		elapsed := time.Since(start)
		if g, err := s.genRepo.Get(bgCtx, gen.ID); err == nil && g != nil {
			g.DurationMs = elapsed.Milliseconds()
			_ = s.genRepo.Update(bgCtx, g)
		}
	}()

	return gen, nil
}



func (s *GenerationService) AcceptGeneration(ctx context.Context, sceneID, genID string) error {
	if _, loaded := s.acceptInFlight.LoadOrStore(sceneID, true); loaded {
		return fmt.Errorf("accept already in progress for scene %s", sceneID)
	}
	defer s.acceptInFlight.Delete(sceneID)

	gen, err := s.genRepo.Get(ctx, genID)
	if err != nil {
		return err
	}
	if gen == nil {
		return fmt.Errorf("generation not found")
	}

	// Atomically mark all: first clear any existing accepted, then set this one.
	// The in-flight lock prevents races between concurrent accept calls.
	gens, err := s.genRepo.ListByScene(ctx, sceneID)
	if err != nil {
		return err
	}
	for _, g := range gens {
		g.Accepted = g.ID == genID
		if err := s.genRepo.Update(ctx, g); err != nil {
			slog.Error("accept gen: marking failed", "genId", g.ID, "error", err)
		}
	}

	scene, err := s.sceneRepo.Get(ctx, sceneID)
	if err != nil {
		return fmt.Errorf("accept gen: get scene: %w", err)
	}
	if scene != nil {
		if err := scene.CanTransitionTo(domain.SceneStatusAccepted); err != nil {
			return fmt.Errorf("accept gen: %w", err)
		}
		scene.Status = domain.SceneStatusAccepted
		if err := s.sceneRepo.Update(ctx, scene); err != nil {
			return fmt.Errorf("accept gen: update scene status: %w", err)
		}
	}

	s.publishEvent(ctx, events.Event{
		Type: events.EventGenerationAccepted, StoryID: scene.StoryID, SceneID: sceneID, GenID: genID,
	})

	return nil
}

func (s *GenerationService) GetGeneration(ctx context.Context, genID string) (*domain.Generation, error) {
	return s.genRepo.Get(ctx, genID)
}

func (s *GenerationService) ListGenerations(ctx context.Context, sceneID string) ([]*domain.Generation, error) {
	return s.genRepo.ListByScene(ctx, sceneID)
}

func (s *GenerationService) setStepStatus(ctx context.Context, genID, step, status string) {
	if err := s.genRepo.SetStepStatus(ctx, genID, step, status); err != nil {
		slog.Error("set step status failed", "genId", genID, "step", step, "status", status, "error", err)
	}
	if s.progress != nil {
		s.progress.Publish(genID, ProgressEvent{GenID: genID, Step: step, Status: status})
	}
}

func (s *GenerationService) publishEvent(ctx context.Context, evt events.Event) {
	if s.eventBus != nil {
		s.eventBus.Publish(ctx, evt)
	}
}

func (s *GenerationService) setGenError(ctx context.Context, genID, errMsg string) {
	gen, err := s.genRepo.Get(ctx, genID)
	if err != nil || gen == nil {
		return
	}
	gen.Error = errMsg
	_ = s.genRepo.Update(ctx, gen)
}

func (s *GenerationService) setGenStatus(ctx context.Context, genID, status string) {
	gen, err := s.genRepo.Get(ctx, genID)
	if err != nil || gen == nil {
		return
	}
	gen.Status = status
	_ = s.genRepo.Update(ctx, gen)
}

func (s *GenerationService) runStep(ctx context.Context, genID, stepName string, fn func(context.Context) error) bool {
	s.setStepStatus(ctx, genID, stepName, domain.StepRunning)

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			slog.Info("retrying step", "genId", genID, "step", stepName, "attempt", attempt)
		}
		lastErr = fn(ctx)
		if lastErr == nil {
			s.setStepStatus(ctx, genID, stepName, domain.StepDone)
			return true
		}
		if ctx.Err() != nil {
			break
		}
	}

	slog.Error("step failed after retries", "genId", genID, "step", stepName, "error", lastErr)
	s.setStepStatus(ctx, genID, stepName, domain.StepFailed)
	return false
}

func (s *GenerationService) runNonCriticalStep(ctx context.Context, genID, stepName string, fn func(context.Context) error) {
	if ctx.Err() != nil {
		return
	}
	s.setStepStatus(ctx, genID, stepName, domain.StepRunning)
	if err := fn(ctx); err != nil {
		slog.Warn("non-critical step failed", "genId", genID, "step", stepName, "error", err)
		s.setStepStatus(ctx, genID, stepName, domain.StepFailed)
		return
	}
	s.setStepStatus(ctx, genID, stepName, domain.StepDone)
}

func (s *GenerationService) runPipeline(ctx context.Context, genID string, scene *domain.Scene) {
	genWorker := worker.NewGenerateSceneWorker(s.proseSvc, s.genRepo, s.sceneRepo)
	extractWorker := worker.NewExtractStateWorker(s.extractSvc, s.stateRepo)
	memWorker := worker.NewMemoryUpdateWorker(s.memRepo, s.embeddingSvc)
	tlWorker := worker.NewTimelineWorker(s.tlRepo, s.edgeRepo)
	sumWorker := worker.NewSummaryWorker(s.summarySvc, s.sumRepo)
	valWorker := worker.NewValidationWorker(s.validateSvc, s.genRepo)

	slog.Info("generation pipeline starting", "genId", genID)

	if s.sceneValidator != nil {
		violations := s.sceneValidator.ValidatePreGeneration(ctx, scene)
		for _, v := range violations {
			slog.Warn("pre-generation validation", "genId", genID, "severity", v.Severity, "field", v.Field, "message", v.Message)
		}
	}

	criticalFailed := false
	anyFailed := false

	var builtContext *BuiltContext

	if s.contextBldr != nil {
		bCtx, err := s.contextBldr.Build(ctx, scene)
		if err != nil {
			slog.Warn("context builder failed, falling back to simple params", "genId", genID, "error", err)
			builtContext = nil
		} else {
			builtContext = bCtx
		}
	}

	var params llm.PromptParams
	charNameToID := make(map[string]string)
	if builtContext != nil {
		for _, name := range builtContext.CharacterNames {
			for _, c := range builtContext.Params.CharacterCards {
				if c.Name == name {
					charNameToID[name] = c.Name
					break
				}
			}
		}
		params = builtContext.Params
	} else {
		params = llm.PromptParams{
			BeatIntent:  scene.BeatIntent,
			POV:         scene.POV,
			Tone:        scene.Tone,
			TargetWords: scene.TargetWords,
		}
	}

	if !s.runStep(ctx, genID, "generate", func(sCtx context.Context) error {
		_, err := genWorker.Work(sCtx, worker.GenerateSceneArgs{
			SceneID: scene.ID,
			GenID:   genID,
			Context: params,
		})
		return err
	}) {
		criticalFailed = true
		anyFailed = true
	}

	s.publishEvent(ctx, events.Event{
		Type: events.EventSceneGenerated, StoryID: scene.StoryID, SceneID: scene.ID, GenID: genID,
		Data: map[string]any{"criticalFailed": criticalFailed},
	})

	sceneText, _ := func() (string, error) {
		gen, err := s.genRepo.Get(ctx, genID)
		if err != nil || gen == nil {
			return "", err
		}
		return gen.Output, nil
	}()
	if sceneText == "" {
		sceneText = scene.GeneratedContent
	}

	if !criticalFailed && sceneText != "" {
		if !s.runStep(ctx, genID, "extract", func(sCtx context.Context) error {
			return extractWorker.Work(sCtx, worker.ExtractStateArgs{
				StoryID: scene.StoryID, SceneID: scene.ID, SceneText: sceneText,
				CharacterRefs: scene.Participants, CharNameToID: charNameToID,
			})
		}) {
			anyFailed = true
		}
		s.publishEvent(ctx, events.Event{
			Type: events.EventCharacterStatesExtracted, StoryID: scene.StoryID, SceneID: scene.ID, GenID: genID,
		})

		// Post-generation invariant validation
		if s.sceneValidator != nil {
			newStates, _ := s.stateRepo.ListByScene(ctx, scene.ID)
			var checks []validation.PostGenerationCheck
			for _, ns := range newStates {
				check := validation.PostGenerationCheck{
					CharacterID: ns.CharacterID,
					NewLocation: ns.Location,
					Learned:     extractLearned(ns.Changes),
				}
				prevStates, _ := s.stateRepo.ListByCharacter(ctx, ns.CharacterID)
				for _, ps := range prevStates {
					if ps.SceneID != scene.ID {
						check.PreviousLocation = ps.Location
						check.PreviousKnowledge = ps.Knowledge
						break
					}
				}
				checks = append(checks, check)
			}
			violations := s.sceneValidator.ValidatePostGeneration(ctx, scene, checks)
			for _, v := range violations {
				slog.Warn("post-generation validation", "genId", genID, "severity", v.Severity, "field", v.Field, "message", v.Message)
			}
		}
	}

	if sceneText != "" {
		s.runNonCriticalStep(ctx, genID, "memory", func(sCtx context.Context) error {
			for _, charID := range scene.Participants {
				if err := memWorker.Work(sCtx, worker.MemoryUpdateArgs{
					StoryID: scene.StoryID, CharacterID: charID, SceneID: scene.ID,
					Content: sceneText, Importance: 0.5,
				}); err != nil {
					slog.Error("memory step failed", "genId", genID, "charId", charID, "error", err)
				}
			}
			return nil
		})
		s.publishEvent(ctx, events.Event{
			Type: events.EventMemoriesCreated, StoryID: scene.StoryID, SceneID: scene.ID, GenID: genID,
		})
	}

	s.runNonCriticalStep(ctx, genID, "timeline", func(sCtx context.Context) error {
		return tlWorker.Work(sCtx, worker.TimelineArgs{
			StoryID: scene.StoryID, SceneID: scene.ID,
			Title: scene.Title, Order: scene.TimelinePosition,
		})
	})
	s.publishEvent(ctx, events.Event{
		Type: events.EventTimelineRecorded, StoryID: scene.StoryID, SceneID: scene.ID, GenID: genID,
	})

	if sceneText != "" {
		s.runNonCriticalStep(ctx, genID, "summary", func(sCtx context.Context) error {
			prevSummary := ""
			if existing, _ := s.sumRepo.GetByLevel(sCtx, scene.StoryID, domain.SummaryLevelStory); existing != nil {
				prevSummary = existing.Content
			}
			return sumWorker.Work(sCtx, worker.SummaryArgs{
				StoryID: scene.StoryID, SceneID: scene.ID,
				PreviousSummary: prevSummary,
				AcceptedScene:   sceneText,
			})
		})
		s.publishEvent(ctx, events.Event{
			Type: events.EventSummaryUpdated, StoryID: scene.StoryID, SceneID: scene.ID, GenID: genID,
		})
	}

	if sceneText != "" {
		s.runNonCriticalStep(ctx, genID, "validate", func(sCtx context.Context) error {
			canonXML := ""
			if story, _ := s.storyRepo.Get(sCtx, scene.StoryID); story != nil {
				if len(story.CanonPins) > 0 {
					b, _ := json.Marshal(story.CanonPins)
					canonXML = string(b)
				}
			}
			charState := ""
			if states, _ := s.stateRepo.ListByScene(sCtx, scene.ID); len(states) > 0 {
				b, _ := json.Marshal(states)
				charState = string(b)
			}
			return valWorker.Work(sCtx, worker.ValidateArgs{
				GenerationID: genID, CanonXML: canonXML, CharState: charState, SceneText: sceneText,
			})
		})
		s.publishEvent(ctx, events.Event{
			Type: events.EventSceneValidated, StoryID: scene.StoryID, SceneID: scene.ID, GenID: genID,
		})
	}

	switch {
	case criticalFailed:
		s.setGenStatus(ctx, genID, domain.GenStatusFailed)
		s.setGenError(ctx, genID, "generate step failed after retries")
		slog.Error("generation pipeline failed", "genId", genID)
		s.publishEvent(ctx, events.Event{
			Type: events.EventPipelineFailed, StoryID: scene.StoryID, SceneID: scene.ID, GenID: genID,
		})
	case anyFailed:
		s.setGenStatus(ctx, genID, domain.GenStatusPartialSuccess)
		slog.Warn("generation pipeline completed with partial failures", "genId", genID)
		s.publishEvent(ctx, events.Event{
			Type: events.EventPipelineComplete, StoryID: scene.StoryID, SceneID: scene.ID, GenID: genID,
			Data: map[string]any{"status": domain.GenStatusPartialSuccess},
		})
	default:
		s.setGenStatus(ctx, genID, domain.GenStatusSuccess)
		slog.Info("generation pipeline complete", "genId", genID)
		s.publishEvent(ctx, events.Event{
			Type: events.EventPipelineComplete, StoryID: scene.StoryID, SceneID: scene.ID, GenID: genID,
			Data: map[string]any{"status": domain.GenStatusSuccess},
		})
	}
}

func extractLearned(changes map[string]any) []string {
	if changes == nil {
		return nil
	}
	raw, ok := changes["learned"]
	if !ok {
		return nil
	}
	learned, ok := raw.([]string)
	if !ok {
		rawList, ok := raw.([]any)
		if !ok {
			return nil
		}
		learned = make([]string, 0, len(rawList))
		for _, item := range rawList {
			if s, ok := item.(string); ok {
				learned = append(learned, s)
			}
		}
		return learned
	}
	return learned
}
