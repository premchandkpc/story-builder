package service

import (
	"context"
	"math"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

var modelCosts = map[string]struct {
	input  float64
	output float64
}{
	"claude-sonnet":  {input: 0.015, output: 0.075},
	"claude-haiku":   {input: 0.0025, output: 0.0125},
	"local-7b":       {input: 0.0005, output: 0.0010},
	"claude-opus":    {input: 0.075, output: 0.300},
}

type MetricsService struct {
	genRepo repository.GenerationRepository
}

func NewMetricsService(genRepo repository.GenerationRepository) *MetricsService {
	return &MetricsService{genRepo: genRepo}
}

func (s *MetricsService) GetLlmMetrics(ctx context.Context, storyID string) (*domain.LlmMetrics, error) {
	gens, err := s.genRepo.ListByStory(ctx, storyID)
	if err != nil {
		return nil, err
	}

	m := &domain.LlmMetrics{
		ByModel: make(map[string]domain.ModelTokenUsage),
		ByAgent: make(map[string]domain.AgentTokenUsage),
	}

	for _, gen := range gens {
		m.TotalPromptTokens += gen.PromptTokens
		m.TotalCompletionTokens += gen.CompletionTokens
		m.TotalTokens += gen.TotalTokens
		m.GenerationCount++

		model := gen.Model
		if model == "" {
			model = "unknown"
		}

		mu := m.ByModel[model]
		mu.PromptTokens += gen.PromptTokens
		mu.CompletionTokens += gen.CompletionTokens
		if costs, ok := modelCosts[model]; ok {
			mu.Cost += float64(gen.PromptTokens)/1000*costs.input +
				float64(gen.CompletionTokens)/1000*costs.output
		}
		m.ByModel[model] = mu

		for agent, status := range gen.StepStatus {
			if status == "" {
				continue
			}
			au := m.ByAgent[agent]
			au.TurnCount++
			au.PromptTokens += gen.PromptTokens / max(len(gen.StepStatus), 1)
			au.CompletionTokens += gen.CompletionTokens / max(len(gen.StepStatus), 1)
			m.ByAgent[agent] = au
		}
	}

	m.TotalCostEstimate = 0
	for _, mu := range m.ByModel {
		m.TotalCostEstimate += mu.Cost
	}
	m.TotalCostEstimate = math.Round(m.TotalCostEstimate*10000) / 10000

	return m, nil
}
