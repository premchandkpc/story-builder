package service

import (
	"context"
	"testing"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type mockChapterRepo struct {
	repository.ChapterRepository
	chapters   map[string]*domain.Chapter
	byStory    map[string][]*domain.Chapter
	byAct      map[string][]*domain.Chapter
	err        error
	nextID     int
}

func newMockChapterRepo() *mockChapterRepo {
	return &mockChapterRepo{
		chapters: make(map[string]*domain.Chapter),
		byStory:  make(map[string][]*domain.Chapter),
		byAct:    make(map[string][]*domain.Chapter),
		nextID:   1,
	}
}

func (m *mockChapterRepo) Create(ctx context.Context, c *domain.Chapter) error {
	if m.err != nil {
		return m.err
	}
	if c.ID == "" {
		c.ID = "ch-" + c.Title
	}
	c.CreatedAt = time.Now()
	m.chapters[c.ID] = c
	m.byStory[c.StoryID] = append(m.byStory[c.StoryID], c)
	actKey := c.StoryID + ":" + string(rune('0'+c.ActNumber))
	m.byAct[actKey] = append(m.byAct[actKey], c)
	return nil
}

func (m *mockChapterRepo) Get(ctx context.Context, id string) (*domain.Chapter, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.chapters[id], nil
}

func (m *mockChapterRepo) ListByStory(ctx context.Context, storyID string) ([]*domain.Chapter, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.byStory[storyID], nil
}

func (m *mockChapterRepo) ListByAct(ctx context.Context, storyID string, actNumber int) ([]*domain.Chapter, error) {
	if m.err != nil {
		return nil, m.err
	}
	actKey := storyID + ":" + string(rune('0'+actNumber))
	return m.byAct[actKey], nil
}

func (m *mockChapterRepo) Update(ctx context.Context, c *domain.Chapter) error {
	if m.err != nil {
		return m.err
	}
	m.chapters[c.ID] = c
	return nil
}

func (m *mockChapterRepo) DeleteByStory(ctx context.Context, storyID string) error {
	if m.err != nil {
		return m.err
	}
	for id, c := range m.chapters {
		if c.StoryID == storyID {
			delete(m.chapters, id)
		}
	}
	delete(m.byStory, storyID)
	return nil
}

func TestChapterService_Create(t *testing.T) {
	mock := newMockChapterRepo()
	svc := NewChapterSvc(mock)

	c, err := svc.Create(context.Background(), &domain.Chapter{
		StoryID:    "s1",
		ActNumber:  1,
		ChapterNum: 1,
		Title:      "Chapter One",
		Summary:    "The beginning",
		Goal:       "Setup the world",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Title != "Chapter One" {
		t.Fatalf("expected 'Chapter One', got %q", c.Title)
	}
	if c.Status != "" {
		t.Fatalf("expected empty status, got %q", c.Status)
	}
	if c.CreatedAt.IsZero() {
		t.Fatal("expected createdAt to be set")
	}
}

func TestChapterService_Get(t *testing.T) {
	mock := newMockChapterRepo()
	svc := NewChapterSvc(mock)

	c, _ := svc.Create(context.Background(), &domain.Chapter{
		StoryID: "s1", Title: "Get Test",
	})

	t.Run("existing chapter", func(t *testing.T) {
		got, err := svc.Get(context.Background(), c.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected chapter, got nil")
		}
		if got.Title != "Get Test" {
			t.Fatalf("expected 'Get Test', got %q", got.Title)
		}
	})

	t.Run("missing chapter", func(t *testing.T) {
		got, err := svc.Get(context.Background(), "nonexistent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Fatal("expected nil for missing chapter")
		}
	})
}

func TestChapterService_ListByStory(t *testing.T) {
	mock := newMockChapterRepo()
	svc := NewChapterSvc(mock)

	svc.Create(context.Background(), &domain.Chapter{StoryID: "s1", Title: "C1"})
	svc.Create(context.Background(), &domain.Chapter{StoryID: "s1", Title: "C2"})
	svc.Create(context.Background(), &domain.Chapter{StoryID: "s2", Title: "Other"})

	chapters, err := svc.ListByStory(context.Background(), "s1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(chapters))
	}

	chapters, err = svc.ListByStory(context.Background(), "empty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chapters) != 0 {
		t.Fatalf("expected 0 chapters, got %d", len(chapters))
	}
}

func TestChapterService_ListByAct(t *testing.T) {
	mock := newMockChapterRepo()
	svc := NewChapterSvc(mock)

	svc.Create(context.Background(), &domain.Chapter{StoryID: "s1", ActNumber: 1, Title: "A1C1"})
	svc.Create(context.Background(), &domain.Chapter{StoryID: "s1", ActNumber: 1, Title: "A1C2"})
	svc.Create(context.Background(), &domain.Chapter{StoryID: "s1", ActNumber: 2, Title: "A2C1"})

	chapters, err := svc.ListByAct(context.Background(), "s1", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters in act 1, got %d", len(chapters))
	}

	chapters, err = svc.ListByAct(context.Background(), "s1", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chapters) != 1 {
		t.Fatalf("expected 1 chapter in act 2, got %d", len(chapters))
	}
}

func TestChapterService_Update(t *testing.T) {
	mock := newMockChapterRepo()
	svc := NewChapterSvc(mock)

	c, _ := svc.Create(context.Background(), &domain.Chapter{
		StoryID: "s1", Title: "Original", Goal: "Old goal",
	})

	c.Title = "Updated"
	c.Goal = "New goal"
	updated, err := svc.Update(context.Background(), c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Title != "Updated" {
		t.Fatalf("expected 'Updated', got %q", updated.Title)
	}
	if updated.Goal != "New goal" {
		t.Fatalf("expected 'New goal', got %q", updated.Goal)
	}
	if updated.UpdatedAt.IsZero() {
		t.Fatal("expected updatedAt to be set")
	}
}
