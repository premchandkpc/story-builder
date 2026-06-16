package telemetry

import (
	"context"
	"log/slog"
	"time"

	"github.com/premchand/story-builder/internal/llm"
)

type TracedLLMClient struct {
	inner llm.LLMClient
}

func NewTracedLLMClient(inner llm.LLMClient) *TracedLLMClient {
	return &TracedLLMClient{inner: inner}
}

func (c *TracedLLMClient) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	ctx, span := StartSpan(ctx, "llm.complete",
		slog.String("model", string(req.Model)),
		slog.String("prompt_type", detectPromptType(req)),
		slog.Int("max_tokens", req.MaxTokens),
		slog.Float64("temperature", req.Temperature),
	)
	start := time.Now()

	resp, err := c.inner.Complete(ctx, req)
	latency := time.Since(start)

	attrs := []slog.Attr{
		slog.String("model", string(req.Model)),
		slog.Duration("latency", latency),
	}

	if err != nil {
		span.RecordError(err)
		span.End(WithLevel(slog.LevelError))
		LLMErrors.Inc(attrs...)
		LLMCalls.Add(1, append(attrs, slog.Bool("success", false))...)
		LLMLatency.Record(latency, append(attrs, slog.Bool("success", false))...)
		return nil, err
	}

	if resp != nil {
		attrs = append(attrs, slog.String("response_model", resp.Model))
		span.SetAttrs(slog.String("response_model", resp.Model))
	}
	span.End()
	LLMCalls.Add(1, append(attrs, slog.Bool("success", true))...)
	LLMLatency.Record(latency, append(attrs, slog.Bool("success", true))...)
	return resp, nil
}

var promptTypePrefixes = []struct {
	name string
	pfix string
}{
	{"scene_prose", "Write ONE scene"},
	{"state_extract", "continuity clerk"},
	{"summary_update", "running plot summary"},
	{"join_merge", "converging"},
	{"canon_validate", "continuity editor"},
	{"outline_story", "story architect"},
	{"generate_title", "title generator"},
}

func detectPromptType(req llm.CompletionRequest) string {
	if req.System == "" {
		return "unknown"
	}
	for _, pt := range promptTypePrefixes {
		if len(req.System) >= len(pt.pfix) && req.System[:len(pt.pfix)] == pt.pfix {
			return pt.name
		}
	}
	return "custom"
}
