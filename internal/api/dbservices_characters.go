package api

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/db"
)

func NewDBCharService(q *db.Queries) *dbCharService {
	return &dbCharService{q: q}
}

type dbCharService struct{ q *db.Queries }

func (s *dbCharService) Create(name, persona, backstory, moralAlignment string, personality, flaws, goals, traits, voiceSamples []string, parentID *uuid.UUID, relationships map[string]string) (*canon.Character, error) {
	if voiceSamples == nil {
		voiceSamples = []string{}
	}
	var pid pgtype.UUID
	if parentID != nil {
		pid = toUUID(*parentID)
	}
	c, err := s.q.CreateCharacter(context.Background(), db.CreateCharacterParams{
		Name:           name,
		Persona:        persona,
		Backstory:      backstory,
		MoralAlignment: moralAlignment,
		Personality:    jsonBytes(personality),
		Flaws:          jsonBytes(flaws),
		Goals:          jsonBytes(goals),
		Traits:         jsonBytes(traits),
		VoiceSamples:   voiceSamples,
		Relationships:  jsonBytes(relationships),
		ParentID:       pid,
	})
	if err != nil {
		return nil, err
	}
	return toDomainChar(c), nil
}

func (s *dbCharService) Get(id uuid.UUID, version int) (*canon.Character, error) {
	if version > 0 {
		c, err := s.q.GetCharacterAtVersion(context.Background(), db.GetCharacterAtVersionParams{
			ID:      toUUID(id),
			Version: int32(version),
		})
		if err != nil {
			return nil, err
		}
		return toDomainChar(c), nil
	}
	c, err := s.q.GetCharacterLatest(context.Background(), toUUID(id))
	if err != nil {
		return nil, err
	}
	return toDomainCharFromLatest(c), nil
}

func (s *dbCharService) Update(id uuid.UUID, name, persona, backstory, moralAlignment string, personality, flaws, goals, traits, voiceSamples []string, parentID *uuid.UUID, relationships map[string]string) (*canon.Character, error) {
	if voiceSamples == nil {
		voiceSamples = []string{}
	}
	var pid pgtype.UUID
	if parentID != nil {
		pid = toUUID(*parentID)
	}
	c, err := s.q.UpdateCharacter(context.Background(), db.UpdateCharacterParams{
		ID:             toUUID(id),
		Name:           name,
		Persona:        persona,
		Backstory:      backstory,
		MoralAlignment: moralAlignment,
		Column6:        jsonBytes(personality),
		Column7:        jsonBytes(flaws),
		Column8:        jsonBytes(goals),
		Column9:        jsonBytes(traits),
		VoiceSamples:   voiceSamples,
		Column11:       jsonBytes(relationships),
		ParentID:       pid,
	})
	if err != nil {
		return nil, err
	}
	return toDomainChar(c), nil
}

func (s *dbCharService) List() ([]canon.Character, error) {
	chars, err := s.q.ListCharacters(context.Background())
	if err != nil {
		return nil, err
	}
	result := make([]canon.Character, len(chars))
	for i, c := range chars {
		result[i] = *toDomainCharFromLatest(c)
	}
	return result, nil
}

func toDomainChar(c db.Character) *canon.Character {
	var traits []string
	json.Unmarshal(c.Traits, &traits)
	var personality []string
	json.Unmarshal(c.Personality, &personality)
	var flaws []string
	json.Unmarshal(c.Flaws, &flaws)
	var goals []string
	json.Unmarshal(c.Goals, &goals)
	var rel map[string]string
	json.Unmarshal(c.Relationships, &rel)
	var parentID *uuid.UUID
	if c.ParentID.Valid {
		p := fromUUID(c.ParentID)
		parentID = &p
	}
	return &canon.Character{
		ID:             fromUUID(c.ID),
		Version:        int(c.Version),
		Name:           c.Name,
		Persona:        c.Persona,
		Backstory:      c.Backstory,
		MoralAlignment: c.MoralAlignment,
		Personality:    personality,
		Flaws:          flaws,
		Goals:          goals,
		Traits:         traits,
		VoiceSamples:   c.VoiceSamples,
		ParentID:       parentID,
		Relationships:  rel,
		CreatedAt:      c.CreatedAt.Time,
	}
}

func toDomainCharFromLatest(c db.LatestCharacter) *canon.Character {
	var traits []string
	json.Unmarshal(c.Traits, &traits)
	var personality []string
	json.Unmarshal(c.Personality, &personality)
	var flaws []string
	json.Unmarshal(c.Flaws, &flaws)
	var goals []string
	json.Unmarshal(c.Goals, &goals)
	var rel map[string]string
	json.Unmarshal(c.Relationships, &rel)
	var parentID *uuid.UUID
	if c.ParentID.Valid {
		p := fromUUID(c.ParentID)
		parentID = &p
	}
	return &canon.Character{
		ID:             fromUUID(c.ID),
		Version:        int(c.Version),
		Name:           c.Name,
		Persona:        c.Persona,
		Backstory:      c.Backstory,
		MoralAlignment: c.MoralAlignment,
		Personality:    personality,
		Flaws:          flaws,
		Goals:          goals,
		Traits:         traits,
		VoiceSamples:   c.VoiceSamples,
		ParentID:       parentID,
		Relationships:  rel,
		CreatedAt:      c.CreatedAt.Time,
	}
}

