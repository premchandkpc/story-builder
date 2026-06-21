package llm

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/premchand/story-builder/internal/cache"
)

type cachedResponse struct {
	Content   string `json:"content"`
	Model     string `json:"model"`
	PromptTokens int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type CachedLLMClient struct {
	wrapped LLMClient
	store   cache.RedisClient
	ttl     time.Duration
	enabled bool
}

func NewCachedLLMClient(wrapped LLMClient, store cache.RedisClient, ttl time.Duration) *CachedLLMClient {
	return &CachedLLMClient{
		wrapped: wrapped,
		store:   store,
		ttl:     ttl,
		enabled: store != nil,
	}
}

func (c *CachedLLMClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if !c.enabled {
		return c.wrapped.Complete(ctx, req)
	}

	key := c.cacheKey(req)

	if cached, err := c.store.Get(ctx, key); err == nil {
		var resp cachedResponse
		if json.Unmarshal([]byte(cached), &resp) == nil {
			slog.Debug("prompt cache hit", "model", req.Model)
			return &CompletionResponse{
				Content: resp.Content,
				Model:   resp.Model,
				Usage: Usage{
					PromptTokens:     resp.PromptTokens,
					CompletionTokens: resp.CompletionTokens,
					TotalTokens:      resp.TotalTokens,
				},
			}, nil
		}
	}

	resp, err := c.wrapped.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	cached := cachedResponse{
		Content:          resp.Content,
		Model:            resp.Model,
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}
	if b, err := json.Marshal(cached); err == nil {
		if err := c.store.Set(ctx, key, string(b), c.ttl); err != nil {
			slog.Debug("prompt cache set failed", "error", err)
		}
	}

	return resp, nil
}

func (c *CachedLLMClient) cacheKey(req CompletionRequest) string {
	h := sha256.New()
	h.Write([]byte(req.System))
	h.Write([]byte(req.UserMessage))
	h.Write([]byte(fmt.Sprintf("%f", req.Temperature)))
	if req.Tools != nil {
		if b, err := json.Marshal(req.Tools); err == nil {
			h.Write(b)
		}
	}
	return fmt.Sprintf("prompt:%x", h.Sum(nil))
}
