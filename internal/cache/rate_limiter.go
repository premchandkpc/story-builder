package cache

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

const rateLimitScript = `
local key = KEYS[1]
local window = tonumber(ARGV[1])
local max = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local member = ARGV[4]

redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
local count = redis.call('ZCARD', key)
if count >= max then
	return 0
end
redis.call('ZADD', key, now, member)
redis.call('EXPIRE', key, window / 1000)
return 1
`

type RateLimitConfig struct {
	Key    string
	Limit  int
	Window time.Duration
}

type SlidingWindowRateLimiter struct {
	client  RedisClient
	configs []RateLimitConfig
}

func NewSlidingWindowRateLimiter(client RedisClient, configs []RateLimitConfig) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		client:  client,
		configs: configs,
	}
}

func (rl *SlidingWindowRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	for _, cfg := range rl.configs {
		if !strings.HasPrefix(key, cfg.Key) {
			continue
		}
		return rl.check(ctx, key, cfg)
	}
	return true, nil
}

func (rl *SlidingWindowRateLimiter) AllowWithKey(ctx context.Context, resource, suffix string) (bool, error) {
	key := fmt.Sprintf(string(PrefixRateLimit), resource+":"+suffix)
	return rl.check(ctx, key, rl.matchConfig(resource))
}

func (rl *SlidingWindowRateLimiter) matchConfig(key string) RateLimitConfig {
	for _, cfg := range rl.configs {
		if cfg.Key == key {
			return cfg
		}
	}
	return RateLimitConfig{Key: key, Limit: 100, Window: time.Minute}
}

func (rl *SlidingWindowRateLimiter) check(ctx context.Context, redisKey string, cfg RateLimitConfig) (bool, error) {
	now := time.Now().UnixMilli()
	member := fmt.Sprintf("%d:%d", now, rand.Int63())
	windowMs := cfg.Window.Milliseconds()

	result, err := rl.client.Eval(ctx, rateLimitScript, []string{redisKey},
		windowMs, cfg.Limit, now, member,
	)
	if err != nil {
		return false, fmt.Errorf("rate limit eval: %w", err)
	}

	switch v := result.(type) {
	case int64:
		return v == 1, nil
	default:
		// Try int
		if i, ok := result.(int); ok {
			return i == 1, nil
		}
		return false, nil
	}
}

var DefaultRateLimits = []RateLimitConfig{
	{Key: "http:api", Limit: 1000, Window: time.Minute},
}
