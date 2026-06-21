package config

import "os"

type Config struct {
	Port         string
	MongoURI     string
	MongoDB      string
	RedisAddr    string
	RedisPass    string
	AnthropicKey string
	OpenCodeURL   string
	OpenCodeModel string
	HeadroomURL  string
	HeadroomKey  string
	LogLevel     string
}

func FromEnv() Config {
	return Config{
		Port:         env("PORT", "8080"),
		MongoURI:     env("MONGO_URI", "mongodb://storybuilder:storybuilder@localhost:27017"),
		MongoDB:      env("MONGO_DB", "storybuilder"),
		RedisAddr:    env("REDIS_ADDR", "localhost:6379"),
		RedisPass:    env("REDIS_PASSWORD", ""),
		AnthropicKey: env("ANTHROPIC_API_KEY", ""),
		OpenCodeURL:   env("OPENCODE_URL", "http://localhost:11434"),
		OpenCodeModel: env("OPENCODE_MODEL", "llama3.2:3b"),
		HeadroomURL:  env("HEADROOM_BASE_URL", ""),
		HeadroomKey:  env("HEADROOM_API_KEY", ""),
		LogLevel:     env("LOG_LEVEL", "info"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
