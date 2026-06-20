package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/repository"
)

type BibleService struct {
	bibleRepo  repository.BibleRepository
	storyRepo  repository.StoryRepository
	charRepo   repository.CharacterRepository
	genSvc     llm.BibleService
	genInFlight sync.Map
}

func NewBibleService(bibleRepo repository.BibleRepository, storyRepo repository.StoryRepository, charRepo repository.CharacterRepository, genSvc llm.BibleService) *BibleService {
	return &BibleService{
		bibleRepo:  bibleRepo,
		storyRepo:  storyRepo,
		charRepo:   charRepo,
		genSvc:     genSvc,
	}
}

func (s *BibleService) Get(ctx context.Context, storyID string) (*domain.StoryBible, error) {
	return s.bibleRepo.GetByStory(ctx, storyID)
}

func (s *BibleService) Generate(ctx context.Context, storyID string) (*domain.StoryBible, error) {
	if _, loaded := s.genInFlight.LoadOrStore(storyID, true); loaded {
		return nil, fmt.Errorf("bible generation already in progress for story %s", storyID)
	}
	defer s.genInFlight.Delete(storyID)

	story, err := s.storyRepo.Get(ctx, storyID)
	if err != nil {
		return nil, fmt.Errorf("get story: %w", err)
	}
	if story == nil {
		return nil, fmt.Errorf("story not found")
	}

	synopsis := story.MainPrompt
	if synopsis == "" {
		synopsis = story.GeneralPrompt
	}
	if synopsis == "" {
		synopsis = story.Title
	}

	characters, _ := s.charRepo.ListByStory(ctx, storyID)

	bible, err := s.genSvc.GenerateBible(ctx, storyID, synopsis, characters)
	if err != nil {
		return nil, fmt.Errorf("generate bible: %w", err)
	}

	bible.CreatedAt = time.Now()
	bible.UpdatedAt = time.Now()

	if err := s.bibleRepo.Create(ctx, bible); err != nil {
		return nil, fmt.Errorf("store bible: %w", err)
	}

	slog.Info("bible generated", "storyId", storyID, "bibleId", bible.ID)
	return bible, nil
}

func (s *BibleService) Update(ctx context.Context, bible *domain.StoryBible) error {
	bible.UpdatedAt = time.Now()
	return s.bibleRepo.Update(ctx, bible)
}

func (s *BibleService) DeleteByStory(ctx context.Context, storyID string) error {
	return s.bibleRepo.DeleteByStory(ctx, storyID)
}
