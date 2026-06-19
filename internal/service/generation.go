package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

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
	EdgeRepo    repository.SceneEdgeRepository
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
	edgeRepo    repository.SceneEdgeRepository
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
	progress       ProgressPublisher
}

func NewGenerationService(cfg GenerationServiceConfig) *GenerationService {
	return &GenerationService{
		genRepo: cfg.GenRepo, sceneRepo: cfg.SceneRepo, storyRepo: cfg.StoryRepo,
		charRepo: cfg.CharRepo, stateRepo: cfg.StateRepo, edgeRepo: cfg.EdgeRepo,
		memRepo: cfg.MemRepo, tlRepo: cfg.TlRepo, sumRepo: cfg.SumRepo, locRepo: cfg.LocRepo,
		proseSvc: cfg.ProseSvc, extractSvc: cfg.ExtractSvc, summarySvc: cfg.SummarySvc, validateSvc: cfg.ValidateSvc,
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
	}
	if err := s.genRepo.Create(ctx, gen); err != nil {
		s.genInFlight.Delete(sceneID)
		return nil, fmt.Errorf("create generation: %w", err)
	}

	params, charNameToID := s.buildPromptParams(ctx, scene)

	go func() {
		defer s.genInFlight.Delete(sceneID)
		defer func() {
			if r := recover(); r != nil {
				slog.Error("pipeline panic", "genId", gen.ID, "recover", r)
				s.setStepStatus(context.Background(), gen.ID, "pipeline", domain.StepFailed)
			}
		}()
		pCtx, pCancel := context.WithTimeout(ctx, 5*time.Minute)
		defer pCancel()
		s.runPipeline(pCtx, gen.ID, scene, params, charNameToID)
	}()

	return gen, nil
}

func (s *GenerationService) buildPromptParams(ctx context.Context, scene *domain.Scene) (llm.PromptParams, map[string]string) {
	params := llm.PromptParams{
		BeatIntent:  scene.BeatIntent,
		POV:         scene.POV,
		Tone:        scene.Tone,
		TargetWords: scene.TargetWords,
	}

	allChars, err := s.charRepo.ListByStory(ctx, scene.StoryID)
	if err != nil {
		slog.Warn("buildPromptParams: list characters", "storyId", scene.StoryID, "error", err)
		return params, nil
	}

	charNameToID := make(map[string]string, len(scene.Participants))

	participantIDs := make(map[string]bool, len(scene.Participants))
	for _, pid := range scene.Participants {
		participantIDs[pid] = true
	}
	params.CharState = make(map[string]interface{})
	for _, c := range allChars {
		if !participantIDs[c.ID] && !participantIDs[c.CharID] {
			continue
		}
		charNameToID[c.Name] = c.CharID

		card := llm.CharacterCard{
			Name:         c.Name,
			Description:  c.Persona,
			Type:         "character",
			Traits:       c.Traits,
			VoiceSamples: c.VoiceSamples,
			Want:         c.Want,
			Need:         c.Need,
			FalseBelief:  c.FalseBelief,
			ArcType:      c.ArcType,
		}
		if c.Backstory != "" {
			card.Description = c.Persona + ". " + c.Backstory
		}
		if c.Relationships != nil {
			card.Relationships = c.Relationships
		}
		if len(c.RelData) > 0 {
			relData := make(map[string]llm.NumericRelationships, len(c.RelData))
			for _, r := range c.RelData {
				relData[r.TargetName] = llm.NumericRelationships{
					Trust:     r.Trust,
					Respect:   r.Respect,
					Fear:      r.Fear,
					Affection: r.Affection,
				}
			}
			card.RelData = relData
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

	memories := make(map[string][]string)
	for charID := range participantIDs {
		mems, err := s.memRepo.ListByCharacter(ctx, charID)
		if err != nil || len(mems) == 0 {
			continue
		}
		charName := ""
		for _, c := range allChars {
			if c.CharID == charID || c.ID == charID {
				charName = c.Name
				break
			}
		}
		if charName == "" {
			continue
		}
		snippets := make([]string, 0, len(mems))
		for _, m := range mems {
			snippets = append(snippets, m.Content)
		}
		memories[charName] = snippets
	}
	if len(memories) > 0 {
		params.Memories = memories
	}

	return params, charNameToID
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

func (s *GenerationService) runPipeline(ctx context.Context, genID string, scene *domain.Scene, params llm.PromptParams, charNameToID map[string]string) {
	genWorker := worker.NewGenerateSceneWorker(s.proseSvc, s.genRepo, s.sceneRepo)
	extractWorker := worker.NewExtractStateWorker(s.extractSvc, s.stateRepo)
	memWorker := worker.NewMemoryUpdateWorker(s.memRepo)
	tlWorker := worker.NewTimelineWorker(s.tlRepo, s.edgeRepo)
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
		StoryID: scene.StoryID, SceneID: scene.ID, SceneText: sceneText, CharacterRefs: scene.Participants, CharNameToID: charNameToID,
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
