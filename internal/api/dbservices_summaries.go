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

func (s *dbSummaryService) UpsertSceneSummary(ctx context.Context, storyID, nodeID uuid.UUID, content string) error {
	return s.q.UpsertSceneSummary(ctx, db.UpsertSceneSummaryParams{
		StoryID:   toUUID(storyID),
		NodeID:    toUUID(nodeID),
		Content:   content,
		WordCount: int32(len(strings.Fields(content))),
	})
}

func (s *dbSummaryService) UpsertActSummary(ctx context.Context, storyID uuid.UUID, content string) error {
	return s.q.UpsertActSummary(ctx, db.UpsertActSummaryParams{
		StoryID:   toUUID(storyID),
		Content:   content,
		WordCount: int32(len(strings.Fields(content))),
	})
}

func (s *dbSummaryService) UpsertStorySummary(ctx context.Context, storyID uuid.UUID, content string) error {
	return s.q.UpsertStorySummary(ctx, db.UpsertStorySummaryParams{
		StoryID:   toUUID(storyID),
		Content:   content,
		WordCount: int32(len(strings.Fields(content))),
	})
}

func (s *dbSummaryService) GetSceneSummary(ctx context.Context, storyID, nodeID uuid.UUID) (*compiler.StorySummary, error) {
	row, err := s.q.GetSceneSummary(ctx, db.GetSceneSummaryParams{
		StoryID: toUUID(storyID),
		NodeID:  toUUID(nodeID),
	})
	if err != nil {
		return nil, err
	}
	return toDomainSummary(row), nil
}

func (s *dbSummaryService) GetSummaryByLevel(ctx context.Context, storyID uuid.UUID, level compiler.SummaryLevel) (*compiler.StorySummary, error) {
	row, err := s.q.GetSummaryByLevel(ctx, db.GetSummaryByLevelParams{
		StoryID: toUUID(storyID),
		Level:   string(level),
	})
	if err != nil {
		return nil, err
	}
	return toDomainSummary(row), nil
}

func (s *dbSummaryService) ListSummariesByLevel(ctx context.Context, storyID uuid.UUID, level compiler.SummaryLevel) ([]compiler.StorySummary, error) {
	rows, err := s.q.ListSummariesByLevel(ctx, db.ListSummariesByLevelParams{
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

func (s *dbSummaryService) CountSummariesByLevel(ctx context.Context, storyID uuid.UUID, level compiler.SummaryLevel) (int, error) {
	count, err := s.q.CountSummariesByLevel(ctx, db.CountSummariesByLevelParams{
		StoryID: toUUID(storyID),
		Level:   string(level),
	})
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func (s *dbSummaryService) ShouldElevate(ctx context.Context, storyID uuid.UUID, level compiler.SummaryLevel, threshold int) (bool, error) {
	count, err := s.CountSummariesByLevel(ctx, storyID, level)
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
