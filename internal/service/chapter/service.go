package chapter

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/premchand/story-builder/internal/db"
	"github.com/premchand/story-builder/internal/graph"
)

type Service interface {
	Create(ctx context.Context, storyID uuid.UUID, title string, orderIndex int) (*graph.Chapter, error)
	Get(ctx context.Context, id uuid.UUID) (*graph.Chapter, error)
	Update(ctx context.Context, id uuid.UUID, title, goal, summary, status string, orderIndex int) (*graph.Chapter, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, storyID uuid.UUID) ([]graph.Chapter, error)
}

// ── DB-backed ──────────────────────────────────────────────────────────

type dbService struct {
	q *db.Queries
}

func NewDBService(q *db.Queries) *dbService {
	return &dbService{q: q}
}

// ── In-Memory ──────────────────────────────────────────────────────────

type memoryService struct {
	store *graph.MemoryStore
}

func NewMemoryService(store *graph.MemoryStore) *memoryService {
	return &memoryService{store: store}
}

func (s *memoryService) Create(ctx context.Context, storyID uuid.UUID, title string, orderIndex int) (*graph.Chapter, error) {
	return s.store.CreateChapter(storyID, title, orderIndex)
}

func (s *memoryService) Get(ctx context.Context, id uuid.UUID) (*graph.Chapter, error) {
	return s.store.GetChapter(id)
}

func (s *memoryService) Update(ctx context.Context, id uuid.UUID, title, goal, summary, status string, orderIndex int) (*graph.Chapter, error) {
	ch, err := s.store.GetChapter(id)
	if err != nil {
		return nil, err
	}
	ch.Title = title
	ch.Goal = goal
	ch.Summary = summary
	ch.Status = graph.ChapterStatus(status)
	ch.OrderIndex = orderIndex
	return ch, nil
}

func (s *memoryService) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (s *memoryService) List(ctx context.Context, storyID uuid.UUID) ([]graph.Chapter, error) {
	chapters, err := s.store.ListChapters(storyID)
	if err != nil {
		return nil, err
	}
	sort.Slice(chapters, func(i, j int) bool {
		return chapters[i].OrderIndex < chapters[j].OrderIndex
	})
	return chapters, nil
}

func (s *dbService) Create(ctx context.Context, storyID uuid.UUID, title string, orderIndex int) (*graph.Chapter, error) {
	ch, err := s.q.CreateChapter(ctx, db.CreateChapterParams{
		StoryID:    db.ToUUID(storyID),
		Title:      title,
		Goal:       "",
		OrderIndex: int32(orderIndex),
	})
	if err != nil {
		return nil, err
	}
	return toDomainChapter(ch), nil
}

func (s *dbService) Get(ctx context.Context, id uuid.UUID) (*graph.Chapter, error) {
	ch, err := s.q.GetChapter(ctx, db.ToUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("chapter %s not found", id)
		}
		return nil, err
	}
	return toDomainChapter(ch), nil
}

func (s *dbService) Update(ctx context.Context, id uuid.UUID, title, goal, summary, status string, orderIndex int) (*graph.Chapter, error) {
	ch, err := s.q.UpdateChapter(ctx, db.UpdateChapterParams{
		ID:         db.ToUUID(id),
		Title:      title,
		Goal:       goal,
		OrderIndex: int32(orderIndex),
		Summary:    summary,
		Status:     status,
	})
	if err != nil {
		return nil, err
	}
	return toDomainChapter(ch), nil
}

func (s *dbService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.q.DeleteChapter(ctx, db.ToUUID(id))
}

func (s *dbService) List(ctx context.Context, storyID uuid.UUID) ([]graph.Chapter, error) {
	chapters, err := s.q.ListChapters(ctx, db.ToUUID(storyID))
	if err != nil {
		return nil, err
	}
	result := make([]graph.Chapter, len(chapters))
	for i, ch := range chapters {
		result[i] = *toDomainChapter(ch)
	}
	return result, nil
}

func toDomainChapter(ch db.Chapter) *graph.Chapter {
	return &graph.Chapter{
		ID:         db.FromUUID(ch.ID),
		StoryID:    db.FromUUID(ch.StoryID),
		Title:      ch.Title,
		Goal:       ch.Goal,
		OrderIndex: int(ch.OrderIndex),
		Summary:    ch.Summary,
		Status:     graph.ChapterStatus(ch.Status),
		CreatedAt:  ch.CreatedAt.Time,
		UpdatedAt:  ch.UpdatedAt.Time,
	}
}
