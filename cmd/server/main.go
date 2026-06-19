package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/premchand/story-builder/internal/api"
	"github.com/premchand/story-builder/internal/cache"
	"github.com/premchand/story-builder/internal/config"
	"github.com/premchand/story-builder/internal/events"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/log"
	"github.com/premchand/story-builder/internal/prompt"
	mgorepo "github.com/premchand/story-builder/internal/repository/mongo"
	"github.com/premchand/story-builder/internal/service"
	"github.com/premchand/story-builder/internal/validation"
)

func main() {
	cfg := config.FromEnv()
	log.Init(log.Config{Level: cfg.LogLevel})
	slog.Info("starting story-builder", "port", cfg.Port)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	db, err := mgorepo.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		slog.Error("mongo connection failed", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to mongo", "db", cfg.MongoDB)

	if err := mgorepo.EnsureIndexes(ctx, db); err != nil {
		slog.Error("mongo indexes failed", "error", err)
		os.Exit(1)
	}
	slog.Info("mongo indexes ensured")

	// ─── Repositories ──────────────────────────────────────
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

	// ─── LLM Clients ───────────────────────────────────────
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

	// ─── Prompt Compiler ───────────────────────────────────
	promptStore := prompt.NewMemoryStore()
	for _, tmpl := range prompt.DefaultTemplates() {
		_ = promptStore.Save(tmpl)
	}
	promptCompiler := prompt.NewCompilerService(promptStore)

	// ─── LLM Services ──────────────────────────────────────
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

	// ─── Cache ─────────────────────────────────────────────
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

	locRepo := mgorepo.NewLocationRepo(db)

	// ─── Services ──────────────────────────────────────────
	storySvc := service.NewStoryService(storyRepo, &service.StoryCascadeDeleter{
		SceneRepo:   sceneRepo,
		EdgeRepo:    edgeRepo,
		CharRepo:    charRepo,
		StateRepo:   stateRepo,
		GenRepo:     genRepo,
		MemRepo:     memRepo,
		TlRepo:      tlRepo,
		SumRepo:     sumRepo,
		LocRepo:     locRepo,
		BibleRepo:   bibleRepo,
		ChapterRepo: chapterRepo,
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
	genSvc := service.NewGenerationService(service.GenerationServiceConfig{
		GenRepo: genRepo, SceneRepo: sceneRepo, StoryRepo: storyRepo,
		CharRepo: charRepo, StateRepo: stateRepo, EdgeRepo: edgeRepo,
		MemRepo: memRepo, TlRepo: tlRepo, SumRepo: sumRepo, LocRepo: locRepo,
		ProseSvc: proseSvc, ExtractSvc: extractSvc, SummarySvc: summarySvc, ValidateSvc: validateSvc,
		ContextBldr: contextBldr, EventBus: eventBus, EmbeddingSvc: embedSvc,
		SceneValidator: sceneValidator,
	})
	tlSvc := service.NewTimelineService(tlRepo)
	sumSvc := service.NewSummaryService(sumRepo)
	memSvc := service.NewMemoryService(memRepo, embedSvc)

	// ─── Progress Hub ──────────────────────────────────────
	progressHub := api.NewProgressHub()
	genSvc.SetProgressPublisher(progressHub)

	// ─── Handlers ──────────────────────────────────────────
	h := api.NewHandlers(storySvc, sceneSvc, edgeSvc, charSvc, genSvc, genSvc, tlSvc, sumSvc, memSvc, locSvc, bibleSvc, chapterSvc, outlineSvc, titleSvc, progressHub)

	// ─── Server ────────────────────────────────────────────
	srv := api.NewServer(h, rateLimiter)

	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      srv.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 300 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("http server listening", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// ─── Shutdown ──────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.Info("shutting down", "signal", sig)
	case err := <-errCh:
		slog.Error("http server error, shutting down", "error", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("http shutdown error", "error", err)
	}
	cancel()
	slog.Info("server stopped")
}
