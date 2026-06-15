package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/premchand/story-builder/internal/api"
	"github.com/premchand/story-builder/internal/cache"
	"github.com/premchand/story-builder/internal/config"
	"github.com/premchand/story-builder/internal/db"
	"github.com/premchand/story-builder/internal/graph"
	grpcserver "github.com/premchand/story-builder/internal/grpc/server"
	"github.com/premchand/story-builder/internal/llm"
	applog "github.com/premchand/story-builder/internal/log"
	"github.com/premchand/story-builder/internal/migrate"
	"github.com/premchand/story-builder/internal/river"
	cachesvc "github.com/premchand/story-builder/internal/service/cache"
	blueprintsvc "github.com/premchand/story-builder/internal/service/blueprint"
	canonsvc "github.com/premchand/story-builder/internal/service/canon"
	chaptersvc "github.com/premchand/story-builder/internal/service/chapter"
	edgesvc "github.com/premchand/story-builder/internal/service/edge"
	gensvc "github.com/premchand/story-builder/internal/service/generation"
	nodesvc "github.com/premchand/story-builder/internal/service/node"
	scenesvc "github.com/premchand/story-builder/internal/service/scene"
	storysvc "github.com/premchand/story-builder/internal/service/story"
	summarysvc "github.com/premchand/story-builder/internal/service/summary"
	timelinesvc "github.com/premchand/story-builder/internal/service/timeline"
	riv "github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

