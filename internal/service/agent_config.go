package service

import (
	"context"
	"fmt"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type AgentConfigService struct {
	repo repository.AgentConfigRepository
}

func NewAgentConfigService(repo repository.AgentConfigRepository) *AgentConfigService {
	return &AgentConfigService{repo: repo}
}

func (s *AgentConfigService) Create(ctx context.Context, cfg *domain.AgentConfig) error {
	return s.repo.Create(ctx, cfg)
}

func (s *AgentConfigService) Get(ctx context.Context, name string) (*domain.AgentConfig, error) {
	return s.repo.Get(ctx, name)
}

func (s *AgentConfigService) List(ctx context.Context) ([]*domain.AgentConfig, error) {
	return s.repo.List(ctx)
}

func (s *AgentConfigService) Export(ctx context.Context, name string) (*domain.AgentConfig, error) {
	cfg, err := s.repo.Get(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get agent config: %w", err)
	}
	if cfg == nil {
		return nil, fmt.Errorf("agent config %s not found: %w", name, ErrNotFound)
	}
	return cfg, nil
}

func (s *AgentConfigService) Import(ctx context.Context, cfg *domain.AgentConfig) error {
	cfg.CreatedAt = time.Now()
	cfg.UpdatedAt = time.Now()
	return s.repo.Create(ctx, cfg)
}

func (s *AgentConfigService) Delete(ctx context.Context, name string) error {
	return s.repo.Delete(ctx, name)
}

func (s *AgentConfigService) ListShared(ctx context.Context) ([]*domain.AgentConfig, error) {
	all, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	var shared []*domain.AgentConfig
	for _, c := range all {
		if c.Shared {
			shared = append(shared, c)
		}
	}
	return shared, nil
}
