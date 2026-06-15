package config

import (
	"os"
)

type Config struct {
	Port         string
	GrpcPort     string
	DatabaseURL  string
	AnthropicKey string
	OllamaURL    string
	RedisAddr    string
	RedisPass    string
	RedisDB      int
	MongoURI     string
	MongoDB      string
	QdrantAddr   string
	KafkaBrokers string
	KafkaGroupID string
}

func FromEnv() Config {
	return Config{
		Port:         env("PORT", "8080"),
		GrpcPort:     env("GRPC_PORT", "9090"),
		DatabaseURL:  env("DATABASE_URL", "postgres://storybuilder:storybuilder@localhost:5432/storybuilder?sslmode=disable"),
		AnthropicKey: os.Getenv("ANTHROPIC_API_KEY"),
		OllamaURL:    env("OLLAMA_URL", "http://localhost:11434"),
		RedisAddr:    env("REDIS_ADDR", "localhost:6379"),
		RedisPass:    os.Getenv("REDIS_PASSWORD"),
		RedisDB:      0,
		MongoURI:     env("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:      env("MONGO_DB", "storybuilder"),
		QdrantAddr:   env("QDRANT_ADDR", "localhost:6334"),
		KafkaBrokers: os.Getenv("KAFKA_BROKERS"),
		KafkaGroupID: env("KAFKA_GROUP_ID", "storybuilder"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
