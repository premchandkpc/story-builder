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
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/log"
	"github.com/premchand/story-builder/internal/prompt"
	mgorepo "github.com/premchand/story-builder/internal/repository/mongo"
	"github.com/premchand/story-builder/internal/service"
)

func main() {
	cfg := config.FromEnv()
	log.Init(log.Config{Level: "info"})
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

	// ─── LLM Clients ───────────────────────────────────────
	var anthropic llm.LLMClient
	if cfg.AnthropicKey != "" {
		anthropic = llm.NewAnthropicClient(cfg.AnthropicKey)
		slog.Info("anthropic client created")
	} else {
		slog.Warn("no ANTHROPIC_API_KEY set, using Ollama for all tiers")
		anthropic = llm.NewOllamaClient(cfg.OllamaURL)
	}
	ollama := llm.NewOllamaClient(cfg.OllamaURL)
	router := llm.NewRouter(anthropic, ollama)

	// ─── Prompt Compiler ───────────────────────────────────
	promptStore := prompt.NewMemoryStore()
	promptCompiler := prompt.NewCompilerService(promptStore)

	// ─── LLM Services ──────────────────────────────────────
	proseSvc := llm.NewProseService(router, promptCompiler)
	extractSvc := llm.NewExtractionService(router, promptCompiler)
	summarySvc := llm.NewSummaryService(router, promptCompiler)
	mergeSvc := llm.NewMergeService(router, promptCompiler)
	validateSvc := llm.NewValidationService(router, promptCompiler)
	outlineSvc := llm.NewOutlineService(router, promptCompiler)
	titleSvc := llm.NewTitleService(router)

	_ = mergeSvc
	_ = outlineSvc
	_ = titleSvc

	// ─── Cache ─────────────────────────────────────────────
	var rateLimiter *cache.SlidingWindowRateLimiter

	if cfg.RedisAddr != "" {
		redisClient, err := cache.NewGoRedisClient(cfg.RedisAddr, cfg.RedisPass, 0)
		if err != nil {
			slog.Warn("redis unavailable, running without cache", "error", err)
		} else {
			rateLimiter = cache.NewSlidingWindowRateLimiter(redisClient, []cache.RateLimitConfig{
				{Key: "api", Limit: 10, Window: time.Minute},
			})
			slog.Info("redis cache enabled")
		}
	} else {
		slog.Info("no REDIS_ADDR set, running without cache")
	}

	// ─── Services ──────────────────────────────────────────
	storySvc := service.NewStoryService(storyRepo)
	sceneSvc := service.NewSceneService(sceneRepo, edgeRepo)
	edgeSvc := service.NewEdgeService(edgeRepo)
	charSvc := service.NewCharacterService(charRepo, stateRepo)
	genSvc := service.NewGenerationService(genRepo, sceneRepo, charRepo, stateRepo, memRepo, tlRepo, sumRepo, proseSvc, extractSvc, summarySvc, validateSvc)
	tlSvc := service.NewTimelineService(tlRepo)
	sumSvc := service.NewSummaryService(sumRepo)
	memSvc := service.NewMemoryService(memRepo)

	// ─── Handlers ──────────────────────────────────────────
	h := api.NewHandlers(storySvc, sceneSvc, edgeSvc, charSvc, genSvc, tlSvc, sumSvc, memSvc)

	// ─── Server ────────────────────────────────────────────
	srv := api.NewServer(h, rateLimiter)

	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      srv.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("http server listening", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	// ─── Shutdown ──────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("shutting down", "signal", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("http shutdown error", "error", err)
	}
	slog.Info("server stopped")
}