// ── Actor (DB-backed) ──────────────────────────────────────────

func NewDBActorService(q *db.Queries) *dbActorService {
	return &dbActorService{q: q}
}

type dbActorService struct{ q *db.Queries }

func (s *dbActorService) Create(name, gender, ethnicity, race, skinTone, eyeColor, hairColor, hairStyle, build, nationality string, heightCm, weightKg, age int, traits map[string]interface{}) (*canon.Actor, error) {
	if traits == nil {
		traits = make(map[string]interface{})
	}
	a, err := s.q.CreateActor(context.Background(), db.CreateActorParams{
		Name:        name,
		Gender:      gender,
		Ethnicity:   ethnicity,
		Race:        race,
		SkinTone:    skinTone,
		EyeColor:    eyeColor,
		HairColor:   hairColor,
		HairStyle:   hairStyle,
		Build:       build,
		HeightCm:    int32(heightCm),
		WeightKg:    int32(weightKg),
		Age:         int32(age),
		Nationality: nationality,
		Traits:      jsonBytes(traits),
	})
	if err != nil {
		return nil, err
	}
	return toDomainActor(a), nil
}

func (s *dbActorService) Get(id uuid.UUID) (*canon.Actor, error) {
	a, err := s.q.GetActor(context.Background(), toUUID(id))
	if err != nil {
		return nil, err
	}
	return toDomainActor(a), nil
}

func (s *dbActorService) Update(id uuid.UUID, name, gender, ethnicity, race, skinTone, eyeColor, hairColor, hairStyle, build, nationality string, heightCm, weightKg, age int, traits map[string]interface{}) (*canon.Actor, error) {
	if traits == nil {
		traits = make(map[string]interface{})
	}
	a, err := s.q.UpdateActor(context.Background(), db.UpdateActorParams{
		ID:          toUUID(id),
		Name:        name,
		Gender:      gender,
		Ethnicity:   ethnicity,
		Race:        race,
		SkinTone:    skinTone,
		EyeColor:    eyeColor,
		HairColor:   hairColor,
		HairStyle:   hairStyle,
		Build:       build,
		HeightCm:    int32(heightCm),
		WeightKg:    int32(weightKg),
		Age:         int32(age),
		Nationality: nationality,
		Column15:    jsonBytes(traits),
	})
	if err != nil {
		return nil, err
	}
	return toDomainActor(a), nil
}

func (s *dbActorService) List() ([]canon.Actor, error) {
	actors, err := s.q.ListActors(context.Background())
	if err != nil {
		return nil, err
	}
	result := make([]canon.Actor, len(actors))
	for i, a := range actors {
		result[i] = *toDomainActor(a)
	}
	return result, nil
}

func toDomainActor(a db.Actor) *canon.Actor {
	var traits map[string]interface{}
	json.Unmarshal(a.Traits, &traits)
	if traits == nil {
		traits = make(map[string]interface{})
	}
	return &canon.Actor{
		ID:          fromUUID(a.ID),
		Name:        a.Name,
		Gender:      a.Gender,
		Ethnicity:   a.Ethnicity,
		Race:        a.Race,
		SkinTone:    a.SkinTone,
		EyeColor:    a.EyeColor,
		HairColor:   a.HairColor,
		HairStyle:   a.HairStyle,
		Build:       a.Build,
		HeightCm:    int(a.HeightCm),
		WeightKg:    int(a.WeightKg),
		Age:         int(a.Age),
		Nationality: a.Nationality,
		Traits:      traits,
		CreatedAt:   a.CreatedAt.Time,
	}
}

// ── CharacterTrait (DB-backed) ──────────────────────────────────

func NewDBCharacterTraitService(q *db.Queries) *dbCharacterTraitService {
	return &dbCharacterTraitService{q: q}
}

type dbCharacterTraitService struct{ q *db.Queries }

func (s *dbCharacterTraitService) Create(name, category, description string) (*canon.CharacterTrait, error) {
	t, err := s.q.CreateCharacterTrait(context.Background(), db.CreateCharacterTraitParams{
		Name:        name,
		Category:    category,
		Description: description,
	})
	if err != nil {
		return nil, err
	}
	return &canon.CharacterTrait{
		ID:          fromUUID(t.ID),
		Name:        t.Name,
		Category:    t.Category,
		Description: t.Description,
		CreatedAt:   t.CreatedAt.Time,
	}, nil
}

