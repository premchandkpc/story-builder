package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/event"
	"github.com/premchand/story-builder/internal/events"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/orchestration"
	"github.com/premchand/story-builder/internal/repository"
	"github.com/premchand/story-builder/internal/validation"
	"github.com/premchand/story-builder/internal/worker"
)

type GenerationJobWorkerConfig struct {
	JobRepo        repository.JobRepository
	LockRepo       repository.SceneLockRepository
	RunRepo        repository.RunRepository
	StepRepo       repository.RunStepRepository
	GenRepo        repository.GenerationRepository
	SceneRepo      repository.SceneRepository
	StoryRepo      repository.StoryRepository
	CharRepo       repository.CharacterRepository
	StateRepo      repository.CharacterStateRepository
	EdgeRepo       repository.SceneEdgeRepository
	BibleRepo      repository.BibleRepository
	MemRepo        repository.MemoryRepository
	TlRepo         repository.TimelineRepository
	SumRepo        repository.SummaryRepository
	LocRepo        repository.LocationRepository
	ProseSvc       llm.ProseService
	ExtractSvc     llm.ExtractionService
	SummarySvc     llm.SummaryService
	ValidateSvc    llm.ValidationService
	ContextBldr    *ContextBuilder
	EventBus       events.Bus
	EmbeddingSvc   llm.EmbeddingService
	SceneValidator *validation.SceneValidator
	EventRepo      repository.NarrativeEventRepository
	EventExtractor *event.EventExtractor
	EventValidator *event.EventValidator
	Progress       ProgressPublisher
	AgentSvc       *AgentService
	PollInterval   time.Duration
	LeaseTime      time.Duration
}

type GenerationJobWorker struct {
	worker *orchestration.Worker
}

func NewGenerationJobWorker(cfg GenerationJobWorkerConfig) *GenerationJobWorker {
	recorder := orchestration.NewRunRecorder(cfg.RunRepo, cfg.StepRepo)
	pipe := buildGenerateScenePipeline(cfg)
	wrk := orchestration.NewWorker(orchestration.WorkerConfig{
		JobRepo:           cfg.JobRepo,
		GenRepo:           cfg.GenRepo,
		LockRepo:          cfg.LockRepo,
		Recorder:          recorder,
		Pipelines:         []*orchestration.PipelineDef{pipe},
		PollInterval:      cfg.PollInterval,
		LeaseTime:         cfg.LeaseTime,
		HeartbeatInterval: 30 * time.Second,
		MaxConcurrency:    3,
	})
	return &GenerationJobWorker{worker: wrk}
}

func (w *GenerationJobWorker) Start() {
	w.worker.Start()
}

func (w *GenerationJobWorker) Stop() {
	w.worker.Stop()
}

func buildGenerateScenePipeline(cfg GenerationJobWorkerConfig) *orchestration.PipelineDef {
	return &orchestration.PipelineDef{
		Name:    "generate_scene",
		JobType: domain.JobTypeGenerateScene,
		RunType: domain.RunTypeGenerateScene,
		Steps: []orchestration.StepDef{
			{
				Name: "generate", Critical: true, MaxRetries: 3,
				Timeout: 4 * time.Minute, Model: "claude-sonnet",
				Run: func(ctx context.Context, sc *orchestration.StepContext) error {
					return runGenerateStep(ctx, sc, cfg)
				},
			},
			{
				Name: "extract", Critical: true, MaxRetries: 3,
				Timeout: 2 * time.Minute, Model: "local-7b",
				Run: func(ctx context.Context, sc *orchestration.StepContext) error {
					return runExtractStep(ctx, sc, cfg)
				},
			},
			{
				Name: "memory", Critical: false, MaxRetries: 0,
				Timeout: 1 * time.Minute, Model: "local-7b",
				Run: func(ctx context.Context, sc *orchestration.StepContext) error {
					return runMemoryStep(ctx, sc, cfg)
				},
			},
			{
				Name: "timeline", Critical: false, MaxRetries: 0,
				Timeout: 30 * time.Second,
				Run: func(ctx context.Context, sc *orchestration.StepContext) error {
					return runTimelineStep(ctx, sc, cfg)
				},
			},
			{
				Name: "summary", Critical: false, MaxRetries: 0,
				Timeout: 1 * time.Minute, Model: "local-7b",
				Run: func(ctx context.Context, sc *orchestration.StepContext) error {
					return runSummaryStep(ctx, sc, cfg)
				},
			},
			{
				Name: "validate", Critical: false, MaxRetries: 0,
				Timeout: 1 * time.Minute, Model: "claude-haiku",
				Run: func(ctx context.Context, sc *orchestration.StepContext) error {
					return runValidateStep(ctx, sc, cfg)
				},
			},
		},
	}
}

