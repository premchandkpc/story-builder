package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type StoryCascadeDeleter struct {
	SceneRepo   repository.SceneRepository
	EdgeRepo    repository.SceneEdgeRepository
	CharRepo    repository.CharacterRepository
	StateRepo   repository.CharacterStateRepository
	GenRepo     repository.GenerationRepository
	MemRepo     repository.MemoryRepository
	TlRepo      repository.TimelineRepository
	SumRepo     repository.SummaryRepository
	LocRepo     repository.LocationRepository
	BibleRepo   repository.BibleRepository
	ChapterRepo repository.ChapterRepository
}

type StoryService struct {
	repo    repository.StoryRepository
	deleter *StoryCascadeDeleter
}

func NewStoryService(repo repository.StoryRepository, deleter *StoryCascadeDeleter) *StoryService {
	return &StoryService{repo: repo, deleter: deleter}
}

func (s *StoryService) Create(ctx context.Context, title string) (*domain.Story, error) {
	st := &domain.Story{
		Title:     title,
		Status:    domain.StoryStatusDraft,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.repo.Create(ctx, st); err != nil {
		return nil, fmt.Errorf("create story: %w", err)
	}
	return st, nil
}

func (s *StoryService) Get(ctx context.Context, id string) (*domain.Story, error) {
	return s.repo.Get(ctx, id)
}

type UpdateStoryParams struct {
	Title  string
	Status string
}

func (s *StoryService) Update(ctx context.Context, id string, params UpdateStoryParams) (*domain.Story, error) {
	st, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get story for update: %w", err)
	}
	if st == nil {
		return nil, fmt.Errorf("story not found: %w", ErrNotFound)
	}
	if params.Title != "" {
		st.Title = params.Title
	}
	if params.Status != "" {
		if err := st.CanTransitionTo(params.Status); err != nil {
			return nil, err
		}
		st.Status = params.Status
	}
	st.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, st); err != nil {
		return nil, err
	}
	return st, nil
}

func (s *StoryService) List(ctx context.Context) ([]*domain.Story, error) {
	return s.repo.List(ctx)
}

func (s *StoryService) GetBlueprint(ctx context.Context, id string) (*domain.StoryBlueprint, error) {
	st, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, fmt.Errorf("story not found: %w", ErrNotFound)
	}
	return st.Blueprint, nil
}

func (s *StoryService) UpdateBlueprint(ctx context.Context, id string, bp *domain.StoryBlueprint) error {
	st, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if st == nil {
		return fmt.Errorf("story not found: %w", ErrNotFound)
	}
	st.Blueprint = bp
	st.UpdatedAt = time.Now()
	return s.repo.Update(ctx, st)
}

func (s *StoryService) Delete(ctx context.Context, id string) error {
	if s.deleter != nil {
		if err := s.deleter.cascade(ctx, id); err != nil {
			return err
		}
	}
	return s.repo.Delete(ctx, id)
}

func (d *StoryCascadeDeleter) cascade(ctx context.Context, storyID string) error {
	type step struct {
		name string
		fn   func(context.Context, string) error
	}
	steps := []step{
		{"character_state", d.StateRepo.DeleteByStory},
		{"character_memories", d.MemRepo.DeleteByStory},
		{"generations", d.GenRepo.DeleteByStory},
		{"summaries", d.SumRepo.DeleteByStory},
		{"timeline_events", d.TlRepo.DeleteByStory},
		{"scene_edges", d.EdgeRepo.DeleteByStory},
		{"scenes", d.SceneRepo.DeleteByStory},
		{"characters", d.CharRepo.DeleteByStory},
		{"locations", d.LocRepo.DeleteByStory},
		{"chapters", d.ChapterRepo.DeleteByStory},
		{"bibles", d.BibleRepo.DeleteByStory},
	}

	var firstErr error
	for _, st := range steps {
		if err := st.fn(ctx, storyID); err != nil {
			slog.Error("cascade delete failed", "collection", st.name, "storyId", storyID, "error", err)
			if firstErr == nil {
				firstErr = fmt.Errorf("cascade delete %s: %w", st.name, err)
			}
		}
	}
	return firstErr
}
