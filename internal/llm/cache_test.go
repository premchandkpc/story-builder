package llm

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

type mockLLMClient struct {
	callCount atomic.Int32
	fn        func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
}

func (m *mockLLMClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	m.callCount.Add(1)
	if m.fn != nil {
		return m.fn(ctx, req)
	}
	return &CompletionResponse{
		Content: "hello",
		Model:   "claude-sonnet",
		Usage:   Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	}, nil
}

func TestCachedLLMClient_Hit(t *testing.T) {
	mock := &mockLLMClient{}
	c := NewCachedLLMClient(mock, nil, 0)
	req := CompletionRequest{System: "sys", UserMessage: "usr", Temperature: 0.5}

	resp1, err1 := c.Complete(context.Background(), req)
	if err1 != nil {
		t.Fatal(err1)
	}
	hits1, misses1 := c.Metrics()
	if misses1 != 1 {
		t.Fatalf("expected 1 miss, got %d", misses1)
	}

	resp2, err2 := c.Complete(context.Background(), req)
	if err2 != nil {
		t.Fatal(err2)
	}
	hits2, misses2 := c.Metrics()
	if hits2-hits1 != 1 {
		t.Fatalf("expected 1 hit, got %d (hits=%d, misses=%d)", hits2-hits1, hits2, misses2)
	}

	if resp1.Content != resp2.Content {
		t.Fatalf("expected cached content %q, got %q", resp1.Content, resp2.Content)
	}
	if mock.callCount.Load() != 1 {
		t.Fatalf("expected 1 wrapped call, got %d", mock.callCount.Load())
	}
}

func TestCachedLLMClient_Miss(t *testing.T) {
	mock := &mockLLMClient{}
	c := NewCachedLLMClient(mock, nil, 0)

	req1 := CompletionRequest{System: "sys1", UserMessage: "usr1", Temperature: 0.5}
	req2 := CompletionRequest{System: "sys2", UserMessage: "usr2", Temperature: 0.5}

	_, _ = c.Complete(context.Background(), req1)
	_, _ = c.Complete(context.Background(), req2)

	if mock.callCount.Load() != 2 {
		t.Fatalf("expected 2 wrapped calls, got %d", mock.callCount.Load())
	}
	hits, misses := c.Metrics()
	if hits != 0 {
		t.Fatalf("expected 0 hits, got %d", hits)
	}
	if misses != 2 {
		t.Fatalf("expected 2 misses, got %d", misses)
	}
}

func TestCachedLLMClient_DifferentTemps(t *testing.T) {
	mock := &mockLLMClient{}
	c := NewCachedLLMClient(mock, nil, 0)

	req1 := CompletionRequest{System: "sys", UserMessage: "usr", Temperature: 0.5}
	req2 := CompletionRequest{System: "sys", UserMessage: "usr", Temperature: 0.8}

	_, _ = c.Complete(context.Background(), req1)
	_, _ = c.Complete(context.Background(), req2)

	if mock.callCount.Load() != 2 {
		t.Fatalf("expected 2 calls for different temps, got %d", mock.callCount.Load())
	}
}

func TestCachedLLMClient_DifferentTools(t *testing.T) {
	mock := &mockLLMClient{}
	c := NewCachedLLMClient(mock, nil, 0)

	req1 := CompletionRequest{System: "sys", UserMessage: "usr", Temperature: 0.5}
	req2 := CompletionRequest{
		System:       "sys",
		UserMessage:  "usr",
		Temperature:  0.5,
		Tools:        []ToolDefinition{{Name: "foo"}},
	}

	_, _ = c.Complete(context.Background(), req1)
	_, _ = c.Complete(context.Background(), req2)

	if mock.callCount.Load() != 2 {
		t.Fatalf("expected 2 calls for different tools, got %d", mock.callCount.Load())
	}
}

func TestCachedLLMClient_ErrorNotCached(t *testing.T) {
	calls := 0
	mock := &mockLLMClient{
		fn: func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
			calls++
			return nil, errors.New("llm error")
		},
	}
	c := NewCachedLLMClient(mock, nil, 0)
	req := CompletionRequest{System: "sys", UserMessage: "usr", Temperature: 0.5}

	_, err := c.Complete(context.Background(), req)
	if err == nil {
		t.Fatal("expected error on first call")
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestMemoryCache(t *testing.T) {
	m := NewMemoryCache(3)
	m.Set("a", "1")
	m.Set("b", "2")

	v, ok := m.Get("a")
	if !ok || v != "1" {
		t.Fatalf("expected '1', got %q (ok=%v)", v, ok)
	}
	v, ok = m.Get("c")
	if ok {
		t.Fatalf("expected miss for 'c', got %q", v)
	}

	m.Set("c", "3")
	m.Set("d", "4")

	if _, ok := m.Get("b"); ok {
		t.Fatal("expected 'b' to be evicted (LRU)")
	}
	if v, ok := m.Get("a"); !ok || v != "1" {
		t.Fatalf("expected 'a' to exist, got %q (ok=%v)", v, ok)
	}
	if v, ok := m.Get("c"); !ok || v != "3" {
		t.Fatalf("expected 'c' to exist, got %q (ok=%v)", v, ok)
	}
}

func TestMemoryCache_Update(t *testing.T) {
	m := NewMemoryCache(3)
	m.Set("a", "1")
	m.Set("a", "2")

	v, ok := m.Get("a")
	if !ok || v != "2" {
		t.Fatalf("expected updated value '2', got %q", v)
	}
}

func TestMemoryCache_MaxSize(t *testing.T) {
	m := NewMemoryCache(2)
	m.Set("a", "1")
	m.Set("b", "2")
	m.Set("c", "3")

	if _, ok := m.Get("a"); ok {
		t.Fatal("expected 'a' to be evicted")
	}
	if _, ok := m.Get("b"); !ok {
		t.Fatal("expected 'b' to exist")
	}
	if _, ok := m.Get("c"); !ok {
		t.Fatal("expected 'c' to exist")
	}

	m.Get("b")
	m.Set("d", "4")

	if _, ok := m.Get("c"); ok {
		t.Fatal("expected 'c' to be evicted (b was recently used)")
	}
}