func getSceneText(ctx context.Context, genRepo repository.GenerationRepository, scene *domain.Scene, genID string) string {
	sceneText, _ := func() (string, error) {
		g, err := genRepo.Get(ctx, genID)
		if err != nil || g == nil {
			return "", err
		}
		return g.Output, nil
	}()
	if sceneText == "" {
		sceneText = scene.GeneratedContent
	}
	return sceneText
}

func publishEvent(eventBus events.Bus, evt events.Event) {
	if eventBus != nil {
		eventBus.Publish(context.Background(), evt) // intentionally background: caller ctx may be cancelled
	}
}

func runGenerateStep(ctx context.Context, sc *orchestration.StepContext, cfg GenerationJobWorkerConfig) error {
	gen, err := cfg.GenRepo.Get(ctx, sc.GenID)
	if err != nil || gen == nil {
		return fmt.Errorf("generation not found: %w", err)
	}

	scene, err := cfg.SceneRepo.Get(ctx, sc.SceneID)
	if err != nil || scene == nil {
		gen.Status = domain.GenStatusFailed
		gen.Error = "scene not found"
		_ = cfg.GenRepo.Update(ctx, gen)
		return fmt.Errorf("scene not found: %w", err)
	}

	gen.Status = domain.GenStatusRunning
	_ = cfg.GenRepo.Update(ctx, gen)

	genWorker := worker.NewGenerateSceneWorker(cfg.ProseSvc, cfg.GenRepo, cfg.SceneRepo)
	extractWorker := worker.NewExtractStateWorker(cfg.ExtractSvc, cfg.StateRepo)
	memWorker := worker.NewMemoryUpdateWorker(cfg.MemRepo, cfg.EmbeddingSvc)
	tlWorker := worker.NewTimelineWorker(cfg.TlRepo, cfg.EdgeRepo, cfg.BibleRepo)
	sumWorker := worker.NewSummaryWorker(cfg.SummarySvc, cfg.SumRepo)
	valWorker := worker.NewValidationWorker(cfg.ValidateSvc, cfg.GenRepo)

	if cfg.Progress != nil {
		cfg.Progress.Publish(gen.ID, ProgressEvent{GenID: gen.ID, Step: "generate", Status: "running"})
	}
	if cfg.SceneValidator != nil {
		violations := cfg.SceneValidator.ValidatePreGeneration(ctx, scene)
		for _, v := range violations {
			slog.Warn("pre-generation validation", "genId", gen.ID, "severity", v.Severity, "field", v.Field, "message", v.Message)
		}
	}

	var builtContext *BuiltContext
	if cfg.ContextBldr != nil {
		bCtx, err := cfg.ContextBldr.Build(ctx, scene)
		if err != nil {
			slog.Warn("context builder failed, falling back to simple params", "genId", gen.ID, "error", err)
		} else {
			builtContext = bCtx
		}
	}

	var params llm.PromptParams
	charNameToID := make(map[string]string)
	if builtContext != nil {
		if builtContext.CharNameToID != nil {
			charNameToID = builtContext.CharNameToID
		}
		params = builtContext.Params

		cc := &llm.CompiledContext{
			CharacterCards: builtContext.Params.CharacterCards,
			LocationCard:   builtContext.Params.LocationCard,
			Lore:           builtContext.Params.Lore,
			BranchSummary:  builtContext.Params.BranchSummary,
			BeatIntent:     builtContext.Params.BeatIntent,
			POV:            builtContext.Params.POV,
			Tone:           builtContext.Params.Tone,
			TargetWords:    builtContext.Params.TargetWords,
			Memories:       builtContext.Params.Memories,
		}
		if builtContext.Params.CharState != nil {
			cc.CharState = make(map[string]llm.CharacterState)
			for k, v := range builtContext.Params.CharState {
				b, _ := json.Marshal(v)
				var cs llm.CharacterState
				if json.Unmarshal(b, &cs) == nil {
					cc.CharState[k] = cs
				}
			}
		}
		var contextHash string
		if h, err := cc.Hash(); err == nil {
			contextHash = h
			gen.ContextHash = h
		}
		gen.PromptSnapshot = cc.BuildScenePromptSnapshot()
		_ = cfg.GenRepo.Update(ctx, gen)

		if contextHash != "" {
			existing, err := cfg.GenRepo.FindByContextHash(ctx, scene.StoryID, contextHash)
			if err == nil && existing != nil && existing.Output != "" && existing.ID != gen.ID {
				slog.Info("context hash cache hit, reusing generation",
					"sceneId", scene.ID, "genId", gen.ID, "existingGenId", existing.ID,
					"contextHash", contextHash)
				gen.Output = existing.Output
				gen.PromptTokens = existing.PromptTokens
				gen.CompletionTokens = existing.CompletionTokens
				gen.TotalTokens = existing.TotalTokens
				gen.Status = domain.GenStatusSuccess
				gen.StepStatus = map[string]string{
					domain.StepGenerate: "cached",
					domain.StepExtract:  "skipped",
					domain.StepMemory:   "skipped",
					domain.StepTimeline: "skipped",
					domain.StepSummary:  "skipped",
					domain.StepValidate: "skipped",
				}
				_ = cfg.GenRepo.Update(ctx, gen)
				scene.GeneratedContent = gen.Output
				_ = cfg.SceneRepo.Update(ctx, scene)
				return nil
			}
		}
	} else {
		params = llm.PromptParams{
			BeatIntent:  scene.BeatIntent,
			POV:         scene.POV,
			Tone:        scene.Tone,
			TargetWords: scene.TargetWords,
		}
	}

	sc.Artifacts["charNameToID"] = charNameToID

	if cfg.AgentSvc != nil && cfg.AgentSvc.IsAgentScene(scene) {
		slog.Info("hybrid pipeline: using agent orchestrator for generation", "sceneId", scene.ID)
		output, err := cfg.AgentSvc.GenerateSceneHybrid(ctx, scene, gen)
		if err != nil {
			return err
		}
		scene.GeneratedContent = output
		_ = cfg.SceneRepo.Update(ctx, scene)
		gen.Output = output
		return cfg.GenRepo.Update(ctx, gen)
	}

	_, err = genWorker.Work(ctx, worker.GenerateSceneArgs{
		SceneID: scene.ID,
		GenID:   gen.ID,
		Context: params,
	})
	if err != nil {
		return err
	}

	publishEvent(cfg.EventBus, events.Event{
		Type: events.EventSceneGenerated, StoryID: scene.StoryID, SceneID: scene.ID, GenID: gen.ID,
	})

	sc.Artifacts["genWorker"] = genWorker
	sc.Artifacts["extractWorker"] = extractWorker
	sc.Artifacts["memWorker"] = memWorker
	sc.Artifacts["tlWorker"] = tlWorker
	sc.Artifacts["sumWorker"] = sumWorker
	sc.Artifacts["valWorker"] = valWorker

	if elapsed := time.Since(gen.CreatedAt); elapsed > 0 {
		gen.DurationMs = elapsed.Milliseconds()
		gen.UpdatedAt = time.Now()
		_ = cfg.GenRepo.Update(ctx, gen)
	}

	return nil
}

