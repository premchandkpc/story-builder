package log

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

type traceLogHandler struct {
	inner slog.Handler
}

func TraceLogHandler(inner slog.Handler) slog.Handler {
	return &traceLogHandler{inner: inner}
}

func (h *traceLogHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return h.inner.Enabled(ctx, lvl)
}

func (h *traceLogHandler) Handle(ctx context.Context, r slog.Record) error {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		sc := span.SpanContext()
		if sc.HasTraceID() {
			r.AddAttrs(slog.String("trace_id", sc.TraceID().String()))
		}
		if sc.HasSpanID() {
			r.AddAttrs(slog.String("span_id", sc.SpanID().String()))
		}
	}
	return h.inner.Handle(ctx, r)
}

func (h *traceLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceLogHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *traceLogHandler) WithGroup(name string) slog.Handler {
	return &traceLogHandler{inner: h.inner.WithGroup(name)}
}
