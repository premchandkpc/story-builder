package service

import (
	"context"
	"errors"
	"testing"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type mockBibleRepo struct {
	repository.BibleRepository
	bibles  map[string]*domain.StoryBible
	byStory map[string]*domain.StoryBible
	err     error
}

func newMockBibleRepo() *mockBibleRepo {
	return &mockBibleRepo{
		bibles:  make(map[string]*domain.StoryBible),
		byStory: make(map[string]*domain.StoryBible),
	}
}

func (m *mockBibleRepo) Create(ctx context.Context, b *domain.StoryBible) error {
	if m.err != nil {
		return m.err
	}
	m.bibles[b.ID] = b
	m.byStory[b.StoryID] = b
	return nil
}

func (m *mockBibleRepo) Get(ctx context.Context, id string) (*domain.StoryBible, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.bibles[id], nil
}

func (m *mockBibleRepo) GetByStory(ctx context.Context, storyID string) (*domain.StoryBible, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.byStory[storyID], nil
}

func (m *mockBibleRepo) Update(ctx context.Context, b *domain.StoryBible) error {
	if m.err != nil {
		return m.err
	}
	m.bibles[b.ID] = b
	m.byStory[b.StoryID] = b
	return nil
}

func (m *mockBibleRepo) DeleteByStory(ctx context.Context, storyID string) error {
	if m.err != nil {
		return m.err
	}
	for id, b := range m.bibles {
		if b.StoryID == storyID {
			delete(m.bibles, id)
		}
	}
	delete(m.byStory, storyID)
	return nil
}

type mockBibleGenSvc struct {
	bibles map[string]*domain.StoryBible
	err    error
}

func (m *mockBibleGenSvc) GenerateBible(ctx context.Context, storyID, synopsis string, characters []*domain.Character) (*domain.StoryBible, error) {
	if m.err != nil {
		return nil, m.err
	}
	b := &domain.StoryBible{
		ID:      "bible-" + storyID,
		StoryID: storyID,
		World:   "A world built from: " + synopsis,
		WorldRules: []domain.WorldRule{
			{Category: "physics", Description: "Magic exists", Strictness: "firm"},
		},
	}
	m.bibles[storyID] = b
	return b, nil
}

func TestBibleService_Get(t *testing.T) {
	bibleRepo := newMockBibleRepo()
	storyRepo := newMockStoryRepo()
	charRepo := newMockCharacterRepo()
	genSvc := &mockBibleGenSvc{bibles: make(map[string]*domain.StoryBible)}
	svc := NewBibleService(bibleRepo, storyRepo, charRepo, genSvc)

	existing := &domain.StoryBible{ID: "b1", StoryID: "s1", World: "test world"}
	bibleRepo.Create(context.Background(), existing)

	t.Run("existing bible", func(t *testing.T) {
		got, err := svc.Get(context.Background(), "s1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.World != "test world" {
			t.Fatalf("expected 'test world', got %q", got.World)
		}
	})

	t.Run("missing bible", func(t *testing.T) {
		got, err := svc.Get(context.Background(), "nonexistent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Fatal("expected nil for missing bible")
		}
	})
}

func TestBibleService_Generate(t *testing.T) {
	bibleRepo := newMockBibleRepo()
	storyRepo := newMockStoryRepo()
	charRepo := newMockCharacterRepo()
	genSvc := &mockBibleGenSvc{bibles: make(map[string]*domain.StoryBible)}
	svc := NewBibleService(bibleRepo, storyRepo, charRepo, genSvc)

	t.Run("successful generation", func(t *testing.T) {
		storyRepo.Create(context.Background(), &domain.Story{Title: "Test Story", MainPrompt: "A fantasy world"})

		b, err := svc.Generate(context.Background(), "mock-id-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.StoryID != "mock-id-1" {
			t.Fatalf("expected storyID 'mock-id-1', got %q", b.StoryID)
		}
		if len(b.WorldRules) == 0 {
			t.Fatal("expected world rules to be set")
		}
		if b.CreatedAt.IsZero() {
			t.Fatal("expected createdAt to be set")
		}
	})

	t.Run("single-flight guard prevents concurrent generation", func(t *testing.T) {
		storyRepo.Create(context.Background(), &domain.Story{Title: "Test Story 2", MainPrompt: "Another world"})
		svc.genInFlight.Store("mock-id-2", true)
		defer svc.genInFlight.Delete("mock-id-2")

		_, err := svc.Generate(context.Background(), "mock-id-2")
		if err == nil {
			t.Fatal("expected error for in-flight generation")
		}
	})

	t.Run("story not found", func(t *testing.T) {
		_, err := svc.Generate(context.Background(), "nonexistent")
		if err == nil {
			t.Fatal("expected error for missing story")
		}
	})
}

func TestBibleService_Update(t *testing.T) {
	bibleRepo := newMockBibleRepo()
	storyRepo := newMockStoryRepo()
	charRepo := newMockCharacterRepo()
	genSvc := &mockBibleGenSvc{bibles: make(map[string]*domain.StoryBible)}
	svc := NewBibleService(bibleRepo, storyRepo, charRepo, genSvc)

	bibleRepo.Create(context.Background(), &domain.StoryBible{ID: "b1", StoryID: "s1", World: "old"})

	err := svc.Update(context.Background(), &domain.StoryBible{ID: "b1", StoryID: "s1", World: "updated"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := bibleRepo.Get(context.Background(), "b1")
	if got.World != "updated" {
		t.Fatalf("expected 'updated', got %q", got.World)
	}
}

func TestBibleService_DeleteByStory(t *testing.T) {
	bibleRepo := newMockBibleRepo()
	storyRepo := newMockStoryRepo()
	charRepo := newMockCharacterRepo()
	genSvc := &mockBibleGenSvc{bibles: make(map[string]*domain.StoryBible)}
	svc := NewBibleService(bibleRepo, storyRepo, charRepo, genSvc)

	bibleRepo.Create(context.Background(), &domain.StoryBible{ID: "b1", StoryID: "s1"})

	err := svc.DeleteByStory(context.Background(), "s1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := bibleRepo.GetByStory(context.Background(), "s1")
	if got != nil {
		t.Fatal("expected bible to be deleted")
	}
}

func TestBibleService_Generate_UsesSynopsis(t *testing.T) {
	bibleRepo := newMockBibleRepo()
	storyRepo := newMockStoryRepo()
	charRepo := newMockCharacterRepo()
	genSvc := &mockBibleGenSvc{bibles: make(map[string]*domain.StoryBible)}
	svc := NewBibleService(bibleRepo, storyRepo, charRepo, genSvc)

	t.Run("uses MainPrompt as synopsis", func(t *testing.T) {
		storyRepo.Create(context.Background(), &domain.Story{
			Title: "S", MainPrompt: "Primary synopsis", GeneralPrompt: "Fallback",
		})
		b, err := svc.Generate(context.Background(), "mock-id-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.World != "A world built from: Primary synopsis" {
			t.Fatalf("expected MainPrompt synopsis, got %q", b.World)
		}
	})

	t.Run("falls back to GeneralPrompt when MainPrompt empty", func(t *testing.T) {
		storyRepo.Create(context.Background(), &domain.Story{
			Title: "S", GeneralPrompt: "Fallback synopsis",
		})
		b, err := svc.Generate(context.Background(), "mock-id-2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.World != "A world built from: Fallback synopsis" {
			t.Fatalf("expected GeneralPrompt synopsis, got %q", b.World)
		}
	})

	t.Run("falls back to Title when both empty", func(t *testing.T) {
		storyRepo.Create(context.Background(), &domain.Story{Title: "Title Only"})
		b, err := svc.Generate(context.Background(), "mock-id-3")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.World != "A world built from: Title Only" {
			t.Fatalf("expected Title synopsis, got %q", b.World)
		}
	})
}

func TestBibleService_Generate_RepoError(t *testing.T) {
	bibleRepo := newMockBibleRepo()
	storyRepo := newMockStoryRepo()
	charRepo := newMockCharacterRepo()
	genSvc := &mockBibleGenSvc{bibles: make(map[string]*domain.StoryBible)}
	svc := NewBibleService(bibleRepo, storyRepo, charRepo, genSvc)

	storyRepo.createErr = errors.New("db error")
	_, err := svc.Generate(context.Background(), "story-1")
	if err == nil {
		t.Fatal("expected error from story repo")
	}
}
