package api

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/compiler"
	"github.com/premchand/story-builder/internal/db"
)

func NewDBSummaryService(q *db.Queries) *dbSummaryService {
	return &dbSummaryService{q: q}
}

type dbSummaryService struct{ q *db.Queries }

func (s *dbSummaryService) UpsertSceneSummary(storyID, nodeID uuid.UUID, content string) error {
	return s.q.UpsertSceneSummary(context.Background(), db.UpsertSceneSummaryParams{
		StoryID:   toUUID(storyID),
		NodeID:    toUUID(nodeID),
		Content:   content,
		WordCount: int32(len(strings.Fields(content))),
	})
}

func (s *dbSummaryService) UpsertActSummary(storyID uuid.UUID, content string) error {
	return s.q.UpsertActSummary(context.Background(), db.UpsertActSummaryParams{
		StoryID:   toUUID(storyID),
		Content:   content,
		WordCount: int32(len(strings.Fields(content))),
	})
}

func (s *dbSummaryService) UpsertStorySummary(storyID uuid.UUID, content string) error {
	return s.q.UpsertStorySummary(context.Background(), db.UpsertStorySummaryParams{
		StoryID:   toUUID(storyID),
		Content:   content,
		WordCount: int32(len(strings.Fields(content))),
	})
}

func (s *dbSummaryService) GetSceneSummary(storyID, nodeID uuid.UUID) (*compiler.StorySummary, error) {
	row, err := s.q.GetSceneSummary(context.Background(), db.GetSceneSummaryParams{
		StoryID: toUUID(storyID),
		NodeID:  toUUID(nodeID),
	})
	if err != nil {
		return nil, err
	}
	return toDomainSummary(row), nil
}

func (s *dbSummaryService) GetSummaryByLevel(storyID uuid.UUID, level compiler.SummaryLevel) (*compiler.StorySummary, error) {
	row, err := s.q.GetSummaryByLevel(context.Background(), db.GetSummaryByLevelParams{
		StoryID: toUUID(storyID),
		Level:   string(level),
	})
	if err != nil {
		return nil, err
	}
	return toDomainSummary(row), nil
}

func (s *dbSummaryService) ListSummariesByLevel(storyID uuid.UUID, level compiler.SummaryLevel) ([]compiler.StorySummary, error) {
	rows, err := s.q.ListSummariesByLevel(context.Background(), db.ListSummariesByLevelParams{
		StoryID: toUUID(storyID),
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

func (s *dbSummaryService) CountSummariesByLevel(storyID uuid.UUID, level compiler.SummaryLevel) (int, error) {
	count, err := s.q.CountSummariesByLevel(context.Background(), db.CountSummariesByLevelParams{
		StoryID: toUUID(storyID),
		Level:   string(level),
	})
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func (s *dbSummaryService) ShouldElevate(storyID uuid.UUID, level compiler.SummaryLevel, threshold int) (bool, error) {
	count, err := s.CountSummariesByLevel(storyID, level)
	if err != nil {
		return false, err
	}
	return count >= threshold, nil
}

func toDomainSummary(row db.StorySummary) *compiler.StorySummary {
	var nodeID *uuid.UUID
	if row.NodeID.Valid {
		n := fromUUID(row.NodeID)
		nodeID = &n
	}
	return &compiler.StorySummary{
		ID:        fromUUID(row.ID),
		StoryID:   fromUUID(row.StoryID),
		NodeID:    nodeID,
		Level:     compiler.SummaryLevel(row.Level),
		Content:   row.Content,
		WordCount: int(row.WordCount),
		CreatedAt: row.CreatedAt.Time.Format(time.RFC3339),
	}
}
