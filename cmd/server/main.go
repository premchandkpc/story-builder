package main

import (
	"context"
	"fmt"
	"log"
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
	"github.com/premchand/story-builder/internal/migrate"
	"github.com/premchand/story-builder/internal/river"
	riv "github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.FromEnv()

	var pool *pgxpool.Pool
	dbOk := false

	var contextCache *cache.ContextCache
	var promptCache *cache.PromptCache
	var rateLimiter *cache.SlidingWindowRateLimiter

	if rc, err := cache.NewGoRedisClient(cfg.RedisAddr, cfg.RedisPass, cfg.RedisDB); err == nil {
		contextCache = cache.NewContextCache(rc)
		promptCache = cache.NewPromptCache(rc)
		rateLimiter = cache.NewSlidingWindowRateLimiter(rc, cache.DefaultRateLimits)
		log.Printf("connected to redis at %s", cfg.RedisAddr)
	} else {
		log.Printf("redis not available (running without cache): %v", err)
	}
	if p, err := pgxpool.New(ctx, cfg.DatabaseURL); err == nil {
		if err := p.Ping(ctx); err == nil {
			pool = p
			dbOk = true
		} else {
			p.Close()
			log.Printf("db ping failed: %v", err)
		}
	} else {
		log.Printf("db connect failed: %v", err)
	}

	if dbOk {
		log.Println("connected to postgres")
		runner := migrate.New(pool, "migrations")
		if err := runner.Run(ctx); err != nil {
			log.Printf("migrate: %v", err)
		}
	} else {
		log.Println("no database, running with in-memory stores")
	}

	var charHandler *api.CharacterHandler
	var actorHandler *api.ActorHandler
	var traitHandler *api.CharacterTraitHandler
	var castingHandler *api.CastingHandler
	var locHandler *api.LocationHandler
	var loreHandler *api.LoreHandler
	var storyHandler *api.StoryHandler
	var nodeHandler *api.NodeHandler
	var genHandler *api.GenerationHandler
	var sceneHandler *api.SceneHandler
	var summaryHandler *api.SummaryHandler
	var storyGenHandler *api.StoryGeneratorHandler
	blueprintService := api.NewInMemoryBlueprintService()
	timelineService := api.NewInMemoryTimelineService()

	if dbOk {
		q := db.New(pool)

		llmClient := createLLMClient(cfg)

		if promptCache != nil {
			llmClient = cache.NewCachedLLMClient(llmClient, promptCache)
		}
		if rateLimiter != nil {
			llmClient = cache.NewRateLimitedLLMClient(llmClient, rateLimiter, "llm:anthropic")
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
			log.Printf("river migrator: %v", err)
		} else if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
			log.Printf("river migrate: %v", err)
		}

		rcfg := &riv.Config{
			Workers: workers,
			Queues: map[string]riv.QueueConfig{
				river.QueueGenerate: {MaxWorkers: 2},
				river.QueueExtract:  {MaxWorkers: 4},
				river.QueueMerge:    {MaxWorkers: 2},
				river.QueueValidate: {MaxWorkers: 1},
			},
		}
		rivClient, err := riv.NewClient(riverpgxv5.New(pool), rcfg)
		if err != nil {
			log.Printf("river init: %v", err)
		} else {
			if err := rivClient.Start(ctx); err != nil {
				log.Printf("river start: %v", err)
			} else {
				defer func() {
					if stopErr := rivClient.Stop(ctx); stopErr != nil {
						log.Printf("river stop: %v", stopErr)
					}
				}()
			}
		}

		charHandler = &api.CharacterHandler{Service: api.NewDBCharService(q)}
		actorHandler = &api.ActorHandler{Service: api.NewDBActorService(q)}
		traitHandler = &api.CharacterTraitHandler{Service: api.NewDBCharacterTraitService(q)}
		castingHandler = &api.CastingHandler{Service: api.NewDBCastingService(q)}
		locHandler = &api.LocationHandler{Service: api.NewDBLocService(q)}
		loreHandler = &api.LoreHandler{Service: api.NewDBLoreService(q)}
		storyHandler = &api.StoryHandler{Service: api.NewDBGraphStoryService(q), BlueprintService: blueprintService, TimelineService: timelineService}
		nodeHandler = &api.NodeHandler{Service: api.NewDBGraphNodeService(q)}
		genHandler = &api.GenerationHandler{Service: api.NewDBGenerationServiceWithCache(q, rivClient, contextCache)}
		sceneHandler = &api.SceneHandler{SceneService: api.NewDBSceneService(q)}
		summaryHandler = &api.SummaryHandler{Service: api.NewDBSummaryService(q)}
		storyGenHandler = &api.StoryGeneratorHandler{Service: api.NewDBStoryGeneratorService(q, rivClient)}
	} else {
		gs := graph.NewMemoryStore()
		charHandler = &api.CharacterHandler{Service: api.NewCharService()}
		actorHandler = &api.ActorHandler{Service: api.NewActorService()}
		traitHandler = &api.CharacterTraitHandler{Service: api.NewCharacterTraitService()}
		castingHandler = &api.CastingHandler{Service: api.NewCastingService()}
		locHandler = &api.LocationHandler{Service: api.NewLocService()}
		loreHandler = &api.LoreHandler{Service: api.NewLoreService()}
		storyHandler = &api.StoryHandler{Service: api.NewGraphStoryService(gs), BlueprintService: blueprintService, TimelineService: timelineService}
		nodeHandler = &api.NodeHandler{Service: api.NewGraphNodeService(gs)}
		genHandler = &api.GenerationHandler{Service: api.NewGenerationService()}
		sceneHandler = &api.SceneHandler{SceneService: api.NewMemorySceneService()}
		summaryHandler = &api.SummaryHandler{Service: api.NewMemorySummaryService()}
		storyGenHandler = &api.StoryGeneratorHandler{Service: api.NewMemoryStoryGeneratorService()}
	}

	srv := api.NewServer(charHandler, actorHandler, traitHandler, castingHandler, locHandler, loreHandler, storyHandler, nodeHandler, genHandler, sceneHandler, summaryHandler, storyGenHandler, rateLimiter)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      srv,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("http server on :%s", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
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
		storyHandler.Service,
		nodeHandler.Service,
		genHandler.Service,
		sceneHandler.SceneService,
		summaryHandler.Service,
		storyGenWrapper{svc: storyGenHandler.Service},
		cfg.GrpcPort,
	)

	go func() {
		log.Printf("gRPC server on :%s", cfg.GrpcPort)
		if err := grpcSrv.Start(ctx); err != nil {
			log.Fatalf("gRPC: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		shutdownCancel()
		log.Fatalf("shutdown: %v", err)
	}
	shutdownCancel()
}

type storyGenWrapper struct {
	svc api.StoryGeneratorService
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
		log.Println("using anthropic client")
	} else {
		log.Println("anthropic key not set, falling back to ollama")
	}

	ollamaURL := cfg.OllamaURL
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	ollama := llm.NewOllamaClient(ollamaURL)
	log.Printf("using ollama client (%s)", ollamaURL)
	return llm.NewRouter(anthropic, ollama)
}
