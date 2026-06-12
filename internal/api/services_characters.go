package api

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/canon"
)

type charService struct {
	chars   []canon.Character
	version map[uuid.UUID]int
}

func NewCharService() *charService {
	return &charService{version: make(map[uuid.UUID]int)}
}

func (s *charService) Create(ctx context.Context, name, persona, backstory, moralAlignment string, personality, flaws, goals, traits, voiceSamples []string, parentID *uuid.UUID, relationships map[string]string) (*canon.Character, error) {
	c := canon.Character{
		ID:             uuid.New(),
		Version:        1,
		Name:           name,
		Persona:        persona,
		Backstory:      backstory,
		MoralAlignment: moralAlignment,
		Personality:    personality,
		Flaws:          flaws,
		Goals:          goals,
		Traits:         traits,
		VoiceSamples:   voiceSamples,
		ParentID:       parentID,
		Relationships:  relationships,
		CreatedAt:      time.Now(),
	}
	s.chars = append(s.chars, c)
	s.version[c.ID] = 2
	return &c, nil
}

func (s *charService) Get(ctx context.Context, id uuid.UUID, version int) (*canon.Character, error) {
	var latest *canon.Character
	for i := range s.chars {
		if s.chars[i].ID == id {
			if version > 0 && s.chars[i].Version == version {
				return &s.chars[i], nil
			}
			if latest == nil || s.chars[i].Version > latest.Version {
				latest = &s.chars[i]
			}
		}
	}
	if latest == nil {
		return nil, fmt.Errorf("character %s not found", id)
	}
	return latest, nil
}

func (s *charService) Update(ctx context.Context, id uuid.UUID, name, persona, backstory, moralAlignment string, personality, flaws, goals, traits, voiceSamples []string, parentID *uuid.UUID, relationships map[string]string) (*canon.Character, error) {
	next := s.version[id]
	if next == 0 {
		return nil, fmt.Errorf("character %s not found", id)
	}
	c := canon.Character{
		ID:             id,
		Version:        next,
		Name:           name,
		Persona:        persona,
		Backstory:      backstory,
		MoralAlignment: moralAlignment,
		Personality:    personality,
		Flaws:          flaws,
		Goals:          goals,
		Traits:         traits,
		VoiceSamples:   voiceSamples,
		ParentID:       parentID,
		Relationships:  relationships,
		CreatedAt:      time.Now(),
	}
	s.chars = append(s.chars, c)
	s.version[id] = next + 1
	return &c, nil
}

func (s *charService) List(ctx context.Context) ([]canon.Character, error) {
	latest := make(map[uuid.UUID]canon.Character)
	for _, c := range s.chars {
		if existing, ok := latest[c.ID]; !ok || c.Version > existing.Version {
			latest[c.ID] = c
		}
	}
	result := make([]canon.Character, 0, len(latest))
	for _, c := range latest {
		result = append(result, c)
	}
	return result, nil
}

type actorService struct {
	actors []canon.Actor
}

func NewActorService() *actorService {
	return &actorService{}
}

func (s *actorService) Create(ctx context.Context, name, gender, ethnicity, race, skinTone, eyeColor, hairColor, hairStyle, build, nationality string, heightCm, weightKg, age int, traits map[string]interface{}) (*canon.Actor, error) {
	if traits == nil {
		traits = make(map[string]interface{})
	}
	a := canon.Actor{
		ID:          uuid.New(),
		Name:        name,
		Gender:      gender,
		Ethnicity:   ethnicity,
		Race:        race,
		SkinTone:    skinTone,
		EyeColor:    eyeColor,
		HairColor:   hairColor,
		HairStyle:   hairStyle,
		Build:       build,
		HeightCm:    heightCm,
		WeightKg:    weightKg,
		Age:         age,
		Nationality: nationality,
		Traits:      traits,
		CreatedAt:   time.Now(),
	}
	s.actors = append(s.actors, a)
	return &a, nil
}

func (s *actorService) Get(ctx context.Context, id uuid.UUID) (*canon.Actor, error) {
	for i := range s.actors {
		if s.actors[i].ID == id {
			return &s.actors[i], nil
		}
	}
	return nil, fmt.Errorf("actor %s not found", id)
}

