package cache

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/premchand/story-builder/internal/llm"
)

type CachedLLMClient struct {
	inner llm.LLMClient
	cache *PromptCache
}

func NewCachedLLMClient(inner llm.LLMClient, cache *PromptCache) *CachedLLMClient {
	return &CachedLLMClient{inner: inner, cache: cache}
}

func (c *CachedLLMClient) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	resp, err := c.cache.Get(ctx, req)
	if err == nil {
		slog.Debug("prompt cache hit", "model", req.Model)
		return resp, nil
	}

	resp, err = c.inner.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	if err := c.cache.Set(ctx, req, resp); err != nil {
		slog.Warn("prompt cache set failed", "error", err)
	}

	return resp, nil
}

type RateLimitedLLMClient struct {
	inner     llm.LLMClient
	limiter   *SlidingWindowRateLimiter
	limitKey  string
}

func NewRateLimitedLLMClient(inner llm.LLMClient, limiter *SlidingWindowRateLimiter, limitKey string) *RateLimitedLLMClient {
	return &RateLimitedLLMClient{inner: inner, limiter: limiter, limitKey: limitKey}
}

func (c *RateLimitedLLMClient) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	allowed, err := c.limiter.AllowWithKey(ctx, c.limitKey, string(req.Model))
	if err != nil {
		return nil, fmt.Errorf("rate limit check: %w", err)
	}
	if !allowed {
		return nil, fmt.Errorf("rate limit exceeded: %s/%s", c.limitKey, req.Model)
	}
	return c.inner.Complete(ctx, req)
}
