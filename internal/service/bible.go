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
	existing, err := s.bibleRepo.GetByStory(ctx, bible.StoryID)
	if err != nil {
		return fmt.Errorf("get bible for update: %w", err)
	}
	if existing == nil {
		bible.UpdatedAt = time.Now()
		return s.bibleRepo.Update(ctx, bible)
	}
	if bible.Title != "" {
		existing.Title = bible.Title
	}
	if bible.World != "" {
		existing.World = bible.World
	}
	if bible.Dimensions != nil {
		existing.Dimensions = bible.Dimensions
	}
	if bible.WorldRules != nil {
		existing.WorldRules = bible.WorldRules
	}
	if bible.MagicSystems != nil {
		existing.MagicSystems = bible.MagicSystems
	}
	if bible.Factions != nil {
		existing.Factions = bible.Factions
	}
	if bible.Cultures != nil {
		existing.Cultures = bible.Cultures
	}
	if bible.Tone != "" {
		existing.Tone = bible.Tone
	}
	if bible.CentralTheme != "" {
		existing.CentralTheme = bible.CentralTheme
	}
	if bible.NarrativeVoice != "" {
		existing.NarrativeVoice = bible.NarrativeVoice
	}
	existing.UpdatedAt = time.Now()
	return s.bibleRepo.Update(ctx, existing)
}

func (s *BibleService) DeleteByStory(ctx context.Context, storyID string) error {
	return s.bibleRepo.DeleteByStory(ctx, storyID)
}

func (s *BibleService) LinkBibleToStory(ctx context.Context, bibleID, targetStoryID string) error {
	bible, err := s.bibleRepo.Get(ctx, bibleID)
	if err != nil {
		return fmt.Errorf("get bible: %w", err)
	}
	if bible == nil {
		return fmt.Errorf("bible %s not found", bibleID)
	}
	for _, sid := range bible.ReferenceStories {
		if sid == targetStoryID {
			return nil
		}
	}
	bible.ReferenceStories = append(bible.ReferenceStories, targetStoryID)
	bible.UpdatedAt = time.Now()
	return s.bibleRepo.Update(ctx, bible)
}

func (s *BibleService) UnlinkBibleFromStory(ctx context.Context, bibleID, targetStoryID string) error {
	bible, err := s.bibleRepo.Get(ctx, bibleID)
	if err != nil {
		return fmt.Errorf("get bible: %w", err)
	}
	if bible == nil {
		return fmt.Errorf("bible %s not found", bibleID)
	}
	filtered := bible.ReferenceStories[:0]
	for _, sid := range bible.ReferenceStories {
		if sid != targetStoryID {
			filtered = append(filtered, sid)
		}
	}
	bible.ReferenceStories = filtered
	bible.UpdatedAt = time.Now()
	return s.bibleRepo.Update(ctx, bible)
}

func (s *BibleService) ListReferencingBibles(ctx context.Context, storyID string) ([]*domain.StoryBible, error) {
	return s.bibleRepo.ListByReferencingStory(ctx, storyID)
}
