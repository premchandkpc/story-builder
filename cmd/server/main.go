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
	"github.com/premchand/story-builder/internal/db"
	"github.com/premchand/story-builder/internal/graph"
	"github.com/premchand/story-builder/internal/migrate"
	"github.com/premchand/story-builder/internal/river"
	riv "github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := configFromEnv()

	var pool *pgxpool.Pool
	dbOk := false
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
	var locHandler *api.LocationHandler
	var loreHandler *api.LoreHandler
	var storyHandler *api.StoryHandler
	var nodeHandler *api.NodeHandler
	var genHandler *api.GenerationHandler

	if dbOk {
		q := db.New(pool)
		charHandler = &api.CharacterHandler{Service: api.NewDBCharService(q)}
		locHandler = &api.LocationHandler{Service: api.NewDBLocService(q)}
		loreHandler = &api.LoreHandler{Service: api.NewDBLoreService(q)}
		storyHandler = &api.StoryHandler{Service: api.NewDBGraphStoryService(q)}
		nodeHandler = &api.NodeHandler{Service: api.NewDBGraphNodeService(q)}
		genHandler = &api.GenerationHandler{Service: api.NewDBGenerationService(q)}

		workers := river.Workers()
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
		client, err := riv.NewClient(riverpgxv5.New(pool), rcfg)
		if err != nil {
			log.Printf("river init: %v", err)
		} else {
			if err := client.Start(ctx); err != nil {
				log.Printf("river start: %v", err)
			} else {
				defer client.Stop(ctx)
			}
		}
	} else {
		gs := graph.NewMemoryStore()
		charHandler = &api.CharacterHandler{Service: api.NewCharService()}
		locHandler = &api.LocationHandler{Service: api.NewLocService()}
		loreHandler = &api.LoreHandler{Service: api.NewLoreService()}
		storyHandler = &api.StoryHandler{Service: api.NewGraphStoryService(gs)}
		nodeHandler = &api.NodeHandler{Service: api.NewGraphNodeService(gs)}
		genHandler = &api.GenerationHandler{Service: api.NewGenerationService()}
	}

	srv := api.NewServer(charHandler, locHandler, loreHandler, storyHandler, nodeHandler, genHandler)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      srv,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("server on :%s", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
}

type config struct {
	Port         string
	DatabaseURL  string
	AnthropicKey string
	OllamaURL    string
}

func configFromEnv() config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://storybuilder:storybuilder@localhost:5432/storybuilder?sslmode=disable"
	}
	return config{
		Port:         port,
		DatabaseURL:  dbURL,
		AnthropicKey: os.Getenv("ANTHROPIC_API_KEY"),
		OllamaURL:    os.Getenv("OLLAMA_URL"),
	}
}