func runExtractStep(ctx context.Context, sc *orchestration.StepContext, cfg GenerationJobWorkerConfig) error {
	scene, err := cfg.SceneRepo.Get(ctx, sc.SceneID)
	if err != nil || scene == nil {
		return fmt.Errorf("scene not found: %w", err)
	}

	charNameToID := sc.Artifacts["charNameToID"]
	charMap, _ := charNameToID.(map[string]string)

	extractWorker := worker.NewExtractStateWorker(cfg.ExtractSvc, cfg.StateRepo)
	sceneText := getSceneText(ctx, cfg.GenRepo, scene, sc.GenID)
	if sceneText == "" {
		return nil
	}

	if err := extractWorker.Work(ctx, worker.ExtractStateArgs{
		StoryID: scene.StoryID, SceneID: scene.ID, SceneText: sceneText,
		CharacterRefs: scene.Participants, CharNameToID: charMap,
	}); err != nil {
		return err
	}

	publishEvent(cfg.EventBus, events.Event{
		Type: events.EventCharacterStatesExtracted, StoryID: scene.StoryID, SceneID: scene.ID, GenID: sc.GenID,
	})

	if cfg.SceneValidator != nil {
		newStates, _ := cfg.StateRepo.ListByScene(ctx, scene.ID)
		var checks []validation.PostGenerationCheck
		for _, ns := range newStates {
			check := validation.PostGenerationCheck{
				CharacterID: ns.CharacterID,
				NewLocation: ns.Location,
				Learned:     extractLearned(ns.Changes),
			}
			prevStates, _ := cfg.StateRepo.ListByCharacter(ctx, ns.CharacterID)
			for _, ps := range prevStates {
				if ps.SceneID != scene.ID {
					check.PreviousLocation = ps.Location
					check.PreviousKnowledge = ps.Knowledge
					break
				}
			}
			checks = append(checks, check)
		}
		violations := cfg.SceneValidator.ValidatePostGeneration(ctx, scene, checks)
		for _, v := range violations {
			slog.Warn("post-generation validation", "genId", sc.GenID, "severity", v.Severity, "field", v.Field, "message", v.Message)
		}
	}

	if cfg.EventExtractor != nil && cfg.EventValidator != nil && cfg.EventRepo != nil {
		states, err := cfg.StateRepo.ListByScene(ctx, scene.ID)
		if err == nil && len(states) > 0 {
			statesSlice := make([]domain.CharacterState, len(states))
			for i, s := range states {
				statesSlice[i] = *s
			}
			candidates := cfg.EventExtractor.ExtractFromStates(ctx, event.ExtractorConfig{
				StoryID: scene.StoryID,
				SceneID: scene.ID,
				RunID:   sc.RunID,
				GenID:   sc.GenID,
			}, statesSlice)

			storyState := &event.StoryState{
				Characters: make(map[string]*domain.CharacterState),
				Timeline:   []domain.TimelineEvent{},
				Scene:      scene,
			}
			for _, s := range states {
				storyState.Characters[s.CharacterID] = s
			}
			if tlEvents, err := cfg.TlRepo.ListByStory(ctx, scene.StoryID); err == nil {
				for _, e := range tlEvents {
					if e != nil {
						storyState.Timeline = append(storyState.Timeline, *e)
					}
				}
			}

			accepted, rejected := cfg.EventValidator.Filter(ctx, candidates, storyState)
			if len(accepted) > 0 {
				acceptedPtrs := make([]*domain.NarrativeEvent, len(accepted))
				for i := range accepted {
					acceptedPtrs[i] = &accepted[i]
				}
				if err := cfg.EventRepo.AppendMany(ctx, acceptedPtrs); err != nil {
					slog.Error("failed to append narrative events", "genId", sc.GenID, "error", err)
				} else {
					slog.Info("narrative events appended", "genId", sc.GenID, "accepted", len(accepted), "rejected", len(rejected))
				}
			}
			if len(rejected) > 0 {
				sc.Artifacts["rejected_events"] = rejected
			}
		}
	}

	return nil
}

