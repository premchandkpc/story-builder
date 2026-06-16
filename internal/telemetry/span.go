package telemetry

import (
	"context"
	"log/slog"
	"time"
)

type Span struct {
	name       string
	start      time.Time
	attrs      []slog.Attr
	ended      bool
}

func StartSpan(ctx context.Context, name string, attrs ...slog.Attr) (context.Context, *Span) {
	span := &Span{
		name:  name,
		start: time.Now(),
		attrs: append([]slog.Attr(nil), attrs...),
	}
	ctx = context.WithValue(ctx, spanKey, span)
	return ctx, span
}

func SpanFromContext(ctx context.Context) *Span {
	if s, ok := ctx.Value(spanKey).(*Span); ok {
		return s
	}
	return nil
}

func (s *Span) SetAttrs(attrs ...slog.Attr) {
	s.attrs = append(s.attrs, attrs...)
}

func (s *Span) RecordError(err error) {
	if err != nil {
		s.attrs = append(s.attrs, slog.String("error", err.Error()))
	}
}

func (s *Span) End(opts ...SpanEndOption) {
	if s.ended {
		return
	}
	s.ended = true
	cfg := spanEndConfig{level: slog.LevelInfo}
	for _, o := range opts {
		o(&cfg)
	}
	elapsed := time.Since(s.start)
	attrs := make([]slog.Attr, 0, len(s.attrs)+2)
	attrs = append(attrs, s.attrs...)
	attrs = append(attrs, slog.String("span", s.name))
	attrs = append(attrs, slog.Duration("elapsed", elapsed))
	slog.LogAttrs(context.Background(), cfg.level, "span:"+s.name, attrs...)
}

type spanEndConfig struct {
	level slog.Level
}

type SpanEndOption func(*spanEndConfig)

func WithLevel(lvl slog.Level) SpanEndOption {
	return func(c *spanEndConfig) {
		c.level = lvl
	}
}

type ctxKey struct{}

var spanKey ctxKey = struct{}{}
