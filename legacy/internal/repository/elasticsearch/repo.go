package elasticsearch

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type SearchRepo struct {
	client *Client
}

func NewSearchRepo(client *Client) *SearchRepo {
	return &SearchRepo{client: client}
}

func (r *SearchRepo) EnsureIndices(ctx context.Context) error {
	return r.client.EnsureIndices(ctx)
}

func (r *SearchRepo) IndexStory(ctx context.Context, s *domain.Story) error {
	if s == nil {
		return nil
	}
	slog.Debug("indexing story", "id", s.ID, "title", s.Title)
	return r.client.IndexStory(ctx, toStoryDoc(s))
}

func (r *SearchRepo) DeleteStoryIndex(ctx context.Context, storyID string) error {
	slog.Debug("deleting story index", "id", storyID)
	return r.client.DeleteStory(ctx, storyID)
}

func (r *SearchRepo) IndexScene(ctx context.Context, s *domain.Scene) error {
	if s == nil {
		return nil
	}
	slog.Debug("indexing scene", "id", s.ID, "storyId", s.StoryID)
	return r.client.IndexScene(ctx, toSceneDoc(s))
}

func (r *SearchRepo) DeleteSceneIndex(ctx context.Context, sceneID string) error {
	slog.Debug("deleting scene index", "id", sceneID)
	return r.client.DeleteScene(ctx, sceneID)
}

func (r *SearchRepo) IndexCharacter(ctx context.Context, c *domain.Character) error {
	if c == nil {
		return nil
	}
	slog.Debug("indexing character", "id", c.ID, "name", c.Name)
	return r.client.IndexCharacter(ctx, toCharacterDoc(c))
}

func (r *SearchRepo) DeleteCharacterIndex(ctx context.Context, charID string) error {
	slog.Debug("deleting character index", "id", charID)
	return r.client.DeleteCharacter(ctx, charID)
}

func (r *SearchRepo) DeleteByStory(ctx context.Context, storyID string) error {
	var errs []error
	if err := r.client.DeleteScenesByStory(ctx, storyID); err != nil {
		errs = append(errs, fmt.Errorf("scenes: %w", err))
	}
	if err := r.client.DeleteCharactersByStory(ctx, storyID); err != nil {
		errs = append(errs, fmt.Errorf("characters: %w", err))
	}
	if err := r.client.DeleteStory(ctx, storyID); err != nil {
		errs = append(errs, fmt.Errorf("story: %w", err))
	}
	if len(errs) > 0 {
		return fmt.Errorf("delete by story: %v", errs)
	}
	return nil
}

func (r *SearchRepo) Search(ctx context.Context, query string, storyID string, limit, offset int) (*repository.SearchResult, error) {
	return r.client.crossEntitySearch(ctx, query, storyID, nil, limit, offset)
}

func (r *SearchRepo) SearchByEntity(ctx context.Context, query string, entityType string, storyID string, limit, offset int) (*repository.SearchResult, error) {
	index := entityTypeToIndex(entityType)
	return r.client.search(ctx, index, query, storyID, limit, offset)
}

func entityTypeToIndex(entity string) string {
	switch entity {
	case "story":
		return IndexStories
	case "scene":
		return IndexScenes
	case "character":
		return IndexCharacters
	default:
		return IndexStories
	}
}
