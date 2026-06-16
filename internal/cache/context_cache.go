package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/premchand/story-builder/internal/compiler"
)

type ContextCache struct {
	client RedisClient
	ttl    time.Duration
}

func NewContextCache(client RedisClient) *ContextCache {
	return &ContextCache{
		client: client,
		ttl:    5 * time.Minute,
	}
}

func (c *ContextCache) Get(ctx context.Context, storyID, sceneID string) (*compiler.CompiledContext, error) {
	key := fmt.Sprintf(string(PrefixContext), storyID, sceneID)
	data, err := c.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("cache get: %w", err)
	}
	var cc compiler.CompiledContext
	if err := json.Unmarshal([]byte(data), &cc); err != nil {
		return nil, fmt.Errorf("cache unmarshal: %w", err)
	}
	return &cc, nil
}

func (c *ContextCache) Set(ctx context.Context, storyID, sceneID string, cc *compiler.CompiledContext) error {
	key := fmt.Sprintf(string(PrefixContext), storyID, sceneID)
	data, err := json.Marshal(cc)
	if err != nil {
		return fmt.Errorf("cache marshal: %w", err)
	}
	return c.client.Set(ctx, key, string(data), c.ttl)
}

func (c *ContextCache) Invalidate(ctx context.Context, storyID, sceneID string) error {
	key := fmt.Sprintf(string(PrefixContext), storyID, sceneID)
	return c.client.Del(ctx, key)
}

func (c *ContextCache) GetHash(ctx context.Context, storyID, sceneID string) (string, error) {
	key := fmt.Sprintf(string(PrefixContextHash), storyID, sceneID)
	return c.client.Get(ctx, key)
}

func (c *ContextCache) SetHash(ctx context.Context, storyID, sceneID, hash string) error {
	key := fmt.Sprintf(string(PrefixContextHash), storyID, sceneID)
	return c.client.Set(ctx, key, hash, c.ttl)
}
