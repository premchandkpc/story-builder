package trace

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Span is an alias for go.opentelemetry.io/otel/trace.Span so callers use trace.Span.
type Span = trace.Span

func tracer() trace.Tracer {
	return otel.Tracer("story-builder")
}

func StartSpan(ctx context.Context, operation string) (context.Context, Span) {
	return tracer().Start(ctx, operation)
}

func SetAttribute(span Span, key string, value any) {
	if span == nil {
		return
	}
	switch v := value.(type) {
	case string:
		span.SetAttributes(attribute.String(key, v))
	case int:
		span.SetAttributes(attribute.Int(key, v))
	case bool:
		span.SetAttributes(attribute.Bool(key, v))
	case float64:
		span.SetAttributes(attribute.Float64(key, v))
	case []string:
		span.SetAttributes(attribute.StringSlice(key, v))
	default:
		span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
	}
}

func SetError(span Span, err error) {
	if span == nil || err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func End(span Span) {
	if span == nil {
		return
	}
	span.End()
}

func NewContext(ctx context.Context, operation string) context.Context {
	ctx, _ = tracer().Start(ctx, operation)
	return ctx
}

func LogSpanError(span Span, msg string, attrs ...any) {
	if span == nil {
		slog.Error(msg, attrs...)
		return
	}
	span.SetStatus(codes.Error, msg)
	span.RecordError(fmt.Errorf("%s", msg))
	slog.Error(msg, attrs...)
}
