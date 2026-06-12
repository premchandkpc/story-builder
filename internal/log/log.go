package log

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"
	"time"
)

type Config struct {
	Level  string // debug, info, warn, error
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

func Duration(d time.Duration) slog.Attr {
	return slog.Duration("duration", d)
}

func Caller(skip int) slog.Attr {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return slog.Attr{}
	}
	return slog.String("caller", file+":"+itoa(line))
}

var ctxKey = struct{}{}

func WithContext(ctx context.Context, attrs ...slog.Attr) context.Context {
	return context.WithValue(ctx, ctxKey, attrs)
}

func Ctx(ctx context.Context) *slog.Logger {
	if attrs, ok := ctx.Value(ctxKey).([]slog.Attr); ok {
		args := make([]any, len(attrs))
		for i, a := range attrs {
			args[i] = a
		}
		return slog.With(args...)
	}
	return slog.Default()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