func main() {
	applog.Init(applog.Config{
		Level: os.Getenv("LOG_LEVEL"),
		JSON:  os.Getenv("LOG_FORMAT") == "json",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.FromEnv()

	var pool *pgxpool.Pool
	dbOk := false

	var redisCache *cachesvc.Cache
	if rc, err := cachesvc.New(cfg.RedisAddr, cfg.RedisPass, cfg.RedisDB); err == nil {
		redisCache = rc
		slog.Info("connected to redis", "addr", cfg.RedisAddr)
	} else {
		slog.Warn("redis not available (running without cache)", "error", err)
	}
	if p, err := pgxpool.New(ctx, cfg.DatabaseURL); err == nil {
		if err := p.Ping(ctx); err == nil {
			pool = p
			dbOk = true
		} else {
			p.Close()
			slog.Warn("db ping failed", "error", err)
		}
	} else {
		slog.Warn("db connect failed", "error", err)
	}

	if dbOk {
		slog.Info("connected to postgres")
		runner := migrate.New(pool, "migrations")
		if err := runner.Run(ctx); err != nil {
			slog.Error("migration failed", "error", err)
		}
	} else {
		slog.Info("no database, running with in-memory stores")
	}

	var charHandler *api.CharacterHandler
	var actorHandler *api.ActorHandler
	var traitHandler *api.CharacterTraitHandler
	var castingHandler *api.CastingHandler
	var locHandler *api.LocationHandler
	var loreHandler *api.LoreHandler
	var storyHandler *api.StoryHandler
	var chapterHandler *api.ChapterHandler
	var nodeHandler *api.NodeHandler
	var genHandler *api.GenerationHandler
	var sceneHandler *api.SceneHandler
	var summaryHandler *api.SummaryHandler
	var storyGenHandler *api.StoryGeneratorHandler
	var titleHandler *api.TitleHandler
	blueprintService := blueprintsvc.NewMemoryService()
	timelineService := timelinesvc.NewMemoryService()

	llmClient := createLLMClient(cfg)

	if dbOk {
		q := db.New(pool)

		if redisCache != nil {
			llmClient = redisCache.WrapLLMClient(llmClient)
		}

		proseSvc := llm.NewProseService(llmClient)
		extractSvc := llm.NewExtractionService(llmClient)
		summarySvc := llm.NewSummaryService(llmClient)
		mergeSvc := llm.NewMergeService(llmClient)
		validateSvc := llm.NewValidationService(llmClient)
		outlineSvc := llm.NewOutlineService(llmClient)

		deps := &river.Dependencies{
			Prose:    proseSvc,
			Extract:  extractSvc,
			Summary:  summarySvc,
			Merge:    mergeSvc,
			Validate: validateSvc,
			Outline:  outlineSvc,
			Queries:  q,
		}
		workers := river.Workers(deps)

		migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
		if err != nil {
			slog.Error("river migrator init", "error", err)
		} else if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
			slog.Error("river migrator run", "error", err)
		}

		rcfg := &riv.Config{
			Workers: workers,
			Queues: map[string]riv.QueueConfig{
				river.QueueDefault:  {MaxWorkers: 1},
				river.QueueGenerate: {MaxWorkers: 2},
				river.QueueExtract:  {MaxWorkers: 4},
				river.QueueMerge:    {MaxWorkers: 2},
				river.QueueValidate: {MaxWorkers: 1},
			},
		}
		rivClient, err := riv.NewClient(riverpgxv5.New(pool), rcfg)
		if err != nil {
			slog.Error("river client init", "error", err)
		} else {
			if err := rivClient.Start(ctx); err != nil {
				slog.Error("river start", "error", err)
			} else {
				defer func() {
					if stopErr := rivClient.Stop(ctx); stopErr != nil {
						slog.Error("river stop", "error", stopErr)
					}
				}()
			}
		}

		storySvc := storysvc.NewDBService(q)
		edgeSvc := edgesvc.NewDBService(q)
		nodeSvc := nodesvc.NewDBService(q)

		charHandler = &api.CharacterHandler{Service: canonsvc.NewDBCharacterService(q)}
		actorHandler = &api.ActorHandler{Service: canonsvc.NewDBActorService(q)}
		traitHandler = &api.CharacterTraitHandler{Service: canonsvc.NewDBTraitService(q)}
		castingHandler = &api.CastingHandler{Service: canonsvc.NewDBCastingService(q)}
		locHandler = &api.LocationHandler{Service: canonsvc.NewDBLocationService(q)}
		loreHandler = &api.LoreHandler{Service: canonsvc.NewDBLoreService(q)}
		chapterHandler = &api.ChapterHandler{Service: chaptersvc.NewDBService(q)}
		storyHandler = &api.StoryHandler{
			StorySvc:         storySvc,
			EdgeSvc:          edgeSvc,
			NodeSvc:          nodeSvc,
			BlueprintService: blueprintService,
			TimelineService:  timelineService,
		}
		nodeHandler = &api.NodeHandler{Service: nodeSvc}

		var contextCache *cache.ContextCache
		if redisCache != nil {
			contextCache = redisCache.ContextCache
		}
		genHandler = &api.GenerationHandler{Service: gensvc.NewDBGenerationServiceWithCache(q, rivClient, contextCache)}
		sceneHandler = &api.SceneHandler{SceneService: scenesvc.NewDBService(q)}
		summaryHandler = &api.SummaryHandler{Service: summarysvc.NewDBService(q)}
		storyGenHandler = &api.StoryGeneratorHandler{Service: gensvc.NewDBStoryGeneratorService(q, rivClient)}
		titleHandler = &api.TitleHandler{Service: llm.NewTitleService(llmClient)}
	} else {
		gs := graph.NewMemoryStore()
		storySvc := storysvc.NewMemoryService(gs)
		nodeSvc := nodesvc.NewMemoryService(gs)
		edgeSvc := edgesvc.NewMemoryService(gs)

		charHandler = &api.CharacterHandler{Service: canonsvc.NewMemoryCharacterService()}
		actorHandler = &api.ActorHandler{Service: canonsvc.NewMemoryActorService()}
		traitHandler = &api.CharacterTraitHandler{Service: canonsvc.NewMemoryTraitService()}
		castingHandler = &api.CastingHandler{Service: canonsvc.NewMemoryCastingService()}
		locHandler = &api.LocationHandler{Service: canonsvc.NewMemoryLocationService()}
		loreHandler = &api.LoreHandler{Service: canonsvc.NewMemoryLoreService()}
		chapterHandler = &api.ChapterHandler{Service: chaptersvc.NewMemoryService(gs)}
		storyHandler = &api.StoryHandler{
			StorySvc:         storySvc,
			EdgeSvc:          edgeSvc,
			NodeSvc:          nodeSvc,
			BlueprintService: blueprintService,
			TimelineService:  timelineService,
		}
		nodeHandler = &api.NodeHandler{Service: nodeSvc}
		genHandler = &api.GenerationHandler{Service: gensvc.NewMemoryGenerationService()}
		sceneHandler = &api.SceneHandler{SceneService: scenesvc.NewMemoryService()}
		summaryHandler = &api.SummaryHandler{Service: summarysvc.NewMemoryService()}
		storyGenHandler = &api.StoryGeneratorHandler{Service: gensvc.NewMemoryStoryGeneratorService()}
		titleHandler = &api.TitleHandler{Service: llm.NewTitleService(llmClient)}
	}

	var rateLimiter *cache.SlidingWindowRateLimiter
	if redisCache != nil {
		rateLimiter = redisCache.RateLimiter
	}
	srv := api.NewServer(charHandler, actorHandler, traitHandler, castingHandler, locHandler, loreHandler, storyHandler, chapterHandler, nodeHandler, genHandler, sceneHandler, summaryHandler, storyGenHandler, titleHandler, rateLimiter)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      srv,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("http server starting", "port", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	// gRPC server
	grpcSrv := grpcserver.New(
		charHandler.Service,
		actorHandler.Service,
		traitHandler.Service,
		castingHandler.Service,
		locHandler.Service,
		loreHandler.Service,
		storyHandler.StorySvc,
		nodeHandler.Service,
		storyHandler.EdgeSvc,
		genHandler.Service,
		sceneHandler.SceneService,
		summaryHandler.Service,
		storyGenWrapper{svc: storyGenHandler.Service},
		cfg.GrpcPort,
	)

	go func() {
		slog.Info("gRPC server starting", "port", cfg.GrpcPort)
		if err := grpcSrv.Start(ctx); err != nil {
			slog.Error("gRPC server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
		os.Exit(1)
	}
}

type storyGenWrapper struct {
	svc gensvc.StoryGeneratorService
}

func (w storyGenWrapper) GenerateStory(ctx context.Context, synopsis string) (string, string, error) {
	r, err := w.svc.GenerateStory(ctx, synopsis)
	if err != nil {
		return "", "", err
	}
	return r.StoryID, r.Status, nil
}

func createLLMClient(cfg config.Config) llm.LLMClient {
	var anthropic llm.LLMClient
	if cfg.AnthropicKey != "" {
		anthropic = llm.NewAnthropicClient(cfg.AnthropicKey)
		slog.Info("using anthropic client")
	} else {
		slog.Info("anthropic key not set, falling back to ollama")
	}

	ollamaURL := cfg.OllamaURL
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	ollama := llm.NewOllamaClient(ollamaURL)
	slog.Info("using ollama client", "url", ollamaURL)
	return llm.NewRouter(anthropic, ollama)
}
