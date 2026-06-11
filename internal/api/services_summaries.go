package api

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/compiler"
)

type memorySummaryService struct {
	scene      map[uuid.UUID]string       // nodeID -> content
	sceneStory map[uuid.UUID]uuid.UUID    // nodeID -> storyID
	act        map[uuid.UUID]string       // storyID -> content
	story      map[uuid.UUID]string       // storyID -> content
}

func NewMemorySummaryService() *memorySummaryService {
	return &memorySummaryService{
		scene:      make(map[uuid.UUID]string),
		sceneStory: make(map[uuid.UUID]uuid.UUID),
		act:        make(map[uuid.UUID]string),
		story:      make(map[uuid.UUID]string),
	}
}

func (s *memorySummaryService) UpsertSceneSummary(storyID, nodeID uuid.UUID, content string) error {
	s.scene[nodeID] = content
	s.sceneStory[nodeID] = storyID
	return nil
}

func (s *memorySummaryService) UpsertActSummary(storyID uuid.UUID, content string) error {
	s.act[storyID] = content
	return nil
}

func (s *memorySummaryService) UpsertStorySummary(storyID uuid.UUID, content string) error {
	s.story[storyID] = content
	return nil
}

func (s *memorySummaryService) GetSceneSummary(storyID, nodeID uuid.UUID) (*compiler.StorySummary, error) {
	content, ok := s.scene[nodeID]
	if !ok {
		return nil, fmt.Errorf("scene summary not found for node %s", nodeID)
	}
	return &compiler.StorySummary{
		ID:        uuid.New(),
		StoryID:   storyID,
		NodeID:    &nodeID,
		Level:     compiler.SummaryScene,
		Content:   content,
		WordCount: len(content),
	}, nil
}

func (s *memorySummaryService) GetSummaryByLevel(storyID uuid.UUID, level compiler.SummaryLevel) (*compiler.StorySummary, error) {
	switch level {
	case compiler.SummaryAct:
		content, ok := s.act[storyID]
		if !ok {
			return nil, fmt.Errorf("act summary not found for story %s", storyID)
		}
		return &compiler.StorySummary{
			ID:        uuid.New(),
			StoryID:   storyID,
			Level:     compiler.SummaryAct,
			Content:   content,
			WordCount: len(content),
		}, nil
	case compiler.SummaryStory:
		content, ok := s.story[storyID]
		if !ok {
			return nil, fmt.Errorf("story summary not found for story %s", storyID)
		}
		return &compiler.StorySummary{
			ID:        uuid.New(),
			StoryID:   storyID,
			Level:     compiler.SummaryStory,
			Content:   content,
			WordCount: len(content),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported level %s", level)
	}
}

func (s *memorySummaryService) ListSummariesByLevel(storyID uuid.UUID, level compiler.SummaryLevel) ([]compiler.StorySummary, error) {
	summary, err := s.GetSummaryByLevel(storyID, level)
	if err != nil {
		return nil, err
	}
	return []compiler.StorySummary{*summary}, nil
}

func (s *memorySummaryService) CountSummariesByLevel(storyID uuid.UUID, level compiler.SummaryLevel) (int, error) {
	if level == compiler.SummaryScene {
		count := 0
		for nodeID, v := range s.scene {
			if v != "" && s.sceneStory[nodeID] == storyID {
				count++
			}
		}
		return count, nil
	}
	return 0, nil
}

func (s *memorySummaryService) ShouldElevate(storyID uuid.UUID, level compiler.SummaryLevel, threshold int) (bool, error) {
	count, err := s.CountSummariesByLevel(storyID, level)
	if err != nil {
		return false, err
	}
	return count >= threshold, nil
}
