package summary

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/compiler"
	"github.com/premchand/story-builder/internal/db"
)

// ── DB Summary Service ───────────────────────────────────────────

type DBSummaryService struct {
	q *db.Queries
}

func NewDBService(q *db.Queries) *DBSummaryService {
	return &DBSummaryService{q: q}
}

func (s *DBSummaryService) UpsertSceneSummary(ctx context.Context, storyID, sceneID uuid.UUID, content string) error {
	return s.q.UpsertSceneSummary(ctx, db.UpsertSceneSummaryParams{
		StoryID:   db.ToUUID(storyID),
		SceneID:   db.ToUUID(sceneID),
		Content:   content,
		WordCount: int32(len(strings.Fields(content))),
	})
}

func (s *DBSummaryService) UpsertActSummary(ctx context.Context, storyID uuid.UUID, content string) error {
	return s.q.UpsertActSummary(ctx, db.UpsertActSummaryParams{
		StoryID:   db.ToUUID(storyID),
		Content:   content,
		WordCount: int32(len(strings.Fields(content))),
	})
}

func (s *DBSummaryService) UpsertStorySummary(ctx context.Context, storyID uuid.UUID, content string) error {
	return s.q.UpsertStorySummary(ctx, db.UpsertStorySummaryParams{
		StoryID:   db.ToUUID(storyID),
		Content:   content,
		WordCount: int32(len(strings.Fields(content))),
	})
}

func (s *DBSummaryService) GetSceneSummary(ctx context.Context, storyID, sceneID uuid.UUID) (*compiler.StorySummary, error) {
	row, err := s.q.GetSceneSummary(ctx, db.GetSceneSummaryParams{
		StoryID: db.ToUUID(storyID),
		SceneID: db.ToUUID(sceneID),
	})
	if err != nil {
		return nil, err
	}
	return toDomainSummary(row), nil
}

func (s *DBSummaryService) GetSummaryByLevel(ctx context.Context, storyID uuid.UUID, level compiler.SummaryLevel) (*compiler.StorySummary, error) {
	row, err := s.q.GetSummaryByLevel(ctx, db.GetSummaryByLevelParams{
		StoryID: db.ToUUID(storyID),
		Level:   string(level),
	})
	if err != nil {
		return nil, err
	}
	return toDomainSummary(row), nil
}

func (s *DBSummaryService) ListSummariesByLevel(ctx context.Context, storyID uuid.UUID, level compiler.SummaryLevel) ([]compiler.StorySummary, error) {
	rows, err := s.q.ListSummariesByLevel(ctx, db.ListSummariesByLevelParams{
		StoryID: db.ToUUID(storyID),
		Level:   string(level),
	})
	if err != nil {
		return nil, err
	}
	result := make([]compiler.StorySummary, len(rows))
	for i, r := range rows {
		result[i] = *toDomainSummary(r)
	}
	return result, nil
}

func (s *DBSummaryService) CountSummariesByLevel(ctx context.Context, storyID uuid.UUID, level compiler.SummaryLevel) (int, error) {
	count, err := s.q.CountSummariesByLevel(ctx, db.CountSummariesByLevelParams{
		StoryID: db.ToUUID(storyID),
		Level:   string(level),
	})
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func (s *DBSummaryService) ShouldElevate(ctx context.Context, storyID uuid.UUID, level compiler.SummaryLevel, threshold int) (bool, error) {
	count, err := s.CountSummariesByLevel(ctx, storyID, level)
	if err != nil {
		return false, err
	}
	return count >= threshold, nil
}

// ── In-Memory Summary Service ────────────────────────────────────

type MemorySummaryService struct {
	scene      map[uuid.UUID]string    // nodeID -> content
	sceneStory map[uuid.UUID]uuid.UUID // nodeID -> storyID
	act        map[uuid.UUID]string    // storyID -> content
	story      map[uuid.UUID]string    // storyID -> content
}

func NewMemoryService() *MemorySummaryService {
	return &MemorySummaryService{
		scene:      make(map[uuid.UUID]string),
		sceneStory: make(map[uuid.UUID]uuid.UUID),
		act:        make(map[uuid.UUID]string),
		story:      make(map[uuid.UUID]string),
	}
}

func (s *MemorySummaryService) UpsertSceneSummary(ctx context.Context, storyID, nodeID uuid.UUID, content string) error {
	s.scene[nodeID] = content
	s.sceneStory[nodeID] = storyID
	return nil
}

func (s *MemorySummaryService) UpsertActSummary(ctx context.Context, storyID uuid.UUID, content string) error {
	s.act[storyID] = content
	return nil
}

func (s *MemorySummaryService) UpsertStorySummary(ctx context.Context, storyID uuid.UUID, content string) error {
	s.story[storyID] = content
	return nil
}

func (s *MemorySummaryService) GetSceneSummary(ctx context.Context, storyID, nodeID uuid.UUID) (*compiler.StorySummary, error) {
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

func (s *MemorySummaryService) GetSummaryByLevel(ctx context.Context, storyID uuid.UUID, level compiler.SummaryLevel) (*compiler.StorySummary, error) {
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

func (s *MemorySummaryService) ListSummariesByLevel(ctx context.Context, storyID uuid.UUID, level compiler.SummaryLevel) ([]compiler.StorySummary, error) {
	summary, err := s.GetSummaryByLevel(ctx, storyID, level)
	if err != nil {
		return nil, err
	}
	return []compiler.StorySummary{*summary}, nil
}

func (s *MemorySummaryService) CountSummariesByLevel(ctx context.Context, storyID uuid.UUID, level compiler.SummaryLevel) (int, error) {
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

func (s *MemorySummaryService) ShouldElevate(ctx context.Context, storyID uuid.UUID, level compiler.SummaryLevel, threshold int) (bool, error) {
	count, err := s.CountSummariesByLevel(ctx, storyID, level)
	if err != nil {
		return false, err
	}
	return count >= threshold, nil
}

// ── Helpers ──────────────────────────────────────────────────────

func toDomainSummary(row db.StorySummary) *compiler.StorySummary {
	var sceneID *uuid.UUID
	if row.SceneID.Valid {
		n := db.FromUUID(row.SceneID)
		sceneID = &n
	}
	return &compiler.StorySummary{
		ID:        db.FromUUID(row.ID),
		StoryID:   db.FromUUID(row.StoryID),
		NodeID:    sceneID,
		Level:     compiler.SummaryLevel(row.Level),
		Content:   row.Content,
		WordCount: int(row.WordCount),
		CreatedAt: row.CreatedAt.Time.Format(time.RFC3339),
	}
}
