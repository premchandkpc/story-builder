package storage

import (
	"context"
	"log/slog"

	mongoadapter "github.com/premchand/story-builder/internal/adapter/mongo"
	qdrantadapter "github.com/premchand/story-builder/internal/adapter/qdrant"
	redisadapter "github.com/premchand/story-builder/internal/adapter/redis"
	"github.com/premchand/story-builder/internal/config"
	kafkabus "github.com/premchand/story-builder/internal/adapter/kafka"
)

type Adapters struct {
	Mongo  *mongoadapter.Client
	Qdrant *qdrantadapter.Client
	Redis  *redisadapter.Client
	EventBus *kafkabus.Bus
}

func Init(ctx context.Context, cfg *config.Config) *Adapters {
	a := &Adapters{}

	if m, err := mongoadapter.NewClient(ctx, cfg.MongoURI, cfg.MongoDB); err == nil {
		a.Mongo = m
		slog.Info("connected to mongodb", "db", cfg.MongoDB)
	} else {
		slog.Warn("mongodb not available", "error", err)
	}

	if q, err := qdrantadapter.NewClient(ctx, cfg.QdrantAddr); err == nil {
		a.Qdrant = q
		slog.Info("connected to qdrant", "addr", cfg.QdrantAddr)
	} else {
		slog.Warn("qdrant not available", "error", err)
	}

	if r, err := redisadapter.NewClient(ctx, cfg.RedisAddr); err == nil {
		a.Redis = r
		slog.Info("connected to redis", "addr", cfg.RedisAddr)
	} else {
		slog.Warn("redis not available", "error", err)
	}

	if cfg.KafkaBrokers != "" {
		brokers := []string{cfg.KafkaBrokers}
		a.EventBus = kafkabus.NewBus(brokers, "storybuilder.events", cfg.KafkaGroupID)
		a.EventBus.Start()
		slog.Info("connected to kafka", "brokers", cfg.KafkaBrokers)
	} else {
		slog.Warn("kafka not configured, using in-memory event bus")
	}

	return a
}
