package telemetry

import (
	"log/slog"
	"os"
)

type Config struct {
	Enabled bool
}

func InitFromEnv() Config {
	cfg := Config{Enabled: true}
	if os.Getenv("TELEMETRY_DISABLED") == "true" {
		cfg.Enabled = false
		slog.Info("telemetry disabled via TELEMETRY_DISABLED=true")
	}
	if cfg.Enabled {
		slog.Info("telemetry enabled")
	}
	return cfg
}
