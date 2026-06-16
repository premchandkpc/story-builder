package telemetry

import (
	"log/slog"
	"sync/atomic"
	"time"
)

type Counter struct {
	name   string
	desc   string
	value  atomic.Int64
}

func NewCounter(name, desc string) *Counter {
	return &Counter{name: name, desc: desc}
}

func (c *Counter) Add(n int64, attrs ...slog.Attr) {
	v := c.value.Add(n)
	args := make([]any, 0, len(attrs))
	for _, a := range attrs {
		args = append(args, a)
	}
	slog.Debug("metric:counter",
		slog.String("metric", c.name),
		slog.Int64("value", v),
		slog.Int64("delta", n),
		slog.Group("attrs", args...),
	)
}

func (c *Counter) Inc(attrs ...slog.Attr) {
	c.Add(1, attrs...)
}

func (c *Counter) Value() int64 {
	return c.value.Load()
}

type Histogram struct {
	name string
	desc string
}

func NewHistogram(name, desc string) *Histogram {
	return &Histogram{name: name, desc: desc}
}

func (h *Histogram) Record(d time.Duration, attrs ...slog.Attr) {
	args := make([]any, 0, len(attrs))
	for _, a := range attrs {
		args = append(args, a)
	}
	slog.Debug("metric:histogram",
		slog.String("metric", h.name),
		slog.Duration("value", d),
		slog.Group("attrs", args...),
	)
}

var (
	LLMCalls      = NewCounter("llm.calls.total", "Total LLM completion calls")
	LLMErrors     = NewCounter("llm.calls.errors", "LLM completion call errors")
	LLMLatency    = NewHistogram("llm.latency", "LLM completion latency")
	HTTPRequests  = NewCounter("http.requests.total", "Total HTTP requests")
	HTTPErrors    = NewCounter("http.requests.errors", "HTTP 5xx responses")
	HTTPLatency   = NewHistogram("http.latency", "HTTP request latency")
	WorkerCalls   = NewCounter("river.workers.total", "Total River worker executions")
	WorkerErrors  = NewCounter("river.workers.errors", "River worker execution errors")
	WorkerLatency = NewHistogram("river.workers.latency", "River worker execution latency")
)
