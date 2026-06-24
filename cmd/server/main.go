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
	"github.com/premchand/story-builder/internal/config"
	"github.com/premchand/story-builder/internal/log"
	mgorepo "github.com/premchand/story-builder/internal/repository/mongo"
	"github.com/premchand/story-builder/internal/trace"
)

func main() {
	cfg := config.FromEnv()
	log.Init(log.Config{Level: cfg.LogLevel})
	slog.Info("starting story-builder", "port", cfg.Port)

	otelShutdown := trace.InitFromEnv(context.Background())
	defer otelShutdown(context.Background())

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

	deps := initAll(cfg, db)
	deps.genJobWorker.Start()
	defer deps.genJobWorker.Stop()

	h := api.NewHandlers(
		deps.storySvc, deps.sceneSvc, deps.edgeSvc, deps.charSvc,
		deps.genSvc, deps.genSvc,
		deps.tlSvc, deps.sumSvc, deps.memSvc, deps.locSvc,
		deps.bibleSvc, deps.chapterSvc,
		deps.outlineSvc, deps.titleSvc,
		deps.metricsSvc, deps.criticSvc, deps.agentCfgSvc,
		deps.progressHub, deps.eventBus,
		deps.agentSvc, deps.agentSvc,
		deps.runSvc, deps.narrativeSvc,
		deps.plannerSvc, deps.diffSvc,
	)

	srv := api.NewServer(h, deps.rateLimiter, func(ctx context.Context) error {
		return db.Client().Ping(ctx, nil)
	})
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
	if err := db.Client().Disconnect(shutdownCtx); err != nil {
		slog.Error("mongo disconnect error", "error", err)
	}
	slog.Info("server stopped")
}
