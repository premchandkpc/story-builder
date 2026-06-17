package llm

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type circuitState int32

const (
	circuitClosed   circuitState = 0
	circuitOpen     circuitState = 1
	circuitHalfOpen circuitState = 2
)

var defaultCircuitBreakerConfig = circuitBreakerConfig{
	threshold: 5,
	timeout:   30 * time.Second,
}

type circuitBreakerConfig struct {
	threshold int
	timeout   time.Duration
}

type CircuitBreakerClient struct {
	client LLMClient
	config circuitBreakerConfig

	mu            sync.Mutex
	state         circuitState
	failures      int
	lastFailureAt time.Time
}

func NewCircuitBreakerClient(client LLMClient) *CircuitBreakerClient {
	return &CircuitBreakerClient{
		client: client,
		config: defaultCircuitBreakerConfig,
		state:  circuitClosed,
	}
}

func (c *CircuitBreakerClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	c.mu.Lock()
	if c.state == circuitOpen {
		if time.Since(c.lastFailureAt) > c.config.timeout {
			c.state = circuitHalfOpen
		} else {
			c.mu.Unlock()
			return nil, fmt.Errorf("circuit breaker: %s open", req.Model)
		}
	}
	halfOpen := c.state == circuitHalfOpen
	c.mu.Unlock()

	resp, err := c.client.Complete(ctx, req)

	c.mu.Lock()
	defer c.mu.Unlock()
	if err != nil {
		c.failures++
		c.lastFailureAt = time.Now()
		if c.failures >= c.config.threshold {
			c.state = circuitOpen
		}
		return nil, err
	}

	if halfOpen {
		c.state = circuitClosed
	}
	c.failures = 0
	return resp, nil
}
