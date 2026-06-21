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

type GenerationJobWorkerConfig struct {
	JobRepo        repository.JobRepository
	GenRepo        repository.GenerationRepository
	SceneRepo      repository.SceneRepository
	StoryRepo      repository.StoryRepository
	CharRepo       repository.CharacterRepository
	StateRepo      repository.CharacterStateRepository
	EdgeRepo       repository.SceneEdgeRepository
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
	Progress       ProgressPublisher
	AgentSvc       *AgentService
	PollInterval   time.Duration
	LeaseTime      time.Duration
}

type GenerationJobWorker struct {
	cfg    GenerationJobWorkerConfig
	stopCh chan struct{}
	wg     sync.WaitGroup
	genInFlight sync.Map
}

func NewGenerationJobWorker(cfg GenerationJobWorkerConfig) *GenerationJobWorker {
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 2 * time.Second
	}
	if cfg.LeaseTime == 0 {
		cfg.LeaseTime = 5 * time.Minute
	}
	return &GenerationJobWorker{
		cfg:    cfg,
		stopCh: make(chan struct{}),
	}
}

func (w *GenerationJobWorker) Start() {
	w.wg.Add(1)
	go w.loop()
	slog.Info("generation job worker started", "pollInterval", w.cfg.PollInterval)
}

func (w *GenerationJobWorker) Stop() {
	close(w.stopCh)
	w.wg.Wait()
	slog.Info("generation job worker stopped")
}

func (w *GenerationJobWorker) loop() {
	defer w.wg.Done()

	w.recoverStuckJobs()

	for {
		select {
		case <-w.stopCh:
			return
		case <-time.After(w.cfg.PollInterval):
			w.processNext()
		}
	}
}

func (w *GenerationJobWorker) recoverStuckJobs() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stuck, err := w.cfg.JobRepo.ListStuck(ctx, w.cfg.LeaseTime*2)
	if err != nil {
		slog.Warn("failed to list stuck jobs", "error", err)
		return
	}
	for _, j := range stuck {
		j.Status = domain.JobStatusFailed
		j.Error = "stuck (worker restart or crash)"
		if err := w.cfg.JobRepo.Update(ctx, j); err != nil {
			slog.Error("failed to mark stuck job as failed", "jobId", j.ID, "error", err)
		} else {
			slog.Warn("marked stuck job as failed", "jobId", j.ID, "genId", j.GenID)
		}

		if j.GenID != "" {
			if gen, err := w.cfg.GenRepo.Get(ctx, j.GenID); err == nil && gen != nil {
				gen.Status = domain.GenStatusFailed
				gen.Error = "worker restart during generation"
				_ = w.cfg.GenRepo.Update(ctx, gen)
			}
		}
	}
}

func (w *GenerationJobWorker) processNext() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	job, err := w.cfg.JobRepo.PickPending(ctx, domain.JobTypeGenerateScene, w.cfg.LeaseTime)
	if err != nil {
		slog.Error("failed to pick pending job", "error", err)
		return
	}
	if job == nil {
		return
	}

	slog.Info("picked generation job", "jobId", job.ID, "sceneId", job.SceneID, "attempt", job.Attempts)

	go func() {
		if _, loaded := w.genInFlight.LoadOrStore(job.SceneID, true); loaded {
			slog.Warn("generation already in flight for scene, skipping job", "sceneId", job.SceneID, "jobId", job.ID)
			w.failJob(job, "generation already in flight")
			return
		}
		defer w.genInFlight.Delete(job.SceneID)

		pCtx, pCancel := context.WithTimeout(context.Background(), w.cfg.LeaseTime)
		defer pCancel()

		start := time.Now()
		gen, err := w.cfg.GenRepo.Get(pCtx, job.GenID)
		if err != nil || gen == nil {
			w.failJob(job, fmt.Sprintf("generation not found: %v", err))
			return
		}

		gen.Status = domain.GenStatusRunning
		_ = w.cfg.GenRepo.Update(pCtx, gen)

		scene, err := w.cfg.SceneRepo.Get(pCtx, job.SceneID)
		if err != nil || scene == nil {
			w.failJob(job, fmt.Sprintf("scene not found: %v", err))
			gen.Status = domain.GenStatusFailed
			gen.Error = "scene not found"
			_ = w.cfg.GenRepo.Update(pCtx, gen)
			return
		}

		defer func() {
			if r := recover(); r != nil {
				slog.Error("pipeline panic", "genId", gen.ID, "jobId", job.ID, "recover", r)
				gen.Status = domain.GenStatusFailed
				gen.Error = fmt.Sprintf("pipeline panic: %v", r)
				_ = w.cfg.GenRepo.Update(context.Background(), gen)
				w.failJob(job, fmt.Sprintf("pipeline panic: %v", r))
			}
		}()

		w.runPipeline(pCtx, gen, scene, job)

		elapsed := time.Since(start)
		gen.DurationMs = elapsed.Milliseconds()
		_ = w.cfg.GenRepo.Update(pCtx, gen)

		job.Status = domain.JobStatusDone
		_ = w.cfg.JobRepo.Update(pCtx, job)

		slog.Info("generation job complete", "jobId", job.ID, "genId", gen.ID, "duration", elapsed)
	}()
}

