package cache

import (
	"context"
	"time"

	"github.com/premchand/story-builder/internal/cache"
)

type NoopRedisClient struct{}

func NewNoopRedisClient() *NoopRedisClient {
	return &NoopRedisClient{}
}

func (c *NoopRedisClient) Get(ctx context.Context, key string) (string, error) {
	return "", cache.ErrCacheMiss
}

func (c *NoopRedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return nil
}

func (c *NoopRedisClient) Del(ctx context.Context, keys ...string) error {
	return nil
}

func (c *NoopRedisClient) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	return true, nil
}

func (c *NoopRedisClient) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	return int64(1), nil
}

func (c *NoopRedisClient) Close() error {
	return nil
}

func (c *NoopRedisClient) Ping(ctx context.Context) error {
	return nil
}
