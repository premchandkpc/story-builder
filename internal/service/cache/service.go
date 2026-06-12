package cache

import (
	cacheredis "github.com/premchand/story-builder/internal/cache"
	"github.com/premchand/story-builder/internal/llm"
)

type Cache struct {
	ContextCache *cacheredis.ContextCache
	PromptCache  *cacheredis.PromptCache
	RateLimiter  *cacheredis.SlidingWindowRateLimiter
}

func New(redisAddr, redisPass string, redisDB int) (*Cache, error) {
	client, err := cacheredis.NewGoRedisClient(redisAddr, redisPass, redisDB)
	if err != nil {
		return nil, err
	}
	return &Cache{
		ContextCache: cacheredis.NewContextCache(client),
		PromptCache:  cacheredis.NewPromptCache(client),
		RateLimiter:  cacheredis.NewSlidingWindowRateLimiter(client, cacheredis.DefaultRateLimits),
	}, nil
}

func NewNoop() *Cache {
	return nil
}

func (c *Cache) WrapLLMClient(client llm.LLMClient) llm.LLMClient {
	if c.PromptCache != nil {
		client = cacheredis.NewCachedLLMClient(client, c.PromptCache)
	}
	if c.RateLimiter != nil {
		client = cacheredis.NewRateLimitedLLMClient(client, c.RateLimiter, "llm:anthropic")
	}
	return client
}

func (c *Cache) IsAvailable() bool {
	return c != nil
}
