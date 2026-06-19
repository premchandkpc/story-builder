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

func NewAnthropicClient(apiKey string) *AnthropicClient {
	transport := &http.Transport{
		ResponseHeaderTimeout: 30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	}
	return &AnthropicClient{apiKey: apiKey, http: &http.Client{
		Timeout:   60 * time.Second,
		Transport: transport,
	}}
}

type AnthropicClient struct {
	apiKey string
	http   *http.Client
}

func (c *AnthropicClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	model := string(req.Model)
	if model == "" || model == "claude-sonnet" {
		model = "claude-sonnet-4-20250514"
	} else if model == "claude-haiku" {
		model = "claude-haiku-3-5-20250224"
	}

	body := map[string]interface{}{
		"model":      model,
		"max_tokens": req.MaxTokens,
		"messages":   []map[string]string{{"role": "user", "content": req.UserMessage}},
	}
	if req.System != "" {
		body["system"] = req.System
	}
	body["temperature"] = req.Temperature

	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("anthropic marshal: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("anthropic new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	res, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic: %w", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("anthropic read: %w", err)
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("anthropic: %d %s", res.StatusCode, string(raw))
	}

	var reply struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return nil, fmt.Errorf("anthropic decode: %w", err)
	}

	text := ""
	for _, c := range reply.Content {
		if c.Type == "text" {
			text += c.Text
		}
	}
	return &CompletionResponse{Content: text, Model: model}, nil
}

func NewOllamaClient(baseURL, defaultModel string) *OllamaClient {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if defaultModel == "" {
		defaultModel = "llama3.2:3b"
	}
	return &OllamaClient{baseURL: baseURL, defaultModel: defaultModel, http: &http.Client{
		Timeout: 300 * time.Second,
		Transport: &http.Transport{
			ResponseHeaderTimeout: 120 * time.Second,
			IdleConnTimeout:       90 * time.Second,
		},
	}}
}

type OllamaClient struct {
	baseURL      string
	defaultModel string
	http         *http.Client
}

func (c *OllamaClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	model := string(req.Model)
	if model == "" || model == "local-7b" {
		model = c.defaultModel
	}

	messages := []map[string]string{}
	if req.System != "" {
		messages = append(messages, map[string]string{"role": "system", "content": req.System})
	}
	messages = append(messages, map[string]string{"role": "user", "content": req.UserMessage})

	body := map[string]interface{}{
		"model":    model,
		"messages": messages,
	}
	body["temperature"] = req.Temperature
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ollama marshal: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat/completions", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("ollama new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama: %w", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("ollama read: %w", err)
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("ollama: %d %s", res.StatusCode, string(raw))
	}

	var reply struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return nil, fmt.Errorf("ollama decode: %w", err)
	}
	if len(reply.Choices) == 0 {
		return nil, fmt.Errorf("ollama: empty response")
	}
	return &CompletionResponse{Content: reply.Choices[0].Message.Content, Model: model}, nil
}