func runMemoryStep(ctx context.Context, sc *orchestration.StepContext, cfg GenerationJobWorkerConfig) error {
	scene, err := cfg.SceneRepo.Get(ctx, sc.SceneID)
	if err != nil || scene == nil {
		return fmt.Errorf("scene not found: %w", err)
	}

	memWorker := worker.NewMemoryUpdateWorker(cfg.MemRepo, cfg.EmbeddingSvc)
	sceneText := getSceneText(ctx, cfg.GenRepo, scene, sc.GenID)
	if sceneText == "" {
		return nil
	}

	for _, charID := range scene.Participants {
		if err := memWorker.Work(ctx, worker.MemoryUpdateArgs{
			StoryID: scene.StoryID, CharacterID: charID, SceneID: scene.ID,
			Content: sceneText, Importance: 0.5,
		}); err != nil {
			slog.Error("memory step failed", "genId", sc.GenID, "charId", charID, "error", err)
		}
	}

	publishEvent(cfg.EventBus, events.Event{
		Type: events.EventMemoriesCreated, StoryID: scene.StoryID, SceneID: scene.ID, GenID: sc.GenID,
	})
	return nil
}

func runTimelineStep(ctx context.Context, sc *orchestration.StepContext, cfg GenerationJobWorkerConfig) error {
	scene, err := cfg.SceneRepo.Get(ctx, sc.SceneID)
	if err != nil || scene == nil {
		return fmt.Errorf("scene not found: %w", err)
	}

	tlWorker := worker.NewTimelineWorker(cfg.TlRepo, cfg.EdgeRepo, cfg.BibleRepo)
	if err := tlWorker.Work(ctx, worker.TimelineArgs{
		StoryID: scene.StoryID, SceneID: scene.ID,
		Title: scene.Title, Order: scene.TimelinePosition,
	}); err != nil {
		return err
	}

	publishEvent(cfg.EventBus, events.Event{
		Type: events.EventTimelineRecorded, StoryID: scene.StoryID, SceneID: scene.ID, GenID: sc.GenID,
	})
	return nil
}

