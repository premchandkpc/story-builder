package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type mockStoryRepo struct {
	repository.StoryRepository
	stories   map[string]*domain.Story
	createErr error
	getErr    error
	listErr   error
	updateErr error
	deleteErr error
	nextID    int
}

func newMockStoryRepo() *mockStoryRepo {
	return &mockStoryRepo{stories: make(map[string]*domain.Story), nextID: 1}
}

func (m *mockStoryRepo) Create(ctx context.Context, s *domain.Story) error {
	if m.createErr != nil {
		return m.createErr
	}
	if s.ID == "" {
		s.ID = fmt.Sprintf("mock-id-%d", m.nextID)
		m.nextID++
	}
	m.stories[s.ID] = s
	return nil
}

func (m *mockStoryRepo) Get(ctx context.Context, id string) (*domain.Story, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	s, ok := m.stories[id]
	if !ok {
		return nil, nil
	}
	return s, nil
}

func (m *mockStoryRepo) List(ctx context.Context) ([]*domain.Story, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	result := make([]*domain.Story, 0, len(m.stories))
	for _, s := range m.stories {
		result = append(result, s)
	}
	return result, nil
}

func (m *mockStoryRepo) Update(ctx context.Context, s *domain.Story) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.stories[s.ID] = s
	return nil
}

func (m *mockStoryRepo) Delete(ctx context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.stories, id)
	return nil
}

func TestStoryService_Create(t *testing.T) {
	mock := newMockStoryRepo()
	svc := NewStoryService(mock)

	t.Run("creates story with title", func(t *testing.T) {
		s, err := svc.Create(context.Background(), "My Story")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Title != "My Story" {
			t.Fatalf("expected title 'My Story', got %q", s.Title)
		}
		if s.Status != domain.StoryStatusDraft {
			t.Fatalf("expected status draft, got %q", s.Status)
		}
		if s.ID == "" {
			t.Fatal("expected non-empty ID")
		}
	})

	t.Run("returns error when repo fails", func(t *testing.T) {
		mock.createErr = errors.New("db down")
		defer func() { mock.createErr = nil }()
		_, err := svc.Create(context.Background(), "fail")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestStoryService_Get(t *testing.T) {
	mock := newMockStoryRepo()
	svc := NewStoryService(mock)
	s, _ := svc.Create(context.Background(), "test")

	t.Run("returns existing story", func(t *testing.T) {
		got, err := svc.Get(context.Background(), s.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Title != "test" {
			t.Fatalf("expected title 'test', got %q", got.Title)
		}
	})

	t.Run("returns nil for missing story", func(t *testing.T) {
		got, err := svc.Get(context.Background(), "nonexistent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Fatal("expected nil for missing story")
		}
	})
}

func TestStoryService_Update(t *testing.T) {
	mock := newMockStoryRepo()
	svc := NewStoryService(mock)
	s, _ := svc.Create(context.Background(), "original")

	t.Run("updates title", func(t *testing.T) {
		updated, err := svc.Update(context.Background(), s.ID, UpdateStoryParams{Title: "updated"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.Title != "updated" {
			t.Fatalf("expected title 'updated', got %q", updated.Title)
		}
	})

	t.Run("errors on missing story", func(t *testing.T) {
		_, err := svc.Update(context.Background(), "nonexistent", UpdateStoryParams{Title: "x"})
		if err == nil {
			t.Fatal("expected error for missing story")
		}
	})
}

func TestStoryService_List(t *testing.T) {
	mock := newMockStoryRepo()
	svc := NewStoryService(mock)
	svc.Create(context.Background(), "a")
	svc.Create(context.Background(), "b")

	t.Run("lists all stories", func(t *testing.T) {
		stories, err := svc.List(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(stories) != 2 {
			t.Fatalf("expected 2 stories, got %d", len(stories))
		}
	})

	t.Run("returns empty list when no stories", func(t *testing.T) {
		emptyMock := newMockStoryRepo()
		emptySvc := NewStoryService(emptyMock)
		stories, err := emptySvc.List(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(stories) != 0 {
			t.Fatalf("expected 0 stories, got %d", len(stories))
		}
	})
}

func TestStoryService_Delete(t *testing.T) {
	mock := newMockStoryRepo()
	svc := NewStoryService(mock)
	s, _ := svc.Create(context.Background(), "test")

	t.Run("deletes existing story", func(t *testing.T) {
		err := svc.Delete(context.Background(), s.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, _ := svc.Get(context.Background(), s.ID)
		if got != nil {
			t.Fatal("story should be deleted")
		}
	})

	t.Run("no error on deleting nonexistent", func(t *testing.T) {
		err := svc.Delete(context.Background(), "nonexistent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestStoryService_Update_NotFound(t *testing.T) {
	mock := newMockStoryRepo()
	svc := NewStoryService(mock)

	_, err := svc.Update(context.Background(), "nonexistent", UpdateStoryParams{Title: "title"})
	if err == nil {
		t.Fatal("expected error for missing story")
	}
}
