package search

import (
	"github.com/google/uuid"
)

type SearchResult struct {
	ID          uuid.UUID `json:"id"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Score       float64   `json:"score"`
	StoryID     uuid.UUID `json:"story_id,omitempty"`
}

type SearchQuery struct {
	Query     string   `json:"query"`
	Types     []string `json:"types,omitempty"`
	StoryID   uuid.UUID `json:"story_id,omitempty"`
	Limit     int      `json:"limit"`
	Offset    int      `json:"offset"`
}

type SearchService interface {
	SearchStories(query SearchQuery) ([]SearchResult, error)
	SearchScenes(query SearchQuery) ([]SearchResult, error)
	SearchCharacters(query SearchQuery) ([]SearchResult, error)
	SearchMemories(query SearchQuery) ([]SearchResult, error)
	SearchAll(query SearchQuery) ([]SearchResult, error)
}

func (q SearchQuery) GetLimit() int {
	if q.Limit <= 0 {
		return 20
	}
	if q.Limit > 100 {
		return 100
	}
	return q.Limit
}
