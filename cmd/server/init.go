package main

import (
	"log/slog"

	"github.com/premchand/story-builder/internal/agents"
	"github.com/premchand/story-builder/internal/api"
	"github.com/premchand/story-builder/internal/cache"
	"github.com/premchand/story-builder/internal/config"
	"github.com/premchand/story-builder/internal/events"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/prompt"
	mgorepo "github.com/premchand/story-builder/internal/repository/mongo"
	"github.com/premchand/story-builder/internal/service"
	"github.com/premchand/story-builder/internal/validation"
	"go.mongodb.org/mongo-driver/mongo"
)

type appDependencies struct {
	storySvc   *service.StoryService
	sceneSvc   *service.SceneService
	edgeSvc    *service.EdgeService
	charSvc    *service.CharacterService
	locSvc     *service.LocationService
	genSvc     *service.GenerationService
	metricsSvc *service.MetricsService
	tlSvc      *service.TimelineService
	sumSvc     *service.SummaryService
	memSvc     *service.MemoryService
	bibleSvc   *service.BibleService
	chapterSvc *service.ChapterSvc
	outlineSvc llm.OutlineService
	titleSvc   llm.TitleService
	genJobWorker *service.GenerationJobWorker
	rateLimiter  *cache.SlidingWindowRateLimiter
	progressHub  *api.ProgressHub
	eventBus     events.Bus
	agentSvc     *service.AgentService
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

	// LLM clients and router
	var anthropic llm.LLMClient
	if cfg.AnthropicKey != "" {
		anthropic = llm.NewCircuitBreakerClient(llm.NewAnthropicClient(cfg.AnthropicKey))
		slog.Info("anthropic client created")
	} else {
		slog.Warn("no ANTHROPIC_API_KEY set, using Ollama for all tiers")
		anthropic = llm.NewCircuitBreakerClient(llm.NewOllamaClient(cfg.OllamaURL, cfg.OllamaModel))
	}
	ollama := llm.NewCircuitBreakerClient(llm.NewOllamaClient(cfg.OllamaURL, cfg.OllamaModel))
	router := llm.NewRouter(anthropic, ollama)

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
	if cfg.RedisAddr != "" {
		redisClient, err := cache.NewGoRedisClient(cfg.RedisAddr, cfg.RedisPass, 0)
		if err != nil {
			slog.Warn("redis unavailable, running without cache", "error", err)
		} else {
			rateLimiter = cache.NewSlidingWindowRateLimiter(redisClient, cache.DefaultRateLimits)
			slog.Info("redis cache enabled")
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
	charSvc := service.NewCharacterService(charRepo)
	locSvc := service.NewLocationService(locRepo)

	contextBldr := service.NewContextBuilder(bibleRepo, storyRepo, charRepo, stateRepo, locRepo, memRepo, sumRepo, tlRepo)
	bibleSvc := service.NewBibleService(bibleRepo, storyRepo, charRepo, bibleGenSvc)
	chapterSvc := service.NewChapterSvc(chapterRepo)

	eventBus := events.NewInMemoryBus()
	embedSvc := llm.NewOllamaEmbeddingService(cfg.OllamaURL, "nomic-embed-text")
	sceneValidator := validation.NewSceneValidator(charRepo, locRepo)

	agentRegistry := agents.NewAgentRegistry()
	agents.RegisterAll(agentRegistry, router, proseSvc, extractSvc, validateSvc)
	slog.Info("agent registry initialized", "count", len(agentRegistry.List()))

	agentOrchestrator := agents.NewOrchestrator(agents.OrchestratorConfig{
		Registry:  agentRegistry,
		LLMClient: router,
		EventBus:  eventBus,
	})

	agentSvc := service.NewAgentService(service.AgentServiceConfig{
		Registry: agentRegistry, Orchestrator: agentOrchestrator,
		TurnRepo: turnRepo, ActorRepo: agentRunRepo, CanonRepo: canonDeltaRepo,
		GenRepo: genRepo, StoryRepo: storyRepo, SceneRepo: sceneRepo,
		CharRepo: charRepo, StateRepo: stateRepo,
		BibleRepo: bibleRepo, EdgeRepo: edgeRepo, MemRepo: memRepo,
		TlRepo: tlRepo, SumRepo: sumRepo,
	})

	genSvc := service.NewGenerationService(service.GenerationServiceConfig{
		GenRepo: genRepo, SceneRepo: sceneRepo, JobRepo: jobRepo,
		EventBus: eventBus, AgentSvc: agentSvc,
	})

	progressHub := api.NewProgressHub()
	genJobWorker := service.NewGenerationJobWorker(service.GenerationJobWorkerConfig{
		JobRepo: jobRepo, GenRepo: genRepo, SceneRepo: sceneRepo,
		StoryRepo: storyRepo, CharRepo: charRepo, StateRepo: stateRepo,
		EdgeRepo: edgeRepo, MemRepo: memRepo, TlRepo: tlRepo,
		SumRepo: sumRepo, LocRepo: locRepo,
		ProseSvc: proseSvc, ExtractSvc: extractSvc,
		SummarySvc: summarySvc, ValidateSvc: validateSvc,
		ContextBldr: contextBldr, EventBus: eventBus,
		EmbeddingSvc: embedSvc, SceneValidator: sceneValidator,
		Progress: progressHub,
	})

	tlSvc := service.NewTimelineService(tlRepo)
	sumSvc := service.NewSummaryService(sumRepo)
	memSvc := service.NewMemoryService(memRepo, embedSvc)
	metricsSvc := service.NewMetricsService(genRepo)

	return appDependencies{
		storySvc: storySvc, sceneSvc: sceneSvc, edgeSvc: edgeSvc,
		charSvc: charSvc, locSvc: locSvc, genSvc: genSvc,
		metricsSvc: metricsSvc,
		tlSvc: tlSvc, sumSvc: sumSvc, memSvc: memSvc,
		bibleSvc: bibleSvc, chapterSvc: chapterSvc,
		outlineSvc: outlineSvc, titleSvc: titleSvc,
		genJobWorker: genJobWorker,
		rateLimiter:  rateLimiter,
		progressHub:  progressHub,
		eventBus:     eventBus,
		agentSvc:     agentSvc,
	}
}
