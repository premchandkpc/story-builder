package main

import (
	"log/slog"
	"time"

	"github.com/premchand/story-builder/internal/agents"
	"github.com/premchand/story-builder/internal/api"
	"github.com/premchand/story-builder/internal/cache"
	"github.com/premchand/story-builder/internal/config"
	"github.com/premchand/story-builder/internal/event"
	"github.com/premchand/story-builder/internal/event/rules"
	"github.com/premchand/story-builder/internal/events"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/prompt"
	mgorepo "github.com/premchand/story-builder/internal/repository/mongo"
	"github.com/premchand/story-builder/internal/service"
	"github.com/premchand/story-builder/internal/validation"
	"go.mongodb.org/mongo-driver/mongo"
)

type appDependencies struct {
	storySvc    *service.StoryService
	sceneSvc    *service.SceneService
	edgeSvc     *service.EdgeService
	charSvc     *service.CharacterService
	locSvc      *service.LocationService
	genSvc      *service.GenerationService
	metricsSvc  *service.MetricsService
	criticSvc   *service.CriticScoresService
	agentCfgSvc *service.AgentConfigService
	tlSvc       *service.TimelineService
	sumSvc      *service.SummaryService
	memSvc      *service.MemoryService
	bibleSvc    *service.BibleService
	chapterSvc  *service.ChapterSvc
	outlineSvc  llm.OutlineService
	titleSvc    llm.TitleService
	genJobWorker *service.GenerationJobWorker
	rateLimiter  *cache.SlidingWindowRateLimiter
	progressHub  *api.ProgressHub
	eventBus     events.Bus
	agentSvc     *service.AgentService
	runSvc       *service.RunService
	narrativeSvc *service.NarrativeEventService
}

