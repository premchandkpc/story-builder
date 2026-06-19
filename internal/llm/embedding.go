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

type EmbeddingService interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
	Model() string
}

type OllamaEmbeddingService struct {
	baseURL string
	model   string
	http    *http.Client
}

func NewOllamaEmbeddingService(baseURL, model string) *OllamaEmbeddingService {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "nomic-embed-text"
	}
	return &OllamaEmbeddingService{
		baseURL: baseURL,
		model:   model,
		http: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				ResponseHeaderTimeout: 10 * time.Second,
				IdleConnTimeout:       90 * time.Second,
			},
		},
	}
}

func (s *OllamaEmbeddingService) Model() string {
	return s.model
}

func (s *OllamaEmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	body := map[string]any{
		"model":  s.model,
		"prompt": truncateText(text, 8000),
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("embed marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/api/embed", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("embed new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("embed read: %w", err)
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("embed: %d %s", res.StatusCode, string(raw))
	}

	var reply struct {
		Embeddings [][]float64 `json:"embeddings"`
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return nil, fmt.Errorf("embed decode: %w", err)
	}
	if len(reply.Embeddings) == 0 {
		return nil, fmt.Errorf("embed: empty response")
	}
	return reply.Embeddings[0], nil
}

func truncateText(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes])
	}
	return s
}
