package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/premchand/story-builder/internal/trace"
)

// Router dispatches completion requests to the appropriate provider based on model tier.
type Router struct {
	anthropic LLMClient
	local     LLMClient
}

func NewRouter(anthropic, local LLMClient) *Router {
	return &Router{anthropic: anthropic, local: local}
}

func (r *Router) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if req.MaxRetries <= 0 {
		req.MaxRetries = 2
	}

	ctx, span := trace.StartSpan(ctx, "llm.Complete")
	if span != nil {
		trace.SetAttribute(span, "model", string(req.Model))
		trace.SetAttribute(span, "system_len", len(req.System))
		trace.SetAttribute(span, "user_len", len(req.UserMessage))
	}
	defer trace.End(span)

	client, ok := r.clientForModel(req.Model)
	if !ok {
		err := fmt.Errorf("unsupported model tier: %s", req.Model)
		trace.SetError(span, err)
		return nil, err
	}

	var lastErr error
	for attempt := 0; attempt <= req.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			trace.SetError(span, err)
			return nil, err
		}

		resp, err := client.Complete(ctx, req)
		if err == nil {
			if resp != nil && resp.Model == "" {
				resp.Model = string(req.Model)
			}
			if req.ValidateJSON && !json.Valid([]byte(strings.TrimSpace(resp.Content))) {
				err = fmt.Errorf("invalid JSON response")
				lastErr = err
				if attempt < req.MaxRetries {
					wait := backoff(attempt, req.Model)
					timer := time.NewTimer(wait)
					select {
					case <-ctx.Done():
						timer.Stop()
						return nil, ctx.Err()
					case <-timer.C:
					}
				}
				continue
			}
			return resp, nil
		}
		lastErr = err
		if attempt < req.MaxRetries {
			wait := backoff(attempt, req.Model)
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
		}
	}
	trace.SetError(span, lastErr)
	return nil, lastErr
}

// backoff returns exponential backoff with jitter based on model tier.
// Anthropic:  initial 1s, factor 2, max 15s
// Local:      initial 200ms, factor 2, max 5s
func backoff(attempt int, tier ModelTier) time.Duration {
	var base time.Duration
	var max time.Duration
	switch tier {
	case ModelSonnet, ModelHaiku:
		base = 1 * time.Second
		max = 15 * time.Second
	default:
		base = 200 * time.Millisecond
		max = 5 * time.Second
	}
	d := base * (1 << attempt) // 2^attempt
	if d > max {
		d = max
	}
	// ±25% jitter
	jitter := time.Duration(float64(d) * (0.5 - rand.Float64()))
	return d + jitter
}

func (r *Router) clientForModel(model ModelTier) (LLMClient, bool) {
	switch model {
	case ModelSonnet, ModelHaiku:
		if r.anthropic != nil {
			return r.anthropic, true
		}
	case ModelLocal:
		if r.local != nil {
			return r.local, true
		}
	}
	return nil, false
}
