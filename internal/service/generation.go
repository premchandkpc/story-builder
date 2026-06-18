package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/repository"
	"github.com/premchand/story-builder/internal/worker"
)

type GenerationServiceConfig struct {
	GenRepo     repository.GenerationRepository
	SceneRepo   repository.SceneRepository
	StoryRepo   repository.StoryRepository
	CharRepo    repository.CharacterRepository
	StateRepo   repository.CharacterStateRepository
	MemRepo     repository.MemoryRepository
	TlRepo      repository.TimelineRepository
	SumRepo     repository.SummaryRepository
	LocRepo     repository.LocationRepository
	ProseSvc    llm.ProseService
	ExtractSvc  llm.ExtractionService
	SummarySvc  llm.SummaryService
	ValidateSvc llm.ValidationService
}

type GenerationService struct {
	genRepo     repository.GenerationRepository
	sceneRepo   repository.SceneRepository
	storyRepo   repository.StoryRepository
	charRepo    repository.CharacterRepository
	stateRepo   repository.CharacterStateRepository
	memRepo     repository.MemoryRepository
	tlRepo      repository.TimelineRepository
	sumRepo     repository.SummaryRepository
	locRepo     repository.LocationRepository

	proseSvc    llm.ProseService
	extractSvc  llm.ExtractionService
	summarySvc  llm.SummaryService
	validateSvc llm.ValidationService

	genInFlight    sync.Map
	acceptInFlight sync.Map
}

func NewGenerationService(cfg GenerationServiceConfig) *GenerationService {
	return &GenerationService{
		genRepo: cfg.GenRepo, sceneRepo: cfg.SceneRepo, storyRepo: cfg.StoryRepo,
		charRepo: cfg.CharRepo, stateRepo: cfg.StateRepo, memRepo: cfg.MemRepo,
		tlRepo: cfg.TlRepo, sumRepo: cfg.SumRepo, locRepo: cfg.LocRepo,
		proseSvc: cfg.ProseSvc, extractSvc: cfg.ExtractSvc, summarySvc: cfg.SummarySvc, validateSvc: cfg.ValidateSvc,
	}
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
	}
	if err := s.genRepo.Create(ctx, gen); err != nil {
		s.genInFlight.Delete(sceneID)
		return nil, fmt.Errorf("create generation: %w", err)
	}

	params := s.buildPromptParams(ctx, scene)

	go func() {
		defer s.genInFlight.Delete(sceneID)
		s.runPipeline(context.WithoutCancel(ctx), gen.ID, scene, params)
	}()

	return gen, nil
}

