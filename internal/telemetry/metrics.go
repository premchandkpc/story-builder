package telemetry

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type TraceContext struct {
	TraceID     string    `json:"trace_id"`
	StoryID     uuid.UUID `json:"story_id,omitempty"`
	SceneID     uuid.UUID `json:"scene_id,omitempty"`
	CharID      uuid.UUID `json:"character_id,omitempty"`
	GenerationID uuid.UUID `json:"generation_id,omitempty"`
	SpanID      string    `json:"span_id,omitempty"`
	ParentSpanID string   `json:"parent_span_id,omitempty"`
}

type MetricRecorder interface {
	IncCounter(name string, tags map[string]string, val float64)
	ObserveDuration(name string, tags map[string]string, dur time.Duration)
	RecordValue(name string, val float64, tags map[string]string)
}

type Metrics struct {
	GenerationDuration  string
	PromptCompileDur    string
	MemoryRetrievalDur  string
	RelUpdateDur        string
	TimelineUpdateDur   string
	GenerationTokens    string
	GenerationCost      string
	CacheHitRatio       string
	MemoryRetrievalCount string
}

var DefaultMetrics = Metrics{
	GenerationDuration:  "scene_generation_duration_seconds",
	PromptCompileDur:    "prompt_compile_duration_seconds",
	MemoryRetrievalDur:  "memory_retrieval_duration_seconds",
	RelUpdateDur:        "relationship_update_duration_seconds",
	TimelineUpdateDur:   "timeline_update_duration_seconds",
	GenerationTokens:    "generation_tokens_total",
	GenerationCost:      "generation_cost_usd",
	CacheHitRatio:       "cache_hit_ratio",
	MemoryRetrievalCount: "memory_retrieval_count",
}

type NoopRecorder struct{}

func (NoopRecorder) IncCounter(name string, tags map[string]string, val float64) {}
func (NoopRecorder) ObserveDuration(name string, tags map[string]string, dur time.Duration) {}
func (NoopRecorder) RecordValue(name string, val float64, tags map[string]string) {}

func StartSpan(ctx context.Context, name string) (context.Context, func()) {
	start := time.Now()
	return ctx, func() {
		slog.Debug("span completed",
			"span", name,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	}
}

func WithTrace(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceCtxKey, traceID)
}

func GetTrace(ctx context.Context) string {
	if v, ok := ctx.Value(traceCtxKey).(string); ok {
		return v
	}
	return ""
}

type ctxKey string

const traceCtxKey ctxKey = "trace_id"
