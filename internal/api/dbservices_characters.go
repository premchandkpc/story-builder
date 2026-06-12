package api

import (
	"context"
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/db"
)

func NewDBCharService(q *db.Queries) *dbCharService {
	return &dbCharService{q: q}
}

type dbCharService struct{ q *db.Queries }

func (s *dbCharService) Create(ctx context.Context, name, persona, backstory, moralAlignment string, personality, flaws, goals, traits, voiceSamples []string, parentID *uuid.UUID, relationships map[string]string) (*canon.Character, error) {
	if voiceSamples == nil {
		voiceSamples = []string{}
	}
	var pid pgtype.UUID
	if parentID != nil {
		pid = toUUID(*parentID)
	}
	c, err := s.q.CreateCharacter(ctx, db.CreateCharacterParams{
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

func (s *dbCharService) Get(ctx context.Context, id uuid.UUID, version int) (*canon.Character, error) {
	if version > 0 {
		c, err := s.q.GetCharacterAtVersion(ctx, db.GetCharacterAtVersionParams{
			ID:      toUUID(id),
			Version: int32(version),
		})
		if err != nil {
			return nil, err
		}
		return toDomainChar(c), nil
	}
	c, err := s.q.GetCharacterLatest(ctx, toUUID(id))
	if err != nil {
		return nil, err
	}
	return toDomainCharFromLatest(c), nil
}

func (s *dbCharService) Update(ctx context.Context, id uuid.UUID, name, persona, backstory, moralAlignment string, personality, flaws, goals, traits, voiceSamples []string, parentID *uuid.UUID, relationships map[string]string) (*canon.Character, error) {
	if voiceSamples == nil {
		voiceSamples = []string{}
	}
	var pid pgtype.UUID
	if parentID != nil {
		pid = toUUID(*parentID)
	}
	c, err := s.q.UpdateCharacter(ctx, db.UpdateCharacterParams{
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

func (s *dbCharService) List(ctx context.Context) ([]canon.Character, error) {
	chars, err := s.q.ListCharacters(ctx)
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
	if err := json.Unmarshal(c.Traits, &traits); err != nil {
		log.Printf("unmarshal traits for character %s: %v", fromUUID(c.ID), err)
	}
	var personality []string
	if err := json.Unmarshal(c.Personality, &personality); err != nil {
		log.Printf("unmarshal personality for character %s: %v", fromUUID(c.ID), err)
	}
	var flaws []string
	if err := json.Unmarshal(c.Flaws, &flaws); err != nil {
		log.Printf("unmarshal flaws for character %s: %v", fromUUID(c.ID), err)
	}
	var goals []string
	if err := json.Unmarshal(c.Goals, &goals); err != nil {
		log.Printf("unmarshal goals for character %s: %v", fromUUID(c.ID), err)
	}
	var rel map[string]string
	if err := json.Unmarshal(c.Relationships, &rel); err != nil {
		log.Printf("unmarshal relationships for character %s: %v", fromUUID(c.ID), err)
	}
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
	if err := json.Unmarshal(c.Traits, &traits); err != nil {
		log.Printf("unmarshal traits for character %s: %v", fromUUID(c.ID), err)
	}
	var personality []string
	if err := json.Unmarshal(c.Personality, &personality); err != nil {
		log.Printf("unmarshal personality for character %s: %v", fromUUID(c.ID), err)
	}
	var flaws []string
	if err := json.Unmarshal(c.Flaws, &flaws); err != nil {
		log.Printf("unmarshal flaws for character %s: %v", fromUUID(c.ID), err)
	}
	var goals []string
	if err := json.Unmarshal(c.Goals, &goals); err != nil {
		log.Printf("unmarshal goals for character %s: %v", fromUUID(c.ID), err)
	}
	var rel map[string]string
	if err := json.Unmarshal(c.Relationships, &rel); err != nil {
		log.Printf("unmarshal relationships for character %s: %v", fromUUID(c.ID), err)
	}
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

func (s *dbActorService) Create(ctx context.Context, name, gender, ethnicity, race, skinTone, eyeColor, hairColor, hairStyle, build, nationality string, heightCm, weightKg, age int, traits map[string]interface{}) (*canon.Actor, error) {
	if traits == nil {
		traits = make(map[string]interface{})
	}
	a, err := s.q.CreateActor(ctx, db.CreateActorParams{
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
		Traits:      jsonBytes(map[string]interface{}{}),
	})
	if err != nil {
		return nil, err
	}
	if err := s.persistActorTraits(ctx, a.ID, traits); err != nil {
		return nil, err
	}
	loadedTraits, err := s.loadActorTraits(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	return toDomainActor(a, loadedTraits), nil
}

func (s *dbActorService) Get(ctx context.Context, id uuid.UUID) (*canon.Actor, error) {
	a, err := s.q.GetActor(ctx, toUUID(id))
	if err != nil {
		return nil, err
	}
	loadedTraits, err := s.loadActorTraits(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	return toDomainActor(a, loadedTraits), nil
}

func (s *dbActorService) Update(ctx context.Context, id uuid.UUID, name, gender, ethnicity, race, skinTone, eyeColor, hairColor, hairStyle, build, nationality string, heightCm, weightKg, age int, traits map[string]interface{}) (*canon.Actor, error) {
	if traits == nil {
		traits = make(map[string]interface{})
	}
	a, err := s.q.UpdateActor(ctx, db.UpdateActorParams{
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
		Column15:    jsonBytes(map[string]interface{}{}),
	})
	if err != nil {
		return nil, err
	}
	if err := s.persistActorTraits(ctx, a.ID, traits); err != nil {
		return nil, err
	}
	loadedTraits, err := s.loadActorTraits(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	return toDomainActor(a, loadedTraits), nil
}

func (s *dbActorService) List(ctx context.Context) ([]canon.Actor, error) {
	actors, err := s.q.ListActors(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]canon.Actor, len(actors))
	for i, a := range actors {
		loadedTraits, err := s.loadActorTraits(ctx, a.ID)
		if err != nil {
			return nil, err
		}
		result[i] = *toDomainActor(a, loadedTraits)
	}
	return result, nil
}

func (s *dbActorService) persistActorTraits(ctx context.Context, actorID pgtype.UUID, traits map[string]interface{}) error {
	if err := s.q.DeleteActorTraits(ctx, actorID); err != nil {
		return err
	}
	for key, value := range traits {
		encoded, err := actorTraitValueToJSON(value)
		if err != nil {
			return err
		}
		if _, err := s.q.CreateActorTrait(ctx, db.CreateActorTraitParams{ActorID: actorID, TraitKey: key, TraitValue: encoded}); err != nil {
			return err
		}
	}
	return nil
}

func (s *dbActorService) loadActorTraits(ctx context.Context, actorID pgtype.UUID) (map[string]interface{}, error) {
	rows, err := s.q.ListActorTraits(ctx, actorID)
	if err != nil {
		return nil, err
	}
	traits := make(map[string]interface{}, len(rows))
	for _, row := range rows {
		value, err := actorTraitValueFromJSON(row.TraitValue)
		if err != nil {
			traits[row.TraitKey] = row.TraitValue
			continue
		}
		traits[row.TraitKey] = value
	}
	return traits, nil
}

func actorTraitValueToJSON(value interface{}) (string, error) {
	if value == nil {
		return "null", nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func actorTraitValueFromJSON(raw string) (interface{}, error) {
	var value interface{}
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, err
	}
	return value, nil
}

func toDomainActor(a db.Actor, traits map[string]interface{}) *canon.Actor {
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

func (s *dbCharacterTraitService) Create(ctx context.Context, name, category, description string) (*canon.CharacterTrait, error) {
	t, err := s.q.CreateCharacterTrait(ctx, db.CreateCharacterTraitParams{
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

func (s *dbCharacterTraitService) Get(ctx context.Context, id uuid.UUID) (*canon.CharacterTrait, error) {
	t, err := s.q.GetCharacterTrait(ctx, toUUID(id))
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

func (s *dbCharacterTraitService) List(ctx context.Context) ([]canon.CharacterTrait, error) {
	traits, err := s.q.ListCharacterTraits(ctx)
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

func (s *dbCharacterTraitService) Assign(ctx context.Context, characterID, traitID uuid.UUID, intensity int, note string) error {
	return s.q.AssignTrait(ctx, db.AssignTraitParams{
		CharacterID: toUUID(characterID),
		TraitID:     toUUID(traitID),
		Intensity:   int32(intensity),
		Note:        note,
	})
}

func (s *dbCharacterTraitService) Unassign(ctx context.Context, characterID, traitID uuid.UUID) error {
	return s.q.UnassignTrait(ctx, db.UnassignTraitParams{
		CharacterID: toUUID(characterID),
		TraitID:     toUUID(traitID),
	})
}

func (s *dbCharacterTraitService) GetAssignments(ctx context.Context, characterID uuid.UUID) ([]canon.TraitAssignment, error) {
	rows, err := s.q.GetTraitAssignments(ctx, toUUID(characterID))
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

func (s *dbCastingService) Create(ctx context.Context, storyID, actorID, characterID uuid.UUID, roleType string) (*canon.Casting, error) {
	c, err := s.q.CreateCasting(ctx, db.CreateCastingParams{
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

func (s *dbCastingService) GetForStory(ctx context.Context, storyID uuid.UUID) ([]canon.Casting, error) {
	rows, err := s.q.ListCastingForStory(ctx, toUUID(storyID))
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

func (s *dbCastingService) GetForCharacter(ctx context.Context, characterID uuid.UUID) ([]canon.Casting, error) {
	rows, err := s.q.ListCastingForCharacter(ctx, toUUID(characterID))
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

func (s *dbCastingService) GetForActor(ctx context.Context, actorID uuid.UUID) ([]canon.Casting, error) {
	rows, err := s.q.ListCastingForActor(ctx, toUUID(actorID))
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
