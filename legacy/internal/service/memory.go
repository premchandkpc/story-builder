package service

import (
	"context"
	"fmt"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/repository"
)

type MemoryService struct {
	repo     repository.MemoryRepository
	embedSvc llm.EmbeddingService
}

func NewMemoryService(repo repository.MemoryRepository, embedSvc llm.EmbeddingService) *MemoryService {
	return &MemoryService{repo: repo, embedSvc: embedSvc}
}

func (s *MemoryService) ListByCharacter(ctx context.Context, charID string) ([]*domain.CharacterMemory, error) {
	return s.repo.ListByCharacter(ctx, charID)
}

func (s *MemoryService) Search(ctx context.Context, storyID, characterID, query string, limit int) ([]*domain.CharacterMemory, error) {
	if s.embedSvc == nil {
		return nil, fmt.Errorf("embedding service not configured")
	}
	embedding, err := s.embedSvc.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}
	return s.repo.Search(ctx, storyID, characterID, embedding, limit)
}
