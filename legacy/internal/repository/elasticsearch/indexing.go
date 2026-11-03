package elasticsearch

import (
	"context"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
)

func (c *Client) IndexStory(ctx context.Context, doc storyDoc) error {
	_, err := c.raw.Index(IndexStories).Id(doc.ID).Request(doc).Do(ctx)
	if err != nil {
		return fmt.Errorf("index story: %w", err)
	}
	return nil
}

func (c *Client) DeleteStory(ctx context.Context, storyID string) error {
	_, err := c.raw.Delete(IndexStories, storyID).Do(ctx)
	if err != nil {
		return fmt.Errorf("delete story index: %w", err)
	}
	return nil
}

func (c *Client) IndexScene(ctx context.Context, doc sceneDoc) error {
	_, err := c.raw.Index(IndexScenes).Id(doc.ID).Request(doc).Do(ctx)
	if err != nil {
		return fmt.Errorf("index scene: %w", err)
	}
	return nil
}

func (c *Client) DeleteScene(ctx context.Context, sceneID string) error {
	_, err := c.raw.Delete(IndexScenes, sceneID).Do(ctx)
	if err != nil {
		return fmt.Errorf("delete scene index: %w", err)
	}
	return nil
}

func (c *Client) DeleteScenesByStory(ctx context.Context, storyID string) error {
	_, err := c.raw.DeleteByQuery(IndexScenes).
		Query(&types.Query{
			Term: map[string]types.TermQuery{
				"storyId": {Value: storyID},
			},
		}).
		Do(ctx)
	if err != nil {
		return fmt.Errorf("delete scenes by story: %w", err)
	}
	return nil
}

func (c *Client) IndexCharacter(ctx context.Context, doc characterDoc) error {
	_, err := c.raw.Index(IndexCharacters).Id(doc.ID).Request(doc).Do(ctx)
	if err != nil {
		return fmt.Errorf("index character: %w", err)
	}
	return nil
}

func (c *Client) DeleteCharacter(ctx context.Context, charID string) error {
	_, err := c.raw.Delete(IndexCharacters, charID).Do(ctx)
	if err != nil {
		return fmt.Errorf("delete character index: %w", err)
	}
	return nil
}

func (c *Client) DeleteCharactersByStory(ctx context.Context, storyID string) error {
	_, err := c.raw.DeleteByQuery(IndexCharacters).
		Query(&types.Query{
			Term: map[string]types.TermQuery{
				"storyId": {Value: storyID},
			},
		}).
		Do(ctx)
	if err != nil {
		return fmt.Errorf("delete characters by story: %w", err)
	}
	return nil
}