func (s *dbCharacterTraitService) Get(id uuid.UUID) (*canon.CharacterTrait, error) {
	t, err := s.q.GetCharacterTrait(context.Background(), toUUID(id))
	if err != nil {
		return nil, err
	}
	return &canon.CharacterTrait{
		ID:          fromUUID(t.ID),
		Name:        t.Name,
		Category:    t.Category,
		Description: t.Description,
		CreatedAt:   t.CreatedAt.Time,
	}, nil
}

func (s *dbCharacterTraitService) List() ([]canon.CharacterTrait, error) {
	traits, err := s.q.ListCharacterTraits(context.Background())
	if err != nil {
		return nil, err
	}
	result := make([]canon.CharacterTrait, len(traits))
	for i, t := range traits {
		result[i] = canon.CharacterTrait{
			ID:          fromUUID(t.ID),
			Name:        t.Name,
			Category:    t.Category,
			Description: t.Description,
			CreatedAt:   t.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *dbCharacterTraitService) Assign(characterID, traitID uuid.UUID, intensity int, note string) error {
	return s.q.AssignTrait(context.Background(), db.AssignTraitParams{
		CharacterID: toUUID(characterID),
		TraitID:     toUUID(traitID),
		Intensity:   int32(intensity),
		Note:        note,
	})
}

func (s *dbCharacterTraitService) Unassign(characterID, traitID uuid.UUID) error {
	return s.q.UnassignTrait(context.Background(), db.UnassignTraitParams{
		CharacterID: toUUID(characterID),
		TraitID:     toUUID(traitID),
	})
}

func (s *dbCharacterTraitService) GetAssignments(characterID uuid.UUID) ([]canon.TraitAssignment, error) {
	rows, err := s.q.GetTraitAssignments(context.Background(), toUUID(characterID))
	if err != nil {
		return nil, err
	}
	result := make([]canon.TraitAssignment, len(rows))
	for i, r := range rows {
		result[i] = canon.TraitAssignment{
			CharacterID: fromUUID(r.CharacterID),
			TraitID:     fromUUID(r.TraitID),
			Intensity:   int(r.Intensity),
			Note:        r.Note,
		}
	}
	return result, nil
}

// ── Casting (DB-backed) ────────────────────────────────────────

func NewDBCastingService(q *db.Queries) *dbCastingService {
	return &dbCastingService{q: q}
}

type dbCastingService struct{ q *db.Queries }

func (s *dbCastingService) Create(storyID, actorID, characterID uuid.UUID, roleType string) (*canon.Casting, error) {
	c, err := s.q.CreateCasting(context.Background(), db.CreateCastingParams{
		StoryID:     toUUID(storyID),
		ActorID:     toUUID(actorID),
		CharacterID: toUUID(characterID),
		RoleType:    roleType,
	})
	if err != nil {
		return nil, err
	}
	return &canon.Casting{
		ID:          fromUUID(c.ID),
		StoryID:     fromUUID(c.StoryID),
		ActorID:     fromUUID(c.ActorID),
		CharacterID: fromUUID(c.CharacterID),
		RoleType:    c.RoleType,
		CreatedAt:   c.CreatedAt.Time,
	}, nil
}

func (s *dbCastingService) GetForStory(storyID uuid.UUID) ([]canon.Casting, error) {
	rows, err := s.q.ListCastingForStory(context.Background(), toUUID(storyID))
	if err != nil {
		return nil, err
	}
	result := make([]canon.Casting, len(rows))
	for i, r := range rows {
		result[i] = canon.Casting{
			ID:          fromUUID(r.ID),
			StoryID:     fromUUID(r.StoryID),
			ActorID:     fromUUID(r.ActorID),
			CharacterID: fromUUID(r.CharacterID),
			RoleType:    r.RoleType,
			CreatedAt:   r.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *dbCastingService) GetForCharacter(characterID uuid.UUID) ([]canon.Casting, error) {
	rows, err := s.q.ListCastingForCharacter(context.Background(), toUUID(characterID))
	if err != nil {
		return nil, err
	}
	result := make([]canon.Casting, len(rows))
	for i, r := range rows {
		result[i] = canon.Casting{
			ID:          fromUUID(r.ID),
			StoryID:     fromUUID(r.StoryID),
			ActorID:     fromUUID(r.ActorID),
			CharacterID: fromUUID(r.CharacterID),
			RoleType:    r.RoleType,
			CreatedAt:   r.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *dbCastingService) GetForActor(actorID uuid.UUID) ([]canon.Casting, error) {
	rows, err := s.q.ListCastingForActor(context.Background(), toUUID(actorID))
	if err != nil {
		return nil, err
	}
	result := make([]canon.Casting, len(rows))
	for i, r := range rows {
		result[i] = canon.Casting{
			ID:          fromUUID(r.ID),
			StoryID:     fromUUID(r.StoryID),
			ActorID:     fromUUID(r.ActorID),
			CharacterID: fromUUID(r.CharacterID),
			RoleType:    r.RoleType,
			CreatedAt:   r.CreatedAt.Time,
		}
	}
	return result, nil
}
