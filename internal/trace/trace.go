package trace

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type spanKey struct{}

type Span struct {
	TraceID    string
	SpanID     string
	ParentID   string
	Operation  string
	Start      time.Time
	End        time.Time
	Attributes map[string]any
	Error      string
}

type spanCarrier struct {
	current *Span
}

func NewContext(ctx context.Context, operation string) context.Context {
	traceID := uuid.New().String()
	span := &Span{
		TraceID:    traceID,
		SpanID:     uuid.New().String(),
		Operation:  operation,
		Start:      time.Now(),
		Attributes: make(map[string]any),
	}
	return context.WithValue(ctx, spanKey{}, &spanCarrier{current: span})
}

func StartSpan(ctx context.Context, operation string) (context.Context, *Span) {
	carrier, ok := ctx.Value(spanKey{}).(*spanCarrier)
	if !ok {
		return NewContext(ctx, operation), nil
	}
	parent := carrier.current
	span := &Span{
		TraceID:    parent.TraceID,
		SpanID:     uuid.New().String(),
		ParentID:   parent.SpanID,
		Operation:  operation,
		Start:      time.Now(),
		Attributes: make(map[string]any),
	}
	childCtx := context.WithValue(ctx, spanKey{}, &spanCarrier{current: span})
	return childCtx, span
}

func SetAttribute(span *Span, key string, value any) {
	if span != nil && span.Attributes != nil {
		span.Attributes[key] = value
	}
}

func SetError(span *Span, err error) {
	if span != nil && err != nil {
		span.Error = err.Error()
	}
}

func End(span *Span) {
	if span == nil {
		return
	}
	span.End = time.Now()
	dur := span.End.Sub(span.Start)
	attrs := []slog.Attr{
		slog.String("trace_id", span.TraceID),
		slog.String("span_id", span.SpanID),
		slog.String("operation", span.Operation),
		slog.Duration("duration", dur),
	}
	if span.ParentID != "" {
		attrs = append(attrs, slog.String("parent_id", span.ParentID))
	}
	if span.Error != "" {
		attrs = append(attrs, slog.String("error", span.Error))
	}
	if len(span.Attributes) > 0 {
		attrs = append(attrs, slog.Any("attrs", span.Attributes))
	}
	if span.Error != "" {
		slog.LogAttrs(context.Background(), slog.LevelError, "span", attrs...)
	} else {
		slog.LogAttrs(context.Background(), slog.LevelDebug, "span", attrs...)
	}
}
