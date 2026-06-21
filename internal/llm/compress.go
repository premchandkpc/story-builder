package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type CompressClient struct {
	wrapped LLMClient
	baseURL string
	apiKey  string
	client  *http.Client
	enabled bool
}

func NewCompressClient(wrapped LLMClient, baseURL, apiKey string) *CompressClient {
	return &CompressClient{
		wrapped: wrapped,
		baseURL: baseURL,
		apiKey:  apiKey,
		enabled: baseURL != "",
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				ResponseHeaderTimeout: 15 * time.Second,
				IdleConnTimeout:       60 * time.Second,
			},
		},
	}
}

type compressMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type compressRequest struct {
	Messages []compressMessage `json:"messages"`
	Model    string            `json:"model,omitempty"`
}

type compressMetrics struct {
	TokensBefore int `json:"tokens_before"`
	TokensAfter  int `json:"tokens_after"`
	TokensSaved  int `json:"tokens_saved"`
}

type compressResponse struct {
	Messages []compressMessage `json:"messages"`
	Metrics  compressMetrics   `json:"metrics,omitempty"`
}

func (c *CompressClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if !c.enabled {
		return c.wrapped.Complete(ctx, req)
	}

	msgs := []compressMessage{}
	if req.System != "" {
		msgs = append(msgs, compressMessage{Role: "system", Content: req.System})
	}
	msgs = append(msgs, compressMessage{Role: "user", Content: req.UserMessage})

	body := compressRequest{
		Messages: msgs,
		Model:    string(req.Model),
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("compress marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/compress", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("compress new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	res, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("compress proxy: %w", err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("compress read: %w", err)
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("compress proxy: %d %s", res.StatusCode, string(raw))
	}

	var reply compressResponse
	if err := json.Unmarshal(raw, &reply); err != nil {
		return nil, fmt.Errorf("compress decode: %w", err)
	}

	compressedReq := req
	compressedReq.System = ""
	compressedReq.UserMessage = ""
	for _, m := range reply.Messages {
		switch m.Role {
		case "system":
			compressedReq.System = m.Content
		default:
			compressedReq.UserMessage = m.Content
		}
	}

	return c.wrapped.Complete(ctx, compressedReq)
}
