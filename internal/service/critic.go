package service

import (
	"context"
	"fmt"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type CriticScoresService struct {
	genRepo   repository.GenerationRepository
	sceneRepo repository.SceneRepository
}

func NewCriticScoresService(genRepo repository.GenerationRepository, sceneRepo repository.SceneRepository) *CriticScoresService {
	return &CriticScoresService{genRepo: genRepo, sceneRepo: sceneRepo}
}

func (s *CriticScoresService) ListByStory(ctx context.Context, storyID string) ([]domain.CriticScoreEntry, error) {
	gens, err := s.genRepo.ListByStory(ctx, storyID)
	if err != nil {
		return nil, fmt.Errorf("list generations: %w", err)
	}
	var out []domain.CriticScoreEntry
	for _, g := range gens {
		if g.CriticScore == 0 {
			continue
		}
		out = append(out, domain.CriticScoreEntry{
			GenerationID: g.ID,
			SceneID:      g.SceneID,
			Score:        g.CriticScore,
			Summary:      g.CriticSummary,
			CreatedAt:    g.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	if out == nil {
		out = []domain.CriticScoreEntry{}
	}
	return out, nil
}
