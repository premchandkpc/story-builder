package llm

import (
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/premchand/story-builder/internal/cache"
)

type cachedResponse struct {
	Content          string `json:"content"`
	Model            string `json:"model"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
}

type lruEntry struct {
	key   string
	value string
}

type MemoryCache struct {
	mu    sync.Mutex
	items map[string]*list.Element
	order *list.List
	max   int
}

func NewMemoryCache(max int) *MemoryCache {
	return &MemoryCache{
		items: make(map[string]*list.Element),
		order: list.New(),
		max:   max,
	}
}

func (m *MemoryCache) Get(key string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	elem, ok := m.items[key]
	if !ok {
		return "", false
	}
	m.order.MoveToFront(elem)
	return elem.Value.(*lruEntry).value, true
}

func (m *MemoryCache) Set(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if elem, ok := m.items[key]; ok {
		m.order.MoveToFront(elem)
		elem.Value.(*lruEntry).value = value
		return
	}
	elem := m.order.PushFront(&lruEntry{key: key, value: value})
	m.items[key] = elem
	if m.order.Len() > m.max {
		if victim := m.order.Back(); victim != nil {
			m.order.Remove(victim)
			delete(m.items, victim.Value.(*lruEntry).key)
		}
	}
}

type CachedLLMClient struct {
	wrapped LLMClient
	redis   cache.RedisClient
	mem     *MemoryCache
	ttl     time.Duration
	enabled bool

	hitCount  atomic.Int64
	missCount atomic.Int64
}

func NewCachedLLMClient(wrapped LLMClient, store cache.RedisClient, ttl time.Duration) *CachedLLMClient {
	return &CachedLLMClient{
		wrapped: wrapped,
		redis:   store,
		mem:     NewMemoryCache(512),
		ttl:     ttl,
		enabled: true,
	}
}

func (c *CachedLLMClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	key := c.cacheKey(req)

	if cached, hitSource := c.lookup(ctx, key); cached != nil {
		c.hitCount.Add(1)
		slog.Debug("prompt cache hit", "model", req.Model, "source", hitSource)
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.SetAttributes(attribute.String("cache_hit", hitSource))
		}
		return cached, nil
	}

	c.missCount.Add(1)
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(attribute.Bool("cache_miss", true))
	}

	start := time.Now()
	resp, err := c.wrapped.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(attribute.Int("cache_saved_ms", int(time.Since(start).Milliseconds())))
	}

	c.store(ctx, key, resp)
	return resp, nil
}

func (c *CachedLLMClient) lookup(ctx context.Context, key string) (*CompletionResponse, string) {
	if val, ok := c.mem.Get(key); ok {
		var resp cachedResponse
		if json.Unmarshal([]byte(val), &resp) == nil {
			return fromCached(&resp), "memory"
		}
	}
	if c.redis != nil {
		if val, err := c.redis.Get(ctx, key); err == nil {
			var resp cachedResponse
			if json.Unmarshal([]byte(val), &resp) == nil {
				c.mem.Set(key, val)
				return fromCached(&resp), "redis"
			}
		}
	}
	return nil, ""
}

func (c *CachedLLMClient) store(ctx context.Context, key string, resp *CompletionResponse) {
	cached := cachedResponse{
		Content:          resp.Content,
		Model:            resp.Model,
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}
	b, err := json.Marshal(cached)
	if err != nil {
		return
	}
	val := string(b)
	c.mem.Set(key, val)
	if c.redis != nil {
		if err := c.redis.Set(ctx, key, val, c.ttl); err != nil {
			slog.Debug("prompt cache redis set failed", "error", err)
		}
	}
}

func (c *CachedLLMClient) Metrics() (hits, misses int64) {
	return c.hitCount.Load(), c.missCount.Load()
}

func fromCached(r *cachedResponse) *CompletionResponse {
	return &CompletionResponse{
		Content: r.Content,
		Model:   r.Model,
		Usage: Usage{
			PromptTokens:     r.PromptTokens,
			CompletionTokens: r.CompletionTokens,
			TotalTokens:      r.TotalTokens,
		},
	}
}

func (c *CachedLLMClient) cacheKey(req CompletionRequest) string {
	h := sha256.New()
	h.Write([]byte(req.System))
	h.Write([]byte(req.UserMessage))
	h.Write([]byte(string(req.Model)))
	h.Write([]byte(fmt.Sprintf("%f", req.Temperature)))
	if req.Tools != nil {
		if b, err := json.Marshal(req.Tools); err == nil {
			h.Write(b)
		}
	}
	return fmt.Sprintf("prompt:%x", h.Sum(nil))
}
