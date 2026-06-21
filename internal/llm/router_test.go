package llm

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type stubClient struct {
	name     string
	resp     *CompletionResponse
	err      error
	callCount int32
}

func (s *stubClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	atomic.AddInt32(&s.callCount, 1)
	return s.resp, s.err
}

type failingThenOKClient struct {
	failCount  int32
	callCount  int32
	okResp     *CompletionResponse
}

func (c *failingThenOKClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	atomic.AddInt32(&c.callCount, 1)
	if atomic.LoadInt32(&c.callCount) <= c.failCount {
		return nil, errors.New("transient failure")
	}
	return c.okResp, nil
}

func TestRouterRoutesByModelTier(t *testing.T) {
	anthropic := &stubClient{name: "anthropic", resp: &CompletionResponse{Content: "claude", Model: "claude-sonnet-4-20250514"}}
	local := &stubClient{name: "local", resp: &CompletionResponse{Content: "llama", Model: "llama3.2:3b"}}
	router := NewRouter(anthropic, local)

	resp, err := router.Complete(context.Background(), CompletionRequest{Model: ModelSonnet, UserMessage: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "claude" {
		t.Fatalf("expected anthropic response, got %q", resp.Content)
	}

	resp, err = router.Complete(context.Background(), CompletionRequest{Model: ModelLocal, UserMessage: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "llama" {
		t.Fatalf("expected local response, got %q", resp.Content)
	}
}

func TestRouter_RetriesOnFailure(t *testing.T) {
	client := &failingThenOKClient{failCount: 1, okResp: &CompletionResponse{Content: "ok"}}
	router := NewRouter(client, client)

	resp, err := router.Complete(context.Background(), CompletionRequest{
		Model:       ModelLocal,
		UserMessage: "hello",
		MaxRetries:  2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "ok" {
		t.Fatalf("expected 'ok', got %q", resp.Content)
	}
	if calls := atomic.LoadInt32(&client.callCount); calls != 2 {
		t.Fatalf("expected 2 calls (1 fail + 1 retry), got %d", calls)
	}
}

func TestRouter_RetriesExhausted(t *testing.T) {
	client := &failingThenOKClient{failCount: 3, okResp: &CompletionResponse{Content: "should not happen"}}
	router := NewRouter(client, client)

	_, err := router.Complete(context.Background(), CompletionRequest{
		Model:       ModelLocal,
		UserMessage: "hello",
		MaxRetries:  2,
	})
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
}

func TestRouter_ValidateJSON_RetriesOnInvalidJSON(t *testing.T) {
	client := &stubClient{
		resp: &CompletionResponse{Content: "not valid json { missing closing"},
		callCount: 0,
	}
	router := NewRouter(client, client)

	_, err := router.Complete(context.Background(), CompletionRequest{
		Model:        ModelLocal,
		UserMessage:  "hello",
		MaxRetries:   2,
		ValidateJSON: true,
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if calls := atomic.LoadInt32(&client.callCount); calls != 3 {
		t.Fatalf("expected 3 calls (1 initial + 2 retries), got %d", calls)
	}
}

func TestRouter_ValidateJSON_PassesValidJSON(t *testing.T) {
	client := &stubClient{
		resp: &CompletionResponse{Content: `{"key": "value"}`},
	}
	router := NewRouter(client, client)

	resp, err := router.Complete(context.Background(), CompletionRequest{
		Model:        ModelLocal,
		UserMessage:  "hello",
		ValidateJSON: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != `{"key": "value"}` {
		t.Fatalf("expected valid JSON, got %q", resp.Content)
	}
}

func TestRouter_UnknownModel(t *testing.T) {
	router := NewRouter(nil, nil)
	_, err := router.Complete(context.Background(), CompletionRequest{
		Model: "unknown-model",
	})
	if err == nil {
		t.Fatal("expected error for unknown model")
	}
}

func TestRouter_ContextCancelled(t *testing.T) {
	client := &stubClient{resp: &CompletionResponse{Content: "should not be called"}}
	router := NewRouter(client, client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := router.Complete(ctx, CompletionRequest{
		Model:      ModelLocal,
		MaxRetries: 2,
	})
	if err == nil {
		t.Fatal("expected context cancelled error")
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	client := &stubClient{err: errors.New("always fails")}
	cb := NewCircuitBreakerClient(client)

	for i := 0; i < 5; i++ {
		_, err := cb.Complete(context.Background(), CompletionRequest{Model: ModelLocal})
		if err == nil {
			t.Fatalf("expected error on attempt %d", i+1)
		}
	}

	_, err := cb.Complete(context.Background(), CompletionRequest{Model: ModelLocal})
	if err == nil {
		t.Fatal("expected circuit breaker open error")
	}
	if err.Error() != "circuit breaker: local-7b open" {
		t.Fatalf("expected circuit breaker message, got %q", err.Error())
	}
}

func TestCircuitBreaker_ClosesOnSuccess(t *testing.T) {
	failClient := &stubClient{err: errors.New("fail")}
	cb := NewCircuitBreakerClient(failClient)

	for i := 0; i < 5; i++ {
		cb.Complete(context.Background(), CompletionRequest{Model: ModelLocal})
	}

	okClient := &stubClient{resp: &CompletionResponse{Content: "ok"}}
	cb.client = okClient

	cb.mu.Lock()
	cb.state = circuitHalfOpen
	cb.lastFailureAt = time.Now().Add(-31 * time.Second)
	cb.mu.Unlock()

	resp, err := cb.Complete(context.Background(), CompletionRequest{Model: ModelLocal})
	if err != nil {
		t.Fatalf("unexpected error after half-open probe: %v", err)
	}
	if resp.Content != "ok" {
		t.Fatalf("expected 'ok', got %q", resp.Content)
	}

	cb.mu.Lock()
	state := cb.state
	cb.mu.Unlock()
	if state != circuitClosed {
		t.Fatal("expected circuit to close after successful probe")
	}
}