func (w *GenerationJobWorker) failJob(job *domain.Job, errMsg string) {
	job.Status = domain.JobStatusFailed
	job.Error = errMsg
	_ = w.cfg.JobRepo.Update(context.Background(), job)
}

func (w *GenerationJobWorker) setStepStatus(ctx context.Context, genID, step, status string) {
	if err := w.cfg.GenRepo.SetStepStatus(ctx, genID, step, status); err != nil {
		slog.Error("set step status failed", "genId", genID, "step", step, "status", status, "error", err)
	}
	if w.cfg.Progress != nil {
		w.cfg.Progress.Publish(genID, ProgressEvent{GenID: genID, Step: step, Status: status})
	}
}

func (w *GenerationJobWorker) publishEvent(ctx context.Context, evt events.Event) {
	if w.cfg.EventBus != nil {
		w.cfg.EventBus.Publish(ctx, evt)
	}
}

func (w *GenerationJobWorker) setGenError(ctx context.Context, genID, errMsg string) {
	gen, err := w.cfg.GenRepo.Get(ctx, genID)
	if err != nil || gen == nil {
		return
	}
	gen.Error = errMsg
	_ = w.cfg.GenRepo.Update(ctx, gen)
}

func (w *GenerationJobWorker) setGenStatus(ctx context.Context, genID, status string) {
	gen, err := w.cfg.GenRepo.Get(ctx, genID)
	if err != nil || gen == nil {
		return
	}
	gen.Status = status
	_ = w.cfg.GenRepo.Update(ctx, gen)
}

func (w *GenerationJobWorker) runStep(ctx context.Context, genID, stepName string, fn func(context.Context) error) bool {
	w.setStepStatus(ctx, genID, stepName, domain.StepRunning)

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			slog.Info("retrying step", "genId", genID, "step", stepName, "attempt", attempt)
		}
		lastErr = fn(ctx)
		if lastErr == nil {
			w.setStepStatus(ctx, genID, stepName, domain.StepDone)
			return true
		}
		if ctx.Err() != nil {
			break
		}
	}

	slog.Error("step failed after retries", "genId", genID, "step", stepName, "error", lastErr)
	w.setStepStatus(ctx, genID, stepName, domain.StepFailed)
	return false
}

func (w *GenerationJobWorker) runNonCriticalStep(ctx context.Context, genID, stepName string, fn func(context.Context) error) {
	if ctx.Err() != nil {
		return
	}
	w.setStepStatus(ctx, genID, stepName, domain.StepRunning)
	if err := fn(ctx); err != nil {
		slog.Warn("non-critical step failed", "genId", genID, "step", stepName, "error", err)
		w.setStepStatus(ctx, genID, stepName, domain.StepFailed)
		return
	}
	w.setStepStatus(ctx, genID, stepName, domain.StepDone)
}

