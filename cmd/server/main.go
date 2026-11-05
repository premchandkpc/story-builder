package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/premchand/story-builder/internal/httpapi"
	"github.com/premchand/story-builder/internal/memory"
	storymongo "github.com/premchand/story-builder/internal/mongo"
	"github.com/premchand/story-builder/internal/story"
)

func main() {
	ctx := context.Background()
	var repo story.Repository = memory.NewRepository()
	var disconnect func(context.Context) error
	if uri := os.Getenv("MONGO_URI"); uri != "" {
		client, mongoRepo, err := storymongo.Connect(ctx, uri, env("MONGO_DB", "story_builder"))
		if err != nil {
			slog.Error("MongoDB unavailable", "error", err)
			os.Exit(1)
		}
		repo = mongoRepo
		disconnect = client.Disconnect
		slog.Info("using MongoDB persistence")
	} else {
		slog.Warn("using in-memory persistence; set MONGO_URI to persist graphs")
	}
	if disconnect != nil {
		defer disconnect(context.Background())
	}
	words, err := strconv.Atoi(env("SCENE_WORDS", "1000"))
	if err != nil || words < 100 || words > 5000 {
		slog.Error("SCENE_WORDS must be between 100 and 5000")
		os.Exit(1)
	}
	generator := story.OpenAIGenerator{APIKey: os.Getenv("OPENAI_API_KEY"), Model: env("OPENAI_MODEL", "gpt-4.1"), TargetWords: words}
	if !generator.Ready() {
		slog.Warn("OpenAI generation unavailable; set OPENAI_API_KEY")
	}
	slog.Info("using OpenAI story generation", "model", generator.Model, "minimum_words", words)

	server := &http.Server{Addr: env("PORT", "127.0.0.1:8080"), Handler: httpapi.New(story.NewService(repo), env("AUTH_TOKEN", ""), generator).Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Minute, IdleTimeout: 60 * time.Second}
	go func() {
		slog.Info("server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}
func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
