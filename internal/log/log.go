package log

import (
	"io"
	"log/slog"
	"os"
)

type Config struct {
	Level  string
	JSON   bool
	Writer io.Writer
}

func Init(cfg Config) {
	var lvl slog.Level
	switch cfg.Level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	w := cfg.Writer
	if w == nil {
		w = os.Stderr
	}
	var h slog.Handler
	if cfg.JSON {
		h = slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lvl, AddSource: true})
	} else {
		h = slog.NewTextHandler(w, &slog.HandlerOptions{Level: lvl, AddSource: true})
	}
	slog.SetDefault(slog.New(h))
}

func Err(err error) slog.Attr {
	return slog.String("error", err.Error())
}
