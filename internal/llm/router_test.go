package llm

import (
	"context"
	"testing"
)

type stubClient struct {
	name string
	resp *CompletionResponse
	err  error
}

func (s *stubClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	return s.resp, s.err
}

func TestRouterRoutesByModelTier(t *testing.T) {
	anthropic := &stubClient{name: "anthropic", resp: &CompletionResponse{Content: "claude", Model: "claude-sonnet-4-20250514"}}
	ollama := &stubClient{name: "ollama", resp: &CompletionResponse{Content: "llama", Model: "llama3.2:3b"}}
	router := NewRouter(anthropic, ollama)

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
		t.Fatalf("expected ollama response, got %q", resp.Content)
	}
}
