package elasticsearch

import (
	"context"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8/typedapi/indices/create"
	"github.com/elastic/go-elasticsearch/v8/typedapi/indices/putmapping"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
)

const (
	IndexStories    = "storybuilder_stories"
	IndexScenes     = "storybuilder_scenes"
	IndexCharacters = "storybuilder_characters"
)

func (c *Client) EnsureIndices(ctx context.Context) error {
	defs := map[string]map[string]types.Property{
		IndexStories: {
			"id":          textWithKeyword(),
			"title":       textProperty(),
			"description": textProperty(),
			"theme":       textProperty(),
			"genre":       keywordProperty(),
			"status":      keywordProperty(),
			"createdAt":   dateProperty(),
			"updatedAt":   dateProperty(),
		},
		IndexScenes: {
			"id":               textWithKeyword(),
			"storyId":          keywordProperty(),
			"title":            textProperty(),
			"beatIntent":       textProperty(),
			"content":          textProperty(),
			"summary":          textProperty(),
			"status":           keywordProperty(),
			"pov":              keywordProperty(),
			"locationRef":      textProperty(),
			"participants":     keywordProperty(),
			"timelinePosition": intProperty(),
			"targetWords":      intProperty(),
			"createdAt":        dateProperty(),
			"updatedAt":        dateProperty(),
		},
		IndexCharacters: {
			"id":        textWithKeyword(),
			"storyId":   keywordProperty(),
			"charId":    keywordProperty(),
			"name":      textProperty(),
			"persona":   textProperty(),
			"backstory": textProperty(),
			"goals":     textProperty(),
			"flaws":     textProperty(),
			"arcType":   keywordProperty(),
			"createdAt": dateProperty(),
		},
	}

	for name, props := range defs {
		if err := c.createOrUpdateIndex(ctx, name, props); err != nil {
			return fmt.Errorf("ensure index %s: %w", name, err)
		}
	}
	return nil
}

func (c *Client) createOrUpdateIndex(ctx context.Context, name string, props map[string]types.Property) error {
	exists, err := c.raw.Indices.Exists(name).Do(ctx)
	if err != nil {
		return err
	}
	if exists {
		_, err := c.raw.Indices.PutMapping(name).
			Request(&putmapping.Request{Properties: props}).
			Do(ctx)
		return err
	}
	one := "1"
	_, err = c.raw.Indices.Create(name).
		Request(&create.Request{
			Settings: &types.IndexSettings{
				NumberOfShards:    &one,
				NumberOfReplicas: &one,
			},
			Mappings: &types.TypeMapping{Properties: props},
		}).Do(ctx)
	return err
}

func textProperty() types.Property {
	return &types.TextProperty{}
}

func textWithKeyword() types.Property {
	return &types.TextProperty{
		Fields: map[string]types.Property{
			"raw": &types.KeywordProperty{},
		},
	}
}

func keywordProperty() types.Property {
	return &types.KeywordProperty{}
}

func dateProperty() types.Property {
	return &types.DateProperty{}
}

func intProperty() types.Property {
	return &types.IntegerNumberProperty{}
}
