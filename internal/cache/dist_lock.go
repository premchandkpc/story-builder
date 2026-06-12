package cache

import (
	"context"
	"fmt"
	"time"
)

const (
	DefaultLockTTL   = 30 * time.Second
	LockRetryInterval = 100 * time.Millisecond
	MaxLockRetries    = 50
)

type DistLock struct {
	client RedisClient
	ttl    time.Duration
}

func NewDistLock(client RedisClient) *DistLock {
	return &DistLock{
		client: client,
		ttl:    DefaultLockTTL,
	}
}

func (l *DistLock) Acquire(ctx context.Context, resource string) (bool, error) {
	key := fmt.Sprintf(string(PrefixLock), resource)
	return l.client.SetNX(ctx, key, "locked", l.ttl)
}

func (l *DistLock) AcquireWithRetry(ctx context.Context, resource string) error {
	for i := 0; i < MaxLockRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		ok, err := l.Acquire(ctx, resource)
		if err != nil {
			return fmt.Errorf("lock acquire: %w", err)
		}
		if ok {
			return nil
		}
		time.Sleep(LockRetryInterval)
	}
	return fmt.Errorf("lock timeout: %s", resource)
}

func (l *DistLock) Release(ctx context.Context, resource string) error {
	key := fmt.Sprintf(string(PrefixLock), resource)
	return l.client.Del(ctx, key)
}
