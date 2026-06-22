package service

import (
	"context"
	"fmt"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type CharacterService struct {
	charRepo  repository.CharacterRepository
	stateRepo repository.CharacterStateRepository
	memRepo   repository.MemoryRepository
}

func NewCharacterService(
	charRepo repository.CharacterRepository,
	stateRepo repository.CharacterStateRepository,
	memRepo repository.MemoryRepository,
) *CharacterService {
	return &CharacterService{charRepo: charRepo, stateRepo: stateRepo, memRepo: memRepo}
}

func (s *CharacterService) Create(ctx context.Context, c *domain.Character) (*domain.Character, error) {
	if err := s.charRepo.Create(ctx, c); err != nil {
		return nil, fmt.Errorf("create character: %w", err)
	}
	return c, nil
}

func (s *CharacterService) Get(ctx context.Context, id string) (*domain.Character, error) {
	return s.charRepo.Get(ctx, id)
}

func (s *CharacterService) GetLatest(ctx context.Context, charID string) (*domain.Character, error) {
	return s.charRepo.GetLatest(ctx, charID)
}

func (s *CharacterService) Update(ctx context.Context, c *domain.Character) (*domain.Character, error) {
	existing, err := s.charRepo.GetLatest(ctx, c.CharID)
	if err != nil {
		return nil, fmt.Errorf("get latest character: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("character not found: %w", ErrNotFound)
	}
	if c.Name != "" {
		existing.Name = c.Name
	}
	if c.Persona != "" {
		existing.Persona = c.Persona
	}
	if c.Backstory != "" {
		existing.Backstory = c.Backstory
	}
	if c.Personality != nil {
		existing.Personality = c.Personality
	}
	if c.MoralAlignment != "" {
		existing.MoralAlignment = c.MoralAlignment
	}
	if c.Goals != nil {
		existing.Goals = c.Goals
	}
	if c.Flaws != nil {
		existing.Flaws = c.Flaws
	}
	if c.Traits != nil {
		existing.Traits = c.Traits
	}
	if c.VoiceSamples != nil {
		existing.VoiceSamples = c.VoiceSamples
	}
	if c.Relationships != nil {
		existing.Relationships = c.Relationships
	}
	if c.Want != "" {
		existing.Want = c.Want
	}
	if c.Need != "" {
		existing.Need = c.Need
	}
	if c.FalseBelief != "" {
		existing.FalseBelief = c.FalseBelief
	}
	if c.Fear != "" {
		existing.Fear = c.Fear
	}
	if c.ArcType != "" {
		existing.ArcType = c.ArcType
	}
	if err := s.charRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update character: %w", err)
	}
	return existing, nil
}

func (s *CharacterService) List(ctx context.Context, storyID string) ([]*domain.Character, error) {
	return s.charRepo.ListByStory(ctx, storyID)
}

func (s *CharacterService) MigrateCharacter(ctx context.Context, charID, targetStoryID string) (*domain.Character, error) {
	char, err := s.charRepo.Get(ctx, charID)
	if err != nil {
		return nil, fmt.Errorf("get character: %w", err)
	}
	if char == nil {
		return nil, fmt.Errorf("character not found: %w", ErrNotFound)
	}
	now := time.Now()
	migrated := &domain.Character{
		CharID:         "",
		StoryID:        targetStoryID,
		Name:           char.Name,
		Persona:        char.Persona,
		Backstory:      char.Backstory,
		Personality:    char.Personality,
		MoralAlignment: char.MoralAlignment,
		Goals:          char.Goals,
		Flaws:          char.Flaws,
		Traits:         char.Traits,
		VoiceSamples:   char.VoiceSamples,
		Relationships:  char.Relationships,
		RelData:        char.RelData,
		Want:           char.Want,
		Need:           char.Need,
		FalseBelief:    char.FalseBelief,
		Fear:           char.Fear,
		ArcType:        char.ArcType,
		MigratedFrom:   char.StoryID,
		MigratedAt:     &now,
		CreatedAt:      time.Now(),
	}
	if err := s.charRepo.Create(ctx, migrated); err != nil {
		return nil, fmt.Errorf("migrate character definition: %w", err)
	}
	if migrated.CharID == "" {
		migrated.CharID = migrated.ID
	}

	srcCharID := char.CharID
	dstCharID := migrated.CharID
	if s.stateRepo != nil {
		states, err := s.stateRepo.ListByCharacter(ctx, srcCharID)
		if err != nil {
			return nil, fmt.Errorf("list character states: %w", err)
		}
		for _, st := range states {
			cp := *st
			cp.CharacterID = dstCharID
			cp.StoryID = targetStoryID
			cp.CreatedAt = time.Now()
			if err := s.stateRepo.Append(ctx, &cp); err != nil {
				return nil, fmt.Errorf("migrate character state: %w", err)
			}
		}
	}

	if s.memRepo != nil {
		mems, err := s.memRepo.ListByCharacter(ctx, srcCharID)
		if err != nil {
			return nil, fmt.Errorf("list character memories: %w", err)
		}
		for _, m := range mems {
			cp := *m
			cp.ID = ""
			cp.CharacterID = dstCharID
			cp.StoryID = targetStoryID
			cp.CreatedAt = time.Now()
			if err := s.memRepo.Create(ctx, &cp); err != nil {
				return nil, fmt.Errorf("migrate character memory: %w", err)
			}
		}
	}

	return migrated, nil
}