func initAll(cfg config.Config, db *mongo.Database) appDependencies {
	// Repositories
	storyRepo := mgorepo.NewStoryRepo(db)
	sceneRepo := mgorepo.NewSceneRepo(db)
	edgeRepo := mgorepo.NewSceneEdgeRepo(db)
	charRepo := mgorepo.NewCharacterRepo(db)
	stateRepo := mgorepo.NewCharacterStateRepo(db)
	genRepo := mgorepo.NewGenerationRepo(db)
	memRepo := mgorepo.NewMemoryRepo(db)
	tlRepo := mgorepo.NewTimelineRepo(db)
	sumRepo := mgorepo.NewSummaryRepo(db)
	bibleRepo := mgorepo.NewBibleRepo(db)
	chapterRepo := mgorepo.NewChapterRepo(db)
	jobRepo := mgorepo.NewJobRepo(db)
	locRepo := mgorepo.NewLocationRepo(db)
	turnRepo := mgorepo.NewSceneTurnRepo(db)
	agentRunRepo := mgorepo.NewAgentRunRepo(db)
	canonDeltaRepo := mgorepo.NewCanonDeltaRepo(db)
	runRepo := mgorepo.NewRunRepo(db)
	stepRepo := mgorepo.NewRunStepRepo(db)

	// LLM clients and router
	var anthropic llm.LLMClient
	if cfg.AnthropicKey != "" {
		anthropic = llm.NewCircuitBreakerClient(llm.NewAnthropicClient(cfg.AnthropicKey))
		slog.Info("anthropic client created")
	} else {
		slog.Warn("no ANTHROPIC_API_KEY set, using OpenCode for all tiers")
		anthropic = llm.NewCircuitBreakerClient(llm.NewOpenCodeClient(cfg.OpenCodeURL, cfg.OpenCodeModel, cfg.OpenCodeKey))
	}
	var opencode llm.LLMClient
	opencode = llm.NewCircuitBreakerClient(llm.NewOpenCodeClient(cfg.OpenCodeURL, cfg.OpenCodeModel, cfg.OpenCodeKey))

	llm.SetDefaultConfig(cfg.OpenCodeURL, cfg.OpenCodeModel)
	llm.SetDefaultAPIKey(cfg.OpenCodeKey)
	llm.SetDefaultEmbedModel("nomic-embed-text")

	if cfg.HeadroomURL != "" {
		anthropic = llm.NewCompressClient(anthropic, cfg.HeadroomURL, cfg.HeadroomKey)
		opencode = llm.NewCompressClient(opencode, cfg.HeadroomURL, cfg.HeadroomKey)
		slog.Info("headroom compression enabled", "url", cfg.HeadroomURL)
	} else {
		slog.Info("no HEADROOM_BASE_URL set, compression disabled")
	}

	router := llm.NewRouter(anthropic, opencode)

	// Prompt compiler
	promptStore := prompt.NewMemoryStore()
	for _, tmpl := range prompt.DefaultTemplates() {
		_ = promptStore.Save(tmpl)
	}
	promptCompiler := prompt.NewCompilerService(promptStore)

	// LLM services
	proseSvc := llm.NewProseService(router, promptCompiler)
	extractSvc := llm.NewExtractionService(router, promptCompiler)
	summarySvc := llm.NewSummaryService(router, promptCompiler)
	validateSvc := llm.NewValidationService(router, promptCompiler)
	outlineSvc := llm.NewOutlineService(router, promptCompiler)
	titleSvc := llm.NewTitleService(router)
	bibleGenSvc := llm.NewBibleService(router, promptCompiler)

	slog.Info("llm services initialized",
		"prose", true, "extract", true, "summary", true,
		"validate", true, "outline", outlineSvc != nil,
		"title", true,
	)

	// Rate limiter / cache
	var rateLimiter *cache.SlidingWindowRateLimiter
	var redisClient cache.RedisClient
	if cfg.RedisAddr != "" {
		var err error
		redisClient, err = cache.NewGoRedisClient(cfg.RedisAddr, cfg.RedisPass, 0)
		if err != nil {
			slog.Warn("redis unavailable, running without cache", "error", err)
		} else {
			rateLimiter = cache.NewSlidingWindowRateLimiter(redisClient, cache.DefaultRateLimits)
			anthropic = llm.NewCachedLLMClient(anthropic, redisClient, 1*time.Hour)
			opencode = llm.NewCachedLLMClient(opencode, redisClient, 1*time.Hour)
			slog.Info("redis cache enabled with prompt caching")
		}
	} else {
		slog.Info("no REDIS_ADDR set, running without cache")
	}

	// Domain services
	storySvc := service.NewStoryService(storyRepo, &service.StoryCascadeDeleter{
		SceneRepo: sceneRepo, EdgeRepo: edgeRepo, CharRepo: charRepo,
		StateRepo: stateRepo, GenRepo: genRepo, MemRepo: memRepo,
		TlRepo: tlRepo, SumRepo: sumRepo, LocRepo: locRepo,
		BibleRepo: bibleRepo, ChapterRepo: chapterRepo,
	})
	sceneSvc := service.NewSceneService(sceneRepo, edgeRepo, genRepo)
	edgeSvc := service.NewEdgeService(edgeRepo)
	charSvc := service.NewCharacterService(charRepo, stateRepo, memRepo)
	locSvc := service.NewLocationService(locRepo)

	contextBldr := service.NewContextBuilder(bibleRepo, storyRepo, charRepo, stateRepo, locRepo, memRepo, sumRepo, tlRepo)
	bibleSvc := service.NewBibleService(bibleRepo, storyRepo, charRepo, bibleGenSvc)
	chapterSvc := service.NewChapterSvc(chapterRepo)

	eventBus := events.NewInMemoryBus()
	embedSvc := llm.NewOpenCodeEmbeddingService(cfg.OpenCodeURL, "nomic-embed-text")
	sceneValidator := validation.NewSceneValidator(charRepo, locRepo)

	narrativeEventRepo := mgorepo.NewNarrativeEventRepo(db)
	charViewRepo := mgorepo.NewCharacterViewRepo(db)
	sceneLockRepo := mgorepo.NewSceneLockRepo(db)
	eventExtractor := event.NewEventExtractor()
	eventValidator := event.NewEventValidator([]event.EventValidationRule{
		&rules.DeadCharacterCannotAct{},
		&rules.TimelineMonotonicity{},
		rules.NewLocationConsistency(nil),
		&rules.ValueBounds{},
		&rules.DuplicateDetector{},
	})
	_, _ = charViewRepo, sceneLockRepo

	agentRegistry := agents.NewAgentRegistry()
	agents.RegisterAll(agentRegistry, router, proseSvc, extractSvc, validateSvc)
	slog.Info("agent registry initialized", "count", len(agentRegistry.List()))

	charManager := agents.NewCharacterManager(agentRegistry, router, proseSvc, eventBus)

	budgetRepo := mgorepo.NewTokenBudgetRepo(db)
	budgetSvc := service.NewTokenBudgetService(budgetRepo)

	agentOrchestrator := agents.NewOrchestrator(agents.OrchestratorConfig{
		Registry:      agentRegistry,
		LLMClient:     router,
		EventBus:      eventBus,
		CharManager:   charManager,
		BudgetChecker: budgetSvc,
	})

	agentSvc := service.NewAgentService(service.AgentServiceConfig{
		Registry: agentRegistry, Orchestrator: agentOrchestrator, CharManager: charManager,
		TurnRepo: turnRepo, ActorRepo: agentRunRepo, CanonRepo: canonDeltaRepo,
		GenRepo: genRepo, StoryRepo: storyRepo, SceneRepo: sceneRepo,
		CharRepo: charRepo, StateRepo: stateRepo,
		BibleRepo: bibleRepo, EdgeRepo: edgeRepo, MemRepo: memRepo,
		TlRepo: tlRepo, SumRepo: sumRepo, BudgetSvc: budgetSvc,
	})

	genSvc := service.NewGenerationService(service.GenerationServiceConfig{
		GenRepo: genRepo, SceneRepo: sceneRepo, JobRepo: jobRepo,
		EventBus: eventBus, AgentSvc: agentSvc, BudgetSvc: budgetSvc,
	})

	progressHub := api.NewProgressHub()
	genJobWorker := service.NewGenerationJobWorker(service.GenerationJobWorkerConfig{
		JobRepo: jobRepo, RunRepo: runRepo, StepRepo: stepRepo,
		GenRepo: genRepo, SceneRepo: sceneRepo,
		StoryRepo: storyRepo, CharRepo: charRepo, StateRepo: stateRepo,
		EdgeRepo: edgeRepo, BibleRepo: bibleRepo, MemRepo: memRepo, TlRepo: tlRepo,
		SumRepo: sumRepo, LocRepo: locRepo,
		ProseSvc: proseSvc, ExtractSvc: extractSvc,
		SummarySvc: summarySvc, ValidateSvc: validateSvc,
		ContextBldr: contextBldr, EventBus: eventBus,
		EmbeddingSvc: embedSvc, SceneValidator: sceneValidator,
		EventRepo: narrativeEventRepo, EventExtractor: eventExtractor,
		EventValidator: eventValidator,
		Progress: progressHub, AgentSvc: agentSvc,
		PollInterval: 5 * time.Second,
		LeaseTime:    5 * time.Minute,
	})

	tlSvc := service.NewTimelineService(tlRepo)
	sumSvc := service.NewSummaryService(sumRepo)
	memSvc := service.NewMemoryService(memRepo, embedSvc)
	runSvc := service.NewRunService(runRepo, stepRepo, jobRepo)
	narrativeSvc := service.NewNarrativeEventService(narrativeEventRepo)
	metricsSvc := service.NewMetricsService(genRepo)
	criticSvc := service.NewCriticScoresService(genRepo, sceneRepo)
	agentCfgRepo := mgorepo.NewAgentConfigRepo(db)
	agentCfgSvc := service.NewAgentConfigService(agentCfgRepo)

	return appDependencies{
		storySvc: storySvc, sceneSvc: sceneSvc, edgeSvc: edgeSvc,
		charSvc: charSvc, locSvc: locSvc, genSvc: genSvc,
		metricsSvc: metricsSvc, criticSvc: criticSvc, agentCfgSvc: agentCfgSvc,
		tlSvc: tlSvc, sumSvc: sumSvc, memSvc: memSvc,
		bibleSvc: bibleSvc, chapterSvc: chapterSvc,
		outlineSvc: outlineSvc, titleSvc: titleSvc,
		genJobWorker: genJobWorker,
		rateLimiter:  rateLimiter,
		progressHub:  progressHub,
		eventBus:     eventBus,
		agentSvc:     agentSvc,
		runSvc:       runSvc,
		narrativeSvc: narrativeSvc,
	}
}
