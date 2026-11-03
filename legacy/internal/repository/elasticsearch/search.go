package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/textquerytype"
	"github.com/premchand/story-builder/internal/repository"
)

func (c *Client) search(ctx context.Context, index string, query, storyID string, limit, offset int) (*repository.SearchResult, error) {
	esQuery := buildESQuery(query, storyID)
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	res, err := c.raw.Search().
		Index(index).
		Query(esQuery).
		From(offset).
		Size(limit).
		Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("search %s: %w", index, err)
	}

	entity := entityFromIndex(index)
	result := &repository.SearchResult{
		Total: res.Hits.Total.Value,
	}

	for _, hit := range res.Hits.Hits {
		r := repository.SearchHit{
			Entity: entity,
			Data:   hit.Source_,
		}
		if hit.Id_ != nil {
			r.ID = *hit.Id_
		}
		if hit.Score_ != nil {
			r.Score = float64(*hit.Score_)
		}

		switch entity {
		case "story":
			var doc storyDoc
			if err := json.Unmarshal(hit.Source_, &doc); err == nil {
				r.Title = doc.Title
				if len(doc.Description) > 200 {
					r.Excerpt = doc.Description[:200]
				} else {
					r.Excerpt = doc.Description
				}
			}
		case "scene":
			var doc sceneDoc
			if err := json.Unmarshal(hit.Source_, &doc); err == nil {
				r.Title = doc.Title
				excerpt := doc.BeatIntent
				if excerpt == "" {
					excerpt = doc.Content
				}
				if len(excerpt) > 200 {
					r.Excerpt = excerpt[:200]
				} else {
					r.Excerpt = excerpt
				}
			}
		case "character":
			var doc characterDoc
			if err := json.Unmarshal(hit.Source_, &doc); err == nil {
				r.Title = doc.Name
				if len(doc.Persona) > 200 {
					r.Excerpt = doc.Persona[:200]
				} else if len(doc.Backstory) > 200 {
					r.Excerpt = doc.Backstory[:200]
				} else {
					r.Excerpt = doc.Persona
				}
			}
		}
		result.Hits = append(result.Hits, r)
	}

	return result, nil
}

func (c *Client) crossEntitySearch(ctx context.Context, query, storyID string, entityTypes []string, limit, offset int) (*repository.SearchResult, error) {
	indices := entityTypes
	if len(indices) == 0 {
		indices = []string{IndexStories, IndexScenes, IndexCharacters}
	}

	esQuery := buildESQuery(query, storyID)
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	res, err := c.raw.Search().
		Index(strings.Join(indices, ",")).
		Query(esQuery).
		From(offset).
		Size(limit).
		Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("cross-entity search: %w", err)
	}

	result := &repository.SearchResult{
		Total: res.Hits.Total.Value,
	}

	for _, hit := range res.Hits.Hits {
		entity := entityFromIndex(hit.Index_)
		r := repository.SearchHit{
			Entity: entity,
			Data:   hit.Source_,
		}
		if hit.Id_ != nil {
			r.ID = *hit.Id_
		}
		if hit.Score_ != nil {
			r.Score = float64(*hit.Score_)
		}

		switch entity {
		case "story":
			var doc storyDoc
			if err := json.Unmarshal(hit.Source_, &doc); err == nil {
				r.Title = doc.Title
				if len(doc.Description) > 200 {
					r.Excerpt = doc.Description[:200]
				} else {
					r.Excerpt = doc.Description
				}
			}
		case "scene":
			var doc sceneDoc
			if err := json.Unmarshal(hit.Source_, &doc); err == nil {
				r.Title = doc.Title
				excerpt := doc.BeatIntent
				if excerpt == "" {
					excerpt = doc.Content
				}
				if len(excerpt) > 200 {
					r.Excerpt = excerpt[:200]
				} else {
					r.Excerpt = excerpt
				}
			}
		case "character":
			var doc characterDoc
			if err := json.Unmarshal(hit.Source_, &doc); err == nil {
				r.Title = doc.Name
				if len(doc.Persona) > 200 {
					r.Excerpt = doc.Persona[:200]
				} else if len(doc.Backstory) > 200 {
					r.Excerpt = doc.Backstory[:200]
				} else {
					r.Excerpt = doc.Persona
				}
			}
		}

		result.Hits = append(result.Hits, r)
	}

	return result, nil
}

func buildESQuery(q, storyID string) *types.Query {
	if q == "" && storyID == "" {
		return &types.Query{MatchAll: &types.MatchAllQuery{}}
	}

	boolQ := &types.BoolQuery{}

	if q != "" {
		bestFields := textquerytype.Bestfields
		boolQ.Must = []types.Query{
			{
				MultiMatch: &types.MultiMatchQuery{
					Query:  q,
					Fields: []string{"*"},
					Type:   &bestFields,
				},
			},
		}
	}

	if storyID != "" {
		boolQ.Filter = []types.Query{
			{Term: map[string]types.TermQuery{"storyId": {Value: storyID}}},
		}
	}

	return &types.Query{Bool: boolQ}
}

func entityFromIndex(index string) string {
	switch index {
	case IndexStories:
		return "story"
	case IndexScenes:
		return "scene"
	case IndexCharacters:
		return "character"
	default:
		parts := strings.Split(index, "_")
		if len(parts) > 1 {
			return parts[1]
		}
		return "unknown"
	}
}
