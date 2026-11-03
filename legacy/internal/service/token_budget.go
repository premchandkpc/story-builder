package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type TokenBudgetService struct {
	repo repository.TokenBudgetRepository
}

func NewTokenBudgetService(repo repository.TokenBudgetRepository) *TokenBudgetService {
	return &TokenBudgetService{repo: repo}
}

var defaultBudgetLimits = map[string]int{
	"claude-sonnet":      100000,
	"claude-haiku":       500000,
	"local-7b":           1000000,
}

func (s *TokenBudgetService) CheckAndConsume(ctx context.Context, storyID, model, agentType string, promptTokens, completionTokens int) error {
	totalTokens := promptTokens + completionTokens

	tb, err := s.repo.Get(ctx, storyID)
	if err != nil {
		return fmt.Errorf("get token budget: %w", err)
	}

	if tb == nil {
		limit, ok := defaultBudgetLimits[model]
		if !ok {
			limit = 100000
		}
		tb = &domain.TokenBudget{
			StoryID:    storyID,
			Model:      model,
			AgentType:  agentType,
			BudgetLimit: limit,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
	}

	if tb.BudgetUsed+totalTokens > tb.BudgetLimit {
		return fmt.Errorf("token budget exceeded: %d/%d for story %s model %s",
			tb.BudgetUsed, tb.BudgetLimit, storyID, model)
	}

	tb.BudgetUsed += totalTokens
	tb.PromptTokens += promptTokens
	tb.CompletionTokens += completionTokens
	tb.TotalTokens += totalTokens
	tb.TurnCount++
	tb.UpdatedAt = time.Now()

	if err := s.repo.Upsert(ctx, tb); err != nil {
		return fmt.Errorf("persist token budget: %w", err)
	}

	usage := float64(tb.BudgetUsed) / float64(tb.BudgetLimit) * 100
	if usage >= 80 {
		slog.Warn("token budget near limit", "storyId", storyID, "usagePercent", usage, "used", tb.BudgetUsed, "limit", tb.BudgetLimit)
	}

	return nil
}

func (s *TokenBudgetService) GetUsage(ctx context.Context, storyID string) (*domain.TokenBudget, error) {
	return s.repo.Get(ctx, storyID)
}