func (s *actorService) Update(ctx context.Context, id uuid.UUID, name, gender, ethnicity, race, skinTone, eyeColor, hairColor, hairStyle, build, nationality string, heightCm, weightKg, age int, traits map[string]interface{}) (*canon.Actor, error) {
	for i := range s.actors {
		if s.actors[i].ID == id {
			if traits == nil {
				traits = make(map[string]interface{})
			}
			s.actors[i].Name = name
			s.actors[i].Gender = gender
			s.actors[i].Ethnicity = ethnicity
			s.actors[i].Race = race
			s.actors[i].SkinTone = skinTone
			s.actors[i].EyeColor = eyeColor
			s.actors[i].HairColor = hairColor
			s.actors[i].HairStyle = hairStyle
			s.actors[i].Build = build
			s.actors[i].HeightCm = heightCm
			s.actors[i].WeightKg = weightKg
			s.actors[i].Age = age
			s.actors[i].Nationality = nationality
			s.actors[i].Traits = traits
			return &s.actors[i], nil
		}
	}
	return nil, fmt.Errorf("actor %s not found", id)
}

func (s *actorService) List(ctx context.Context) ([]canon.Actor, error) {
	r := make([]canon.Actor, len(s.actors))
	copy(r, s.actors)
	return r, nil
}

type characterTraitService struct {
	traits      []canon.CharacterTrait
	assignments []canon.TraitAssignment
}

func NewCharacterTraitService() *characterTraitService {
	return &characterTraitService{}
}

func (s *characterTraitService) Create(ctx context.Context, name, category, description string) (*canon.CharacterTrait, error) {
	t := canon.CharacterTrait{
		ID:          uuid.New(),
		Name:        name,
		Category:    category,
		Description: description,
		CreatedAt:   time.Now(),
	}
	s.traits = append(s.traits, t)
	return &t, nil
}

func (s *characterTraitService) Get(ctx context.Context, id uuid.UUID) (*canon.CharacterTrait, error) {
	for i := range s.traits {
		if s.traits[i].ID == id {
			return &s.traits[i], nil
		}
	}
	return nil, fmt.Errorf("trait %s not found", id)
}

func (s *characterTraitService) List(ctx context.Context) ([]canon.CharacterTrait, error) {
	r := make([]canon.CharacterTrait, len(s.traits))
	copy(r, s.traits)
	return r, nil
}

func (s *characterTraitService) Assign(ctx context.Context, characterID, traitID uuid.UUID, intensity int, note string) error {
	for i := range s.assignments {
		if s.assignments[i].CharacterID == characterID && s.assignments[i].TraitID == traitID {
			s.assignments[i].Intensity = intensity
			s.assignments[i].Note = note
			return nil
		}
	}
	s.assignments = append(s.assignments, canon.TraitAssignment{
		CharacterID: characterID,
		TraitID:     traitID,
		Intensity:   intensity,
		Note:        note,
	})
	return nil
}

func (s *characterTraitService) Unassign(ctx context.Context, characterID, traitID uuid.UUID) error {
	for i := range s.assignments {
		if s.assignments[i].CharacterID == characterID && s.assignments[i].TraitID == traitID {
			s.assignments = append(s.assignments[:i], s.assignments[i+1:]...)
			return nil
		}
	}
	return nil
}

func (s *characterTraitService) GetAssignments(ctx context.Context, characterID uuid.UUID) ([]canon.TraitAssignment, error) {
	var result []canon.TraitAssignment
	for _, a := range s.assignments {
		if a.CharacterID == characterID {
			result = append(result, a)
		}
	}
	return result, nil
}

type castingService struct {
	casts []canon.Casting
}

func NewCastingService() *castingService {
	return &castingService{}
}

func (s *castingService) Create(ctx context.Context, storyID, actorID, characterID uuid.UUID, roleType string) (*canon.Casting, error) {
	c := canon.Casting{
		ID:          uuid.New(),
		StoryID:     storyID,
		ActorID:     actorID,
		CharacterID: characterID,
		RoleType:    roleType,
		CreatedAt:   time.Now(),
	}
	s.casts = append(s.casts, c)
	return &c, nil
}

func (s *castingService) GetForStory(ctx context.Context, storyID uuid.UUID) ([]canon.Casting, error) {
	var result []canon.Casting
	for _, c := range s.casts {
		if c.StoryID == storyID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (s *castingService) GetForCharacter(ctx context.Context, characterID uuid.UUID) ([]canon.Casting, error) {
	var result []canon.Casting
	for _, c := range s.casts {
		if c.CharacterID == characterID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (s *castingService) GetForActor(ctx context.Context, actorID uuid.UUID) ([]canon.Casting, error) {
	var result []canon.Casting
	for _, c := range s.casts {
		if c.ActorID == actorID {
			result = append(result, c)
		}
	}
	return result, nil
}
