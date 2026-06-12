package cache

import (
	"context"
	"errors"
	"time"
)

var ErrCacheMiss = errors.New("cache: miss")

type KeyPrefix string

const (
	PrefixContext    KeyPrefix = "story:%s:context"
	PrefixContextHash KeyPrefix = "story:%s:context:hash"
	PrefixPrompt     KeyPrefix = "prompt:%s"
	PrefixPipeline   KeyPrefix = "pipeline:%s"
	PrefixLock       KeyPrefix = "lock:%s"
	PrefixRateLimit  KeyPrefix = "ratelimit:%s"
)

type CacheEntry struct {
	Data      string    `json:"data"`
	CreatedAt time.Time `json:"created_at"`
	TTL       time.Duration `json:"ttl"`
}

type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error)
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error)
	Close() error
	Ping(ctx context.Context) error
}
