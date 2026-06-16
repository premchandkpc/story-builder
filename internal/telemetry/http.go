package telemetry

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func HTTPTracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		attrs := []slog.Attr{
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		}

		_, span := StartSpan(r.Context(), "http."+r.Method+"."+cleanPath(r.URL.Path),
			attrs...,
		)

		defer func() {
			latency := time.Since(start)
			status := ww.Status()
			spanAttrs := append(attrs,
				slog.Int("status", status),
				slog.Int("bytes", ww.BytesWritten()),
				slog.Duration("latency", latency),
			)
			HTTPLatency.Record(latency, slog.String("method", r.Method), slog.String("path", r.URL.Path), slog.Int("status", status))
			HTTPRequests.Add(1, slog.String("method", r.Method), slog.String("path", r.URL.Path), slog.Int("status", status))

			if status >= 500 {
				span.RecordError(http.ErrAbortHandler)
				span.End(WithLevel(slog.LevelError))
				HTTPErrors.Inc(slog.String("method", r.Method), slog.String("path", r.URL.Path))
			} else if status >= 400 {
				span.End(WithLevel(slog.LevelWarn))
			} else {
				span.End()
			}
			_ = spanAttrs
		}()

		next.ServeHTTP(ww, r)
	})
}

func cleanPath(path string) string {
	b := make([]byte, 0, len(path))
	for i := 0; i < len(path); i++ {
		if path[i] >= 'a' && path[i] <= 'z' || path[i] >= 'A' && path[i] <= 'Z' || path[i] >= '0' && path[i] <= '9' || path[i] == '/' || path[i] == '-' || path[i] == '_' {
			b = append(b, path[i])
		} else if path[i] == '{' || path[i] == '}' {
			b = append(b, path[i])
		}
	}
	return string(b)
}