func (s *GenerationService) buildPromptParams(ctx context.Context, scene *domain.Scene) llm.PromptParams {
	params := llm.PromptParams{
		BeatIntent:  scene.BeatIntent,
		POV:         scene.POV,
		Tone:        scene.Tone,
		TargetWords: scene.TargetWords,
	}

	allChars, err := s.charRepo.ListByStory(ctx, scene.StoryID)
	if err != nil {
		slog.Warn("buildPromptParams: list characters", "storyId", scene.StoryID, "error", err)
		return params
	}

	participantIDs := make(map[string]bool, len(scene.Participants))
	for _, pid := range scene.Participants {
		participantIDs[pid] = true
	}
	params.CharState = make(map[string]interface{})
	for _, c := range allChars {
		if !participantIDs[c.ID] && !participantIDs[c.CharID] {
			continue
		}
		card := llm.CharacterCard{
			Name:         c.Name,
			Description:  c.Persona,
			Type:         "character",
			Traits:       c.Traits,
			VoiceSamples: c.VoiceSamples,
		}
		if c.Backstory != "" {
			card.Description = c.Persona + ". " + c.Backstory
		}
		if c.Relationships != nil {
			card.Relationships = c.Relationships
		}
		params.CharacterCards = append(params.CharacterCards, card)

		states, _ := s.stateRepo.ListByCharacter(ctx, c.CharID)
		if len(states) > 0 {
			latest := states[len(states)-1]
			cs := llm.CharacterState{
				StoryID:     latest.StoryID,
				CharacterID: latest.CharacterID,
				AsOfScene:   latest.SceneID,
				Location:    latest.Location,
				Mood:        latest.Mood,
				Items:       latest.Inventory,
			}
			if latest.Relationships != nil {
				cs.Relationships = latest.Relationships
			}
			if m, ok := latest.Changes["learned"].([]string); ok {
				cs.Knows = m
			}
			if m, ok := latest.Changes["does_not_know"].([]string); ok {
				cs.DoesNotKnow = m
			}
			params.CharState[c.Name] = cs
		}
	}

	if scene.LocationRef != "" {
		locs, err := s.locRepo.ListByStory(ctx, scene.StoryID)
		if err == nil {
			for _, loc := range locs {
				if loc.Name == scene.LocationRef || loc.ID == scene.LocationRef {
					params.LocationCard = &llm.CharacterCard{
						Name:        loc.Name,
						Description: loc.Description,
						Type:        "location",
						Props:       loc.Props,
					}
					break
				}
			}
		}
	}

	if existing, _ := s.sumRepo.GetByLevel(ctx, scene.StoryID, "story"); existing != nil {
		params.BranchSummary = existing.Content
	}

	return params
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

func (s *GenerationService) setStepStatus(ctx context.Context, genID, step, status string) {
	gen, err := s.genRepo.Get(ctx, genID)
	if err != nil || gen == nil {
		return
	}
	if gen.StepStatus == nil {
		gen.StepStatus = map[string]string{}
	}
	gen.StepStatus[step] = status
	s.genRepo.Update(ctx, gen)
}

func (s *GenerationService) runPipeline(ctx context.Context, genID string, scene *domain.Scene, params llm.PromptParams) {
	genWorker := worker.NewGenerateSceneWorker(s.proseSvc, s.genRepo, s.sceneRepo)
	extractWorker := worker.NewExtractStateWorker(s.extractSvc, s.stateRepo)
	memWorker := worker.NewMemoryUpdateWorker(s.memRepo)
	tlWorker := worker.NewTimelineWorker(s.tlRepo)
	sumWorker := worker.NewSummaryWorker(s.summarySvc, s.sumRepo)
	valWorker := worker.NewValidationWorker(s.validateSvc, s.genRepo)

	slog.Info("generation pipeline starting", "genId", genID)

	s.setStepStatus(ctx, genID, "generate", domain.StepRunning)
	sceneText, err := genWorker.Work(ctx, worker.GenerateSceneArgs{
		SceneID: scene.ID,
		GenID:   genID,
		Context: params,
	})
	if err != nil {
		slog.Error("generate step failed", "genId", genID, "error", err)
		s.setStepStatus(ctx, genID, "generate", domain.StepFailed)
		return
	}
	s.setStepStatus(ctx, genID, "generate", domain.StepDone)

	s.setStepStatus(ctx, genID, "extract", domain.StepRunning)
	if err := extractWorker.Work(ctx, worker.ExtractStateArgs{
		StoryID: scene.StoryID, SceneID: scene.ID, SceneText: sceneText, CharacterRefs: scene.Participants,
	}); err != nil {
		slog.Error("extract step failed", "genId", genID, "error", err)
		s.setStepStatus(ctx, genID, "extract", domain.StepFailed)
		return
	}
	s.setStepStatus(ctx, genID, "extract", domain.StepDone)

	s.setStepStatus(ctx, genID, "memory", domain.StepRunning)
	for _, charID := range scene.Participants {
		if err := memWorker.Work(ctx, worker.MemoryUpdateArgs{
			StoryID: scene.StoryID, CharacterID: charID, SceneID: scene.ID,
			Content: sceneText, Importance: 0.5,
		}); err != nil {
			slog.Error("memory step failed", "genId", genID, "error", err)
		}
	}
	s.setStepStatus(ctx, genID, "memory", domain.StepDone)

	s.setStepStatus(ctx, genID, "timeline", domain.StepRunning)
	if err := tlWorker.Work(ctx, worker.TimelineArgs{
		StoryID: scene.StoryID, SceneID: scene.ID,
		Title: scene.Title, Order: scene.TimelinePosition,
	}); err != nil {
		slog.Error("timeline step failed", "genId", genID, "error", err)
		s.setStepStatus(ctx, genID, "timeline", domain.StepFailed)
		return
	}
	s.setStepStatus(ctx, genID, "timeline", domain.StepDone)

	prevSummary := ""
	if existing, _ := s.sumRepo.GetByLevel(ctx, scene.StoryID, "story"); existing != nil {
		prevSummary = existing.Content
	}

	s.setStepStatus(ctx, genID, "summary", domain.StepRunning)
	if err := sumWorker.Work(ctx, worker.SummaryArgs{
		StoryID: scene.StoryID, SceneID: scene.ID,
		PreviousSummary: prevSummary,
		AcceptedScene:   sceneText,
	}); err != nil {
		slog.Error("summary step failed", "genId", genID, "error", err)
		s.setStepStatus(ctx, genID, "summary", domain.StepFailed)
		return
	}
	s.setStepStatus(ctx, genID, "summary", domain.StepDone)

	canonXML := ""
	if story, _ := s.storyRepo.Get(ctx, scene.StoryID); story != nil {
		if len(story.CanonPins) > 0 {
			b, _ := json.Marshal(story.CanonPins)
			canonXML = string(b)
		}
	}

	charState := ""
	if states, _ := s.stateRepo.ListByScene(ctx, scene.ID); len(states) > 0 {
		b, _ := json.Marshal(states)
		charState = string(b)
	}

	s.setStepStatus(ctx, genID, "validate", domain.StepRunning)
	if err := valWorker.Work(ctx, worker.ValidateArgs{
		GenerationID: genID, CanonXML: canonXML, CharState: charState, SceneText: sceneText,
	}); err != nil {
		slog.Error("validation step failed", "genId", genID, "error", err)
		s.setStepStatus(ctx, genID, "validate", domain.StepFailed)
		return
	}
	s.setStepStatus(ctx, genID, "validate", domain.StepDone)

	slog.Info("generation pipeline complete", "genId", genID)
}
