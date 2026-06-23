package llm

import (
	"context"
	"sync"
)

type MockLLMClient struct {
	mu        sync.Mutex
	Responses map[string]string
	Calls     []string
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		Responses: make(map[string]string),
	}
}

func (m *MockLLMClient) SetResponse(key, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Responses[key] = content
}

func (m *MockLLMClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := string(req.Model) + "|" + req.UserMessage
	if len(req.UserMessage) > 80 {
		key = string(req.Model) + "|" + req.UserMessage[:80]
	}
	m.Calls = append(m.Calls, key)
	if resp, ok := m.Responses[key]; ok {
		return &CompletionResponse{Content: resp, Model: string(req.Model)}, nil
	}
	for pattern, resp := range m.Responses {
		if len(pattern) > 0 && len(pattern) <= len(key) && key[:len(pattern)] == pattern {
			m.Calls = append(m.Calls, "matched:"+pattern)
			return &CompletionResponse{Content: resp, Model: string(req.Model)}, nil
		}
	}
	return &CompletionResponse{Content: "{}", Model: string(req.Model)}, nil
}

type MockEmbeddingService struct{}

func (m *MockEmbeddingService) GenerateEmbedding(_ context.Context, _ string) ([]float64, error) {
	return make([]float64, 128), nil
}

func (m *MockEmbeddingService) Model() string { return "mock-embed" }
