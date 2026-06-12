package llm

import (
	"context"
	"fmt"
	"time"
)

// Router dispatches completion requests to the appropriate provider based on model tier.
type Router struct {
	anthropic LLMClient
	ollama    LLMClient
}

func NewRouter(anthropic, ollama LLMClient) *Router {
	return &Router{anthropic: anthropic, ollama: ollama}
}

func (r *Router) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if req.MaxRetries <= 0 {
		req.MaxRetries = 2
	}

	client, ok := r.clientForModel(req.Model)
	if !ok {
		return nil, fmt.Errorf("unsupported model tier: %s", req.Model)
	}

	var lastErr error
	for attempt := 0; attempt <= req.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		resp, err := client.Complete(ctx, req)
		if err == nil {
			if resp != nil && resp.Model == "" {
				resp.Model = string(req.Model)
			}
			return resp, nil
		}
		lastErr = err
		if attempt < req.MaxRetries {
			timer := time.NewTimer(time.Duration(250*(attempt+1)) * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
		}
	}
	return nil, lastErr
}

func (r *Router) clientForModel(model ModelTier) (LLMClient, bool) {
	switch model {
	case ModelSonnet, ModelHaiku:
		if r.anthropic != nil {
			return r.anthropic, true
		}
	case ModelLocal:
		if r.ollama != nil {
			return r.ollama, true
		}
	}
	return nil, false
}