func (w *GenerationJobWorker) runPipeline(ctx context.Context, gen *domain.Generation, scene *domain.Scene, job *domain.Job) {
	genWorker := worker.NewGenerateSceneWorker(w.cfg.ProseSvc, w.cfg.GenRepo, w.cfg.SceneRepo)
	extractWorker := worker.NewExtractStateWorker(w.cfg.ExtractSvc, w.cfg.StateRepo)
	memWorker := worker.NewMemoryUpdateWorker(w.cfg.MemRepo, w.cfg.EmbeddingSvc)
	tlWorker := worker.NewTimelineWorker(w.cfg.TlRepo, w.cfg.EdgeRepo)
	sumWorker := worker.NewSummaryWorker(w.cfg.SummarySvc, w.cfg.SumRepo)
	valWorker := worker.NewValidationWorker(w.cfg.ValidateSvc, w.cfg.GenRepo)

	slog.Info("generation pipeline starting", "genId", gen.ID, "jobId", job.ID)

	if w.cfg.SceneValidator != nil {
		violations := w.cfg.SceneValidator.ValidatePreGeneration(ctx, scene)
		for _, v := range violations {
			slog.Warn("pre-generation validation", "genId", gen.ID, "severity", v.Severity, "field", v.Field, "message", v.Message)
		}
	}

	criticalFailed := false
	anyFailed := false

	var builtContext *BuiltContext

	if w.cfg.ContextBldr != nil {
		bCtx, err := w.cfg.ContextBldr.Build(ctx, scene)
		if err != nil {
			slog.Warn("context builder failed, falling back to simple params", "genId", gen.ID, "error", err)
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

		// Record context hash and prompt snapshot for observability.
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
		if h, err := cc.Hash(); err == nil {
			gen.ContextHash = h
		}
		gen.PromptSnapshot = cc.BuildScenePromptSnapshot()
		_ = w.cfg.GenRepo.Update(ctx, gen)
	} else {
		params = llm.PromptParams{
			BeatIntent:  scene.BeatIntent,
			POV:         scene.POV,
			Tone:        scene.Tone,
			TargetWords: scene.TargetWords,
		}
	}

	if w.cfg.AgentSvc != nil && w.cfg.AgentSvc.IsAgentScene(scene) {
		slog.Info("hybrid pipeline: using agent orchestrator for generation", "sceneId", scene.ID)
		if !w.runStep(ctx, gen.ID, domain.StepGenerate, func(sCtx context.Context) error {
			output, err := w.cfg.AgentSvc.GenerateSceneHybrid(sCtx, scene, gen)
			if err != nil {
				return err
			}
			scene.GeneratedContent = output
			_ = w.cfg.SceneRepo.Update(sCtx, scene)
			gen.Output = output
			return w.cfg.GenRepo.Update(sCtx, gen)
		}) {
			criticalFailed = true
			anyFailed = true
		}
	} else if !w.runStep(ctx, gen.ID, domain.StepGenerate, func(sCtx context.Context) error {
		_, err := genWorker.Work(sCtx, worker.GenerateSceneArgs{
			SceneID: scene.ID,
			GenID:   gen.ID,
			Context: params,
		})
		return err
	}) {
		criticalFailed = true
		anyFailed = true
	}

	w.publishEvent(ctx, events.Event{
		Type: events.EventSceneGenerated, StoryID: scene.StoryID, SceneID: scene.ID, GenID: gen.ID,
		Data: map[string]any{"criticalFailed": criticalFailed},
	})

	sceneText, _ := func() (string, error) {
		g, err := w.cfg.GenRepo.Get(ctx, gen.ID)
		if err != nil || g == nil {
			return "", err
		}
		return g.Output, nil
	}()
	if sceneText == "" {
		sceneText = scene.GeneratedContent
	}

	if !criticalFailed && sceneText != "" {
		if !w.runStep(ctx, gen.ID, domain.StepExtract, func(sCtx context.Context) error {
			return extractWorker.Work(sCtx, worker.ExtractStateArgs{
				StoryID: scene.StoryID, SceneID: scene.ID, SceneText: sceneText,
				CharacterRefs: scene.Participants, CharNameToID: charNameToID,
			})
		}) {
			anyFailed = true
		}
		w.publishEvent(ctx, events.Event{
			Type: events.EventCharacterStatesExtracted, StoryID: scene.StoryID, SceneID: scene.ID, GenID: gen.ID,
		})

		if w.cfg.SceneValidator != nil {
			newStates, _ := w.cfg.StateRepo.ListByScene(ctx, scene.ID)
			var checks []validation.PostGenerationCheck
			for _, ns := range newStates {
				check := validation.PostGenerationCheck{
					CharacterID: ns.CharacterID,
					NewLocation: ns.Location,
					Learned:     extractLearned(ns.Changes),
				}
				prevStates, _ := w.cfg.StateRepo.ListByCharacter(ctx, ns.CharacterID)
				for _, ps := range prevStates {
					if ps.SceneID != scene.ID {
						check.PreviousLocation = ps.Location
						check.PreviousKnowledge = ps.Knowledge
						break
					}
				}
				checks = append(checks, check)
			}
			violations := w.cfg.SceneValidator.ValidatePostGeneration(ctx, scene, checks)
			for _, v := range violations {
				slog.Warn("post-generation validation", "genId", gen.ID, "severity", v.Severity, "field", v.Field, "message", v.Message)
			}
		}
	}

	if sceneText != "" {
		w.runNonCriticalStep(ctx, gen.ID, domain.StepMemory, func(sCtx context.Context) error {
			for _, charID := range scene.Participants {
				if err := memWorker.Work(sCtx, worker.MemoryUpdateArgs{
					StoryID: scene.StoryID, CharacterID: charID, SceneID: scene.ID,
					Content: sceneText, Importance: 0.5,
				}); err != nil {
					slog.Error("memory step failed", "genId", gen.ID, "charId", charID, "error", err)
				}
			}
			return nil
		})
		w.publishEvent(ctx, events.Event{
			Type: events.EventMemoriesCreated, StoryID: scene.StoryID, SceneID: scene.ID, GenID: gen.ID,
		})
	}

	w.runNonCriticalStep(ctx, gen.ID, domain.StepTimeline, func(sCtx context.Context) error {
		return tlWorker.Work(sCtx, worker.TimelineArgs{
			StoryID: scene.StoryID, SceneID: scene.ID,
			Title: scene.Title, Order: scene.TimelinePosition,
		})
	})
	w.publishEvent(ctx, events.Event{
		Type: events.EventTimelineRecorded, StoryID: scene.StoryID, SceneID: scene.ID, GenID: gen.ID,
	})

	if sceneText != "" {
		w.runNonCriticalStep(ctx, gen.ID, domain.StepSummary, func(sCtx context.Context) error {
			prevSummary := ""
			if existing, _ := w.cfg.SumRepo.GetByLevel(sCtx, scene.StoryID, domain.SummaryLevelStory); existing != nil {
				prevSummary = existing.Content
			}
			return sumWorker.Work(sCtx, worker.SummaryArgs{
				StoryID: scene.StoryID, SceneID: scene.ID,
				PreviousSummary: prevSummary,
				AcceptedScene:   sceneText,
			})
		})
		w.publishEvent(ctx, events.Event{
			Type: events.EventSummaryUpdated, StoryID: scene.StoryID, SceneID: scene.ID, GenID: gen.ID,
		})
	}

	if sceneText != "" {
		w.runNonCriticalStep(ctx, gen.ID, domain.StepValidate, func(sCtx context.Context) error {
			canonXML := ""
			if story, _ := w.cfg.StoryRepo.Get(sCtx, scene.StoryID); story != nil {
				if len(story.CanonPins) > 0 {
					b, err := json.Marshal(story.CanonPins)
					if err != nil {
						slog.Error("marshal canon pins", "genId", gen.ID, "error", err)
					} else {
						canonXML = string(b)
					}
				}
			}
			charState := ""
			if states, _ := w.cfg.StateRepo.ListByScene(sCtx, scene.ID); len(states) > 0 {
				b, err := json.Marshal(states)
				if err != nil {
					slog.Error("marshal char states", "genId", gen.ID, "error", err)
				} else {
					charState = string(b)
				}
			}
			return valWorker.Work(sCtx, worker.ValidateArgs{
				GenerationID: gen.ID, CanonXML: canonXML, CharState: charState, SceneText: sceneText,
			})
		})
		w.publishEvent(ctx, events.Event{
			Type: events.EventSceneValidated, StoryID: scene.StoryID, SceneID: scene.ID, GenID: gen.ID,
		})
	}

	switch {
	case criticalFailed:
		w.setGenStatus(ctx, gen.ID, domain.GenStatusFailed)
		w.setGenError(ctx, gen.ID, "generate step failed after retries")
		w.failJob(job, "generate step failed after retries")
		slog.Error("generation pipeline failed", "genId", gen.ID)
		w.publishEvent(ctx, events.Event{
			Type: events.EventPipelineFailed, StoryID: scene.StoryID, SceneID: scene.ID, GenID: gen.ID,
		})
	case anyFailed:
		w.setGenStatus(ctx, gen.ID, domain.GenStatusPartialSuccess)
		slog.Warn("generation pipeline completed with partial failures", "genId", gen.ID)
		w.publishEvent(ctx, events.Event{
			Type: events.EventPipelineComplete, StoryID: scene.StoryID, SceneID: scene.ID, GenID: gen.ID,
			Data: map[string]any{"status": domain.GenStatusPartialSuccess},
		})
	default:
		w.setGenStatus(ctx, gen.ID, domain.GenStatusSuccess)
		slog.Info("generation pipeline complete", "genId", gen.ID)
		w.publishEvent(ctx, events.Event{
			Type: events.EventPipelineComplete, StoryID: scene.StoryID, SceneID: scene.ID, GenID: gen.ID,
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
