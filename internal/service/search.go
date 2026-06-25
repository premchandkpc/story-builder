package service

import (
	"context"
	"fmt"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type SearchService struct {
	repo repository.SearchRepository
}

func NewSearchService(repo repository.SearchRepository) *SearchService {
	return &SearchService{repo: repo}
}

func (s *SearchService) EnsureIndices(ctx context.Context) error {
	return s.repo.EnsureIndices(ctx)
}

func (s *SearchService) IndexStory(ctx context.Context, story *domain.Story) error {
	return s.repo.IndexStory(ctx, story)
}

func (s *SearchService) DeleteStoryIndex(ctx context.Context, storyID string) error {
	return s.repo.DeleteStoryIndex(ctx, storyID)
}

func (s *SearchService) IndexScene(ctx context.Context, scene *domain.Scene) error {
	return s.repo.IndexScene(ctx, scene)
}

func (s *SearchService) DeleteSceneIndex(ctx context.Context, sceneID string) error {
	return s.repo.DeleteSceneIndex(ctx, sceneID)
}

func (s *SearchService) IndexCharacter(ctx context.Context, char *domain.Character) error {
	return s.repo.IndexCharacter(ctx, char)
}

func (s *SearchService) DeleteCharacterIndex(ctx context.Context, charID string) error {
	return s.repo.DeleteCharacterIndex(ctx, charID)
}

func (s *SearchService) DeleteByStory(ctx context.Context, storyID string) error {
	return s.repo.DeleteByStory(ctx, storyID)
}

func (s *SearchService) Search(ctx context.Context, query, entityType, storyID string, limit, offset int) (*repository.SearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("search query required")
	}
	if entityType != "" {
		return s.repo.SearchByEntity(ctx, query, entityType, storyID, limit, offset)
	}
	return s.repo.Search(ctx, query, storyID, limit, offset)
}