func runSummaryStep(ctx context.Context, sc *orchestration.StepContext, cfg GenerationJobWorkerConfig) error {
	scene, err := cfg.SceneRepo.Get(ctx, sc.SceneID)
	if err != nil || scene == nil {
		return fmt.Errorf("scene not found: %w", err)
	}

	sumWorker := worker.NewSummaryWorker(cfg.SummarySvc, cfg.SumRepo)
	sceneText := getSceneText(ctx, cfg.GenRepo, scene, sc.GenID)
	if sceneText == "" {
		return nil
	}

	prevSummary := ""
	if existing, _ := cfg.SumRepo.GetByLevel(ctx, scene.StoryID, domain.SummaryLevelStory); existing != nil {
		prevSummary = existing.Content
	}

	if err := sumWorker.Work(ctx, worker.SummaryArgs{
		StoryID: scene.StoryID, SceneID: scene.ID,
		PreviousSummary: prevSummary,
		AcceptedScene:   sceneText,
	}); err != nil {
		return err
	}

	publishEvent(cfg.EventBus, events.Event{
		Type: events.EventSummaryUpdated, StoryID: scene.StoryID, SceneID: scene.ID, GenID: sc.GenID,
	})
	return nil
}

func runValidateStep(ctx context.Context, sc *orchestration.StepContext, cfg GenerationJobWorkerConfig) error {
	scene, err := cfg.SceneRepo.Get(ctx, sc.SceneID)
	if err != nil || scene == nil {
		return fmt.Errorf("scene not found: %w", err)
	}

	valWorker := worker.NewValidationWorker(cfg.ValidateSvc, cfg.GenRepo)
	sceneText := getSceneText(ctx, cfg.GenRepo, scene, sc.GenID)
	if sceneText == "" {
		return nil
	}

	canonXML := ""
	if story, _ := cfg.StoryRepo.Get(ctx, scene.StoryID); story != nil {
		if len(story.CanonPins) > 0 {
			b, err := json.Marshal(story.CanonPins)
			if err != nil {
				slog.Error("marshal canon pins", "error", err)
			} else {
				canonXML = string(b)
			}
		}
	}
	charState := ""
	if states, _ := cfg.StateRepo.ListByScene(ctx, scene.ID); len(states) > 0 {
		b, err := json.Marshal(states)
		if err != nil {
			slog.Error("marshal char states", "error", err)
		} else {
			charState = string(b)
		}
	}

	if err := valWorker.Work(ctx, worker.ValidateArgs{
		GenerationID: sc.GenID, CanonXML: canonXML, CharState: charState, SceneText: sceneText,
	}); err != nil {
		return err
	}

	publishEvent(cfg.EventBus, events.Event{
		Type: events.EventSceneValidated, StoryID: scene.StoryID, SceneID: scene.ID, GenID: sc.GenID,
	})
	return nil
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
