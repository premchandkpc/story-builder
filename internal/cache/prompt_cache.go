package cache

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/premchand/story-builder/internal/llm"
)

type PromptCache struct {
	client RedisClient
	ttl    time.Duration
}

func NewPromptCache(client RedisClient) *PromptCache {
	return &PromptCache{
		client: client,
		ttl:    24 * time.Hour,
	}
}

func promptCacheKey(system, user string) string {
	h := sha256.New()
	h.Write([]byte(system))
	h.Write([]byte(user))
	return fmt.Sprintf("prompt:%x", h.Sum(nil))
}

func (c *PromptCache) Get(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	key := promptCacheKey(req.System, req.UserMessage)
	data, err := c.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	var resp llm.CompletionResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return nil, fmt.Errorf("prompt cache unmarshal: %w", err)
	}
	return &resp, nil
}

func (c *PromptCache) Set(ctx context.Context, req llm.CompletionRequest, resp *llm.CompletionResponse) error {
	key := promptCacheKey(req.System, req.UserMessage)
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("prompt cache marshal: %w", err)
	}
	return c.client.Set(ctx, key, string(data), c.ttl)
}

func (c *PromptCache) Invalidate(ctx context.Context, system, user string) error {
	key := promptCacheKey(system, user)
	return c.client.Del(ctx, key)
}
