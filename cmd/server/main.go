package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
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
	address := env("PORT", "127.0.0.1:8080")
	authToken := env("AUTH_TOKEN", "")
	if err := validateListener(address, authToken); err != nil {
		slog.Error("unsafe server configuration", "error", err)
		os.Exit(1)
	}
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
	var generator story.ProseGenerator
	switch env("GENERATION_PROVIDER", "openai") {
	case "template":
		generator = story.TemplateGenerator{}
	case "openai":
		provider := story.OpenAIGenerator{APIKey: os.Getenv("OPENAI_API_KEY"), Model: env("OPENAI_MODEL", "gpt-4.1"), TargetWords: words}
		if !provider.Ready() {
			slog.Warn("OpenAI generation unavailable; set OPENAI_API_KEY")
		}
		generator = provider
		slog.Info("using OpenAI story generation", "model", provider.Model, "minimum_words", words)
	default:
		slog.Error("GENERATION_PROVIDER must be openai or template")
		os.Exit(1)
	}
	server := &http.Server{Addr: address, Handler: httpapi.New(story.NewService(repo), authToken, generator).Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 5 * time.Minute, IdleTimeout: 60 * time.Second}
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

// Development without a token is allowed only on an explicit loopback listener.
func validateListener(address, token string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("PORT must be host:port: %w", err)
	}
	ip := net.ParseIP(host)
	if token == "" && host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return fmt.Errorf("AUTH_TOKEN is required for a non-loopback listener")
	}
	return nil
}
