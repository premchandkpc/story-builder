package canon

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pgvector/pgvector-go"
	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/db"
)

// ── CharacterService ────────────────────────────────────────────

type CharacterService interface {
	Create(ctx context.Context, name, persona, backstory, moralAlignment string, personality, flaws, goals, traits, voiceSamples []string, parentID *uuid.UUID, relationships map[string]string) (*canon.Character, error)
	Get(ctx context.Context, id uuid.UUID, version int) (*canon.Character, error)
	Update(ctx context.Context, id uuid.UUID, name, persona, backstory, moralAlignment string, personality, flaws, goals, traits, voiceSamples []string, parentID *uuid.UUID, relationships map[string]string) (*canon.Character, error)
	List(ctx context.Context) ([]canon.Character, error)
}

type MemoryCharacterService struct {
	chars   []canon.Character
	version map[uuid.UUID]int
}

func NewMemoryCharacterService() *MemoryCharacterService {
	return &MemoryCharacterService{version: make(map[uuid.UUID]int)}
}

func (s *MemoryCharacterService) Create(ctx context.Context, name, persona, backstory, moralAlignment string, personality, flaws, goals, traits, voiceSamples []string, parentID *uuid.UUID, relationships map[string]string) (*canon.Character, error) {
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

func (s *MemoryCharacterService) Get(ctx context.Context, id uuid.UUID, version int) (*canon.Character, error) {
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

func (s *MemoryCharacterService) Update(ctx context.Context, id uuid.UUID, name, persona, backstory, moralAlignment string, personality, flaws, goals, traits, voiceSamples []string, parentID *uuid.UUID, relationships map[string]string) (*canon.Character, error) {
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

func (s *MemoryCharacterService) List(ctx context.Context) ([]canon.Character, error) {
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

type DBCharacterService struct {
	q *db.Queries
}

func NewDBCharacterService(q *db.Queries) *DBCharacterService {
	return &DBCharacterService{q: q}
}

func (s *DBCharacterService) Create(ctx context.Context, name, persona, backstory, moralAlignment string, personality, flaws, goals, traits, voiceSamples []string, parentID *uuid.UUID, relationships map[string]string) (*canon.Character, error) {
	if voiceSamples == nil {
		voiceSamples = []string{}
	}
	var pid pgtype.UUID
	if parentID != nil {
		pid = db.ToUUID(*parentID)
	}
	c, err := s.q.CreateCharacter(ctx, db.CreateCharacterParams{
		Name:           name,
		Persona:        persona,
		Backstory:      backstory,
		MoralAlignment: moralAlignment,
		Personality:    db.JSONBytes(personality),
		Flaws:          db.JSONBytes(flaws),
		Goals:          db.JSONBytes(goals),
		Traits:         db.JSONBytes(traits),
		VoiceSamples:   voiceSamples,
		Relationships:  db.JSONBytes(relationships),
		ParentID:       pid,
	})
	if err != nil {
		return nil, err
	}
	return toDomainChar(c), nil
}

func (s *DBCharacterService) Get(ctx context.Context, id uuid.UUID, version int) (*canon.Character, error) {
	if version > 0 {
		c, err := s.q.GetCharacterAtVersion(ctx, db.GetCharacterAtVersionParams{
			ID:      db.ToUUID(id),
			Version: int32(version),
		})
		if err != nil {
			return nil, err
		}
		return toDomainChar(c), nil
	}
	c, err := s.q.GetCharacterLatest(ctx, db.ToUUID(id))
	if err != nil {
		return nil, err
	}
	return toDomainCharFromLatest(c), nil
}

func (s *DBCharacterService) Update(ctx context.Context, id uuid.UUID, name, persona, backstory, moralAlignment string, personality, flaws, goals, traits, voiceSamples []string, parentID *uuid.UUID, relationships map[string]string) (*canon.Character, error) {
	if voiceSamples == nil {
		voiceSamples = []string{}
	}
	var pid pgtype.UUID
	if parentID != nil {
		pid = db.ToUUID(*parentID)
	}
	c, err := s.q.UpdateCharacter(ctx, db.UpdateCharacterParams{
		ID:             db.ToUUID(id),
		Name:           name,
		Persona:        persona,
		Backstory:      backstory,
		MoralAlignment: moralAlignment,
		Column6:        db.JSONBytes(personality),
		Column7:        db.JSONBytes(flaws),
		Column8:        db.JSONBytes(goals),
		Column9:        db.JSONBytes(traits),
		VoiceSamples:   voiceSamples,
		Column11:       db.JSONBytes(relationships),
		ParentID:       pid,
	})
	if err != nil {
		return nil, err
	}
	return toDomainChar(c), nil
}

func (s *DBCharacterService) List(ctx context.Context) ([]canon.Character, error) {
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
		slog.Warn("unmarshal traits for character", "id", db.FromUUID(c.ID), "error", err)
	}
	var personality []string
	if err := json.Unmarshal(c.Personality, &personality); err != nil {
		slog.Warn("unmarshal personality for character", "id", db.FromUUID(c.ID), "error", err)
	}
	var flaws []string
	if err := json.Unmarshal(c.Flaws, &flaws); err != nil {
		slog.Warn("unmarshal flaws for character", "id", db.FromUUID(c.ID), "error", err)
	}
	var goals []string
	if err := json.Unmarshal(c.Goals, &goals); err != nil {
		slog.Warn("unmarshal goals for character", "id", db.FromUUID(c.ID), "error", err)
	}
	var rel map[string]string
	if err := json.Unmarshal(c.Relationships, &rel); err != nil {
		slog.Warn("unmarshal relationships for character", "id", db.FromUUID(c.ID), "error", err)
	}
	var parentID *uuid.UUID
	if c.ParentID.Valid {
		p := db.FromUUID(c.ParentID)
		parentID = &p
	}
	return &canon.Character{
		ID:             db.FromUUID(c.ID),
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
		slog.Warn("unmarshal traits for character", "id", db.FromUUID(c.ID), "error", err)
	}
	var personality []string
	if err := json.Unmarshal(c.Personality, &personality); err != nil {
		slog.Warn("unmarshal personality for character", "id", db.FromUUID(c.ID), "error", err)
	}
	var flaws []string
	if err := json.Unmarshal(c.Flaws, &flaws); err != nil {
		slog.Warn("unmarshal flaws for character", "id", db.FromUUID(c.ID), "error", err)
	}
	var goals []string
	if err := json.Unmarshal(c.Goals, &goals); err != nil {
		slog.Warn("unmarshal goals for character", "id", db.FromUUID(c.ID), "error", err)
	}
	var rel map[string]string
	if err := json.Unmarshal(c.Relationships, &rel); err != nil {
		slog.Warn("unmarshal relationships for character", "id", db.FromUUID(c.ID), "error", err)
	}
	var parentID *uuid.UUID
	if c.ParentID.Valid {
		p := db.FromUUID(c.ParentID)
		parentID = &p
	}
	return &canon.Character{
		ID:             db.FromUUID(c.ID),
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

// ── ActorService ────────────────────────────────────────────────

type ActorService interface {
	Create(ctx context.Context, name, gender, ethnicity, race, skinTone, eyeColor, hairColor, hairStyle, build, nationality string, heightCm, weightKg, age int, traits map[string]interface{}) (*canon.Actor, error)
	Get(ctx context.Context, id uuid.UUID) (*canon.Actor, error)
	Update(ctx context.Context, id uuid.UUID, name, gender, ethnicity, race, skinTone, eyeColor, hairColor, hairStyle, build, nationality string, heightCm, weightKg, age int, traits map[string]interface{}) (*canon.Actor, error)
	List(ctx context.Context) ([]canon.Actor, error)
}

type MemoryActorService struct {
	actors []canon.Actor
}

func NewMemoryActorService() *MemoryActorService {
	return &MemoryActorService{}
}

func (s *MemoryActorService) Create(ctx context.Context, name, gender, ethnicity, race, skinTone, eyeColor, hairColor, hairStyle, build, nationality string, heightCm, weightKg, age int, traits map[string]interface{}) (*canon.Actor, error) {
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

func (s *MemoryActorService) Get(ctx context.Context, id uuid.UUID) (*canon.Actor, error) {
	for i := range s.actors {
		if s.actors[i].ID == id {
			return &s.actors[i], nil
		}
	}
	return nil, fmt.Errorf("actor %s not found", id)
}

func (s *MemoryActorService) Update(ctx context.Context, id uuid.UUID, name, gender, ethnicity, race, skinTone, eyeColor, hairColor, hairStyle, build, nationality string, heightCm, weightKg, age int, traits map[string]interface{}) (*canon.Actor, error) {
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

func (s *MemoryActorService) List(ctx context.Context) ([]canon.Actor, error) {
	r := make([]canon.Actor, len(s.actors))
	copy(r, s.actors)
	return r, nil
}

type DBActorService struct {
	q *db.Queries
}

func NewDBActorService(q *db.Queries) *DBActorService {
	return &DBActorService{q: q}
}

func (s *DBActorService) Create(ctx context.Context, name, gender, ethnicity, race, skinTone, eyeColor, hairColor, hairStyle, build, nationality string, heightCm, weightKg, age int, traits map[string]interface{}) (*canon.Actor, error) {
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
		Traits:      db.JSONBytes(map[string]interface{}{}),
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

func (s *DBActorService) Get(ctx context.Context, id uuid.UUID) (*canon.Actor, error) {
	a, err := s.q.GetActor(ctx, db.ToUUID(id))
	if err != nil {
		return nil, err
	}
	loadedTraits, err := s.loadActorTraits(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	return toDomainActor(a, loadedTraits), nil
}

func (s *DBActorService) Update(ctx context.Context, id uuid.UUID, name, gender, ethnicity, race, skinTone, eyeColor, hairColor, hairStyle, build, nationality string, heightCm, weightKg, age int, traits map[string]interface{}) (*canon.Actor, error) {
	if traits == nil {
		traits = make(map[string]interface{})
	}
	a, err := s.q.UpdateActor(ctx, db.UpdateActorParams{
		ID:          db.ToUUID(id),
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
		Column15:    db.JSONBytes(map[string]interface{}{}),
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

func (s *DBActorService) List(ctx context.Context) ([]canon.Actor, error) {
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

func (s *DBActorService) persistActorTraits(ctx context.Context, actorID pgtype.UUID, traits map[string]interface{}) error {
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

func (s *DBActorService) loadActorTraits(ctx context.Context, actorID pgtype.UUID) (map[string]interface{}, error) {
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
		ID:          db.FromUUID(a.ID),
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

// ── TraitService ────────────────────────────────────────────────

type TraitService interface {
	Create(ctx context.Context, name, category, description string) (*canon.CharacterTrait, error)
	Get(ctx context.Context, id uuid.UUID) (*canon.CharacterTrait, error)
	List(ctx context.Context) ([]canon.CharacterTrait, error)
	Assign(ctx context.Context, characterID, traitID uuid.UUID, intensity int, note string) error
	Unassign(ctx context.Context, characterID, traitID uuid.UUID) error
	GetAssignments(ctx context.Context, characterID uuid.UUID) ([]canon.TraitAssignment, error)
}

type MemoryTraitService struct {
	traits      []canon.CharacterTrait
	assignments []canon.TraitAssignment
}

func NewMemoryTraitService() *MemoryTraitService {
	return &MemoryTraitService{}
}

func (s *MemoryTraitService) Create(ctx context.Context, name, category, description string) (*canon.CharacterTrait, error) {
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

func (s *MemoryTraitService) Get(ctx context.Context, id uuid.UUID) (*canon.CharacterTrait, error) {
	for i := range s.traits {
		if s.traits[i].ID == id {
			return &s.traits[i], nil
		}
	}
	return nil, fmt.Errorf("trait %s not found", id)
}

func (s *MemoryTraitService) List(ctx context.Context) ([]canon.CharacterTrait, error) {
	r := make([]canon.CharacterTrait, len(s.traits))
	copy(r, s.traits)
	return r, nil
}

func (s *MemoryTraitService) Assign(ctx context.Context, characterID, traitID uuid.UUID, intensity int, note string) error {
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

func (s *MemoryTraitService) Unassign(ctx context.Context, characterID, traitID uuid.UUID) error {
	for i := range s.assignments {
		if s.assignments[i].CharacterID == characterID && s.assignments[i].TraitID == traitID {
			s.assignments = append(s.assignments[:i], s.assignments[i+1:]...)
			return nil
		}
	}
	return nil
}

func (s *MemoryTraitService) GetAssignments(ctx context.Context, characterID uuid.UUID) ([]canon.TraitAssignment, error) {
	var result []canon.TraitAssignment
	for _, a := range s.assignments {
		if a.CharacterID == characterID {
			result = append(result, a)
		}
	}
	return result, nil
}

type DBTraitService struct {
	q *db.Queries
}

func NewDBTraitService(q *db.Queries) *DBTraitService {
	return &DBTraitService{q: q}
}

func (s *DBTraitService) Create(ctx context.Context, name, category, description string) (*canon.CharacterTrait, error) {
	t, err := s.q.CreateCharacterTrait(ctx, db.CreateCharacterTraitParams{
		Name:        name,
		Category:    category,
		Description: description,
	})
	if err != nil {
		return nil, err
	}
	return &canon.CharacterTrait{
		ID:          db.FromUUID(t.ID),
		Name:        t.Name,
		Category:    t.Category,
		Description: t.Description,
		CreatedAt:   t.CreatedAt.Time,
	}, nil
}

func (s *DBTraitService) Get(ctx context.Context, id uuid.UUID) (*canon.CharacterTrait, error) {
	t, err := s.q.GetCharacterTrait(ctx, db.ToUUID(id))
	if err != nil {
		return nil, err
	}
	return &canon.CharacterTrait{
		ID:          db.FromUUID(t.ID),
		Name:        t.Name,
		Category:    t.Category,
		Description: t.Description,
		CreatedAt:   t.CreatedAt.Time,
	}, nil
}

func (s *DBTraitService) List(ctx context.Context) ([]canon.CharacterTrait, error) {
	traits, err := s.q.ListCharacterTraits(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]canon.CharacterTrait, len(traits))
	for i, t := range traits {
		result[i] = canon.CharacterTrait{
			ID:          db.FromUUID(t.ID),
			Name:        t.Name,
			Category:    t.Category,
			Description: t.Description,
			CreatedAt:   t.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *DBTraitService) Assign(ctx context.Context, characterID, traitID uuid.UUID, intensity int, note string) error {
	return s.q.AssignTrait(ctx, db.AssignTraitParams{
		CharacterID: db.ToUUID(characterID),
		TraitID:     db.ToUUID(traitID),
		Intensity:   int32(intensity),
		Note:        note,
	})
}

func (s *DBTraitService) Unassign(ctx context.Context, characterID, traitID uuid.UUID) error {
	return s.q.UnassignTrait(ctx, db.UnassignTraitParams{
		CharacterID: db.ToUUID(characterID),
		TraitID:     db.ToUUID(traitID),
	})
}

func (s *DBTraitService) GetAssignments(ctx context.Context, characterID uuid.UUID) ([]canon.TraitAssignment, error) {
	rows, err := s.q.GetTraitAssignments(ctx, db.ToUUID(characterID))
	if err != nil {
		return nil, err
	}
	result := make([]canon.TraitAssignment, len(rows))
	for i, r := range rows {
		result[i] = canon.TraitAssignment{
			CharacterID: db.FromUUID(r.CharacterID),
			TraitID:     db.FromUUID(r.TraitID),
			Intensity:   int(r.Intensity),
			Note:        r.Note,
		}
	}
	return result, nil
}

// ── CastingService ──────────────────────────────────────────────

type CastingService interface {
	Create(ctx context.Context, storyID, actorID, characterID uuid.UUID, roleType string) (*canon.Casting, error)
	GetForStory(ctx context.Context, storyID uuid.UUID) ([]canon.Casting, error)
	GetForCharacter(ctx context.Context, characterID uuid.UUID) ([]canon.Casting, error)
	GetForActor(ctx context.Context, actorID uuid.UUID) ([]canon.Casting, error)
}

type MemoryCastingService struct {
	casts []canon.Casting
}

func NewMemoryCastingService() *MemoryCastingService {
	return &MemoryCastingService{}
}

func (s *MemoryCastingService) Create(ctx context.Context, storyID, actorID, characterID uuid.UUID, roleType string) (*canon.Casting, error) {
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

func (s *MemoryCastingService) GetForStory(ctx context.Context, storyID uuid.UUID) ([]canon.Casting, error) {
	var result []canon.Casting
	for _, c := range s.casts {
		if c.StoryID == storyID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (s *MemoryCastingService) GetForCharacter(ctx context.Context, characterID uuid.UUID) ([]canon.Casting, error) {
	var result []canon.Casting
	for _, c := range s.casts {
		if c.CharacterID == characterID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (s *MemoryCastingService) GetForActor(ctx context.Context, actorID uuid.UUID) ([]canon.Casting, error) {
	var result []canon.Casting
	for _, c := range s.casts {
		if c.ActorID == actorID {
			result = append(result, c)
		}
	}
	return result, nil
}

type DBCastingService struct {
	q *db.Queries
}

func NewDBCastingService(q *db.Queries) *DBCastingService {
	return &DBCastingService{q: q}
}

func (s *DBCastingService) Create(ctx context.Context, storyID, actorID, characterID uuid.UUID, roleType string) (*canon.Casting, error) {
	c, err := s.q.CreateCasting(ctx, db.CreateCastingParams{
		StoryID:     db.ToUUID(storyID),
		ActorID:     db.ToUUID(actorID),
		CharacterID: db.ToUUID(characterID),
		RoleType:    roleType,
	})
	if err != nil {
		return nil, err
	}
	return &canon.Casting{
		ID:          db.FromUUID(c.ID),
		StoryID:     db.FromUUID(c.StoryID),
		ActorID:     db.FromUUID(c.ActorID),
		CharacterID: db.FromUUID(c.CharacterID),
		RoleType:    c.RoleType,
		CreatedAt:   c.CreatedAt.Time,
	}, nil
}

func (s *DBCastingService) GetForStory(ctx context.Context, storyID uuid.UUID) ([]canon.Casting, error) {
	rows, err := s.q.ListCastingForStory(ctx, db.ToUUID(storyID))
	if err != nil {
		return nil, err
	}
	result := make([]canon.Casting, len(rows))
	for i, r := range rows {
		result[i] = canon.Casting{
			ID:          db.FromUUID(r.ID),
			StoryID:     db.FromUUID(r.StoryID),
			ActorID:     db.FromUUID(r.ActorID),
			CharacterID: db.FromUUID(r.CharacterID),
			RoleType:    r.RoleType,
			CreatedAt:   r.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *DBCastingService) GetForCharacter(ctx context.Context, characterID uuid.UUID) ([]canon.Casting, error) {
	rows, err := s.q.ListCastingForCharacter(ctx, db.ToUUID(characterID))
	if err != nil {
		return nil, err
	}
	result := make([]canon.Casting, len(rows))
	for i, r := range rows {
		result[i] = canon.Casting{
			ID:          db.FromUUID(r.ID),
			StoryID:     db.FromUUID(r.StoryID),
			ActorID:     db.FromUUID(r.ActorID),
			CharacterID: db.FromUUID(r.CharacterID),
			RoleType:    r.RoleType,
			CreatedAt:   r.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *DBCastingService) GetForActor(ctx context.Context, actorID uuid.UUID) ([]canon.Casting, error) {
	rows, err := s.q.ListCastingForActor(ctx, db.ToUUID(actorID))
	if err != nil {
		return nil, err
	}
	result := make([]canon.Casting, len(rows))
	for i, r := range rows {
		result[i] = canon.Casting{
			ID:          db.FromUUID(r.ID),
			StoryID:     db.FromUUID(r.StoryID),
			ActorID:     db.FromUUID(r.ActorID),
			CharacterID: db.FromUUID(r.CharacterID),
			RoleType:    r.RoleType,
			CreatedAt:   r.CreatedAt.Time,
		}
	}
	return result, nil
}

// ── LocationService ─────────────────────────────────────────────

type LocationService interface {
	Create(ctx context.Context, name, description string, props []string) (*canon.Location, error)
	Get(ctx context.Context, id uuid.UUID, version int) (*canon.Location, error)
	Update(ctx context.Context, id uuid.UUID, description string, props []string) (*canon.Location, error)
	List(ctx context.Context) ([]canon.Location, error)
}

type MemoryLocationService struct {
	locs    []canon.Location
	version map[uuid.UUID]int
}

func NewMemoryLocationService() *MemoryLocationService {
	return &MemoryLocationService{version: make(map[uuid.UUID]int)}
}

func (s *MemoryLocationService) Create(ctx context.Context, name, description string, props []string) (*canon.Location, error) {
	l := canon.Location{
		ID:          uuid.New(),
		Version:     1,
		Name:        name,
		Description: description,
		Props:       props,
		CreatedAt:   time.Now(),
	}
	s.locs = append(s.locs, l)
	s.version[l.ID] = 2
	return &l, nil
}

func (s *MemoryLocationService) Get(ctx context.Context, id uuid.UUID, version int) (*canon.Location, error) {
	var latest *canon.Location
	for i := range s.locs {
		if s.locs[i].ID == id {
			if version > 0 && s.locs[i].Version == version {
				return &s.locs[i], nil
			}
			if latest == nil || s.locs[i].Version > latest.Version {
				latest = &s.locs[i]
			}
		}
	}
	if latest == nil {
		return nil, fmt.Errorf("location %s not found", id)
	}
	return latest, nil
}

func (s *MemoryLocationService) Update(ctx context.Context, id uuid.UUID, description string, props []string) (*canon.Location, error) {
	next := s.version[id]
	if next == 0 {
		return nil, fmt.Errorf("location %s not found", id)
	}
	var curName string
	for i := len(s.locs) - 1; i >= 0; i-- {
		if s.locs[i].ID == id {
			curName = s.locs[i].Name
			break
		}
	}
	l := canon.Location{
		ID:          id,
		Version:     next,
		Name:        curName,
		Description: description,
		Props:       props,
		CreatedAt:   time.Now(),
	}
	s.locs = append(s.locs, l)
	s.version[id] = next + 1
	return &l, nil
}

func (s *MemoryLocationService) List(ctx context.Context) ([]canon.Location, error) {
	latest := make(map[uuid.UUID]canon.Location)
	for _, l := range s.locs {
		if existing, ok := latest[l.ID]; !ok || l.Version > existing.Version {
			latest[l.ID] = l
		}
	}
	result := make([]canon.Location, 0, len(latest))
	for _, l := range latest {
		result = append(result, l)
	}
	return result, nil
}

type DBLocationService struct {
	q *db.Queries
}

func NewDBLocationService(q *db.Queries) *DBLocationService {
	return &DBLocationService{q: q}
}

func (s *DBLocationService) Create(ctx context.Context, name, description string, props []string) (*canon.Location, error) {
	if props == nil {
		props = []string{}
	}
	l, err := s.q.CreateLocation(ctx, db.CreateLocationParams{
		Name:        name,
		Description: description,
		Props:       db.JSONBytes(props),
	})
	if err != nil {
		return nil, err
	}
	return toDomainLoc(l), nil
}

func (s *DBLocationService) Get(ctx context.Context, id uuid.UUID, version int) (*canon.Location, error) {
	if version > 0 {
		l, err := s.q.GetLocationAtVersion(ctx, db.GetLocationAtVersionParams{
			ID:      db.ToUUID(id),
			Version: int32(version),
		})
		if err != nil {
			return nil, err
		}
		return toDomainLoc(l), nil
	}
	l, err := s.q.GetLocationLatest(ctx, db.ToUUID(id))
	if err != nil {
		return nil, err
	}
	return toDomainLocFromLatest(l), nil
}

func (s *DBLocationService) Update(ctx context.Context, id uuid.UUID, description string, props []string) (*canon.Location, error) {
	l, err := s.q.UpdateLocation(ctx, db.UpdateLocationParams{
		ID:          db.ToUUID(id),
		Description: description,
		Column3:     db.JSONBytes(props),
	})
	if err != nil {
		return nil, err
	}
	return toDomainLoc(l), nil
}

func (s *DBLocationService) List(ctx context.Context) ([]canon.Location, error) {
	locs, err := s.q.ListLocations(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]canon.Location, len(locs))
	for i, l := range locs {
		result[i] = *toDomainLocFromLatest(l)
	}
	return result, nil
}

func toDomainLoc(l db.Location) *canon.Location {
	var props []string
	json.Unmarshal(l.Props, &props)
	return &canon.Location{
		ID:          db.FromUUID(l.ID),
		Version:     int(l.Version),
		Name:        l.Name,
		Description: l.Description,
		Props:       props,
		CreatedAt:   l.CreatedAt.Time,
	}
}

func toDomainLocFromLatest(l db.LatestLocation) *canon.Location {
	var props []string
	json.Unmarshal(l.Props, &props)
	return &canon.Location{
		ID:          db.FromUUID(l.ID),
		Version:     int(l.Version),
		Name:        l.Name,
		Description: l.Description,
		Props:       props,
		CreatedAt:   l.CreatedAt.Time,
	}
}

// ── LoreService ─────────────────────────────────────────────────

type LoreService interface {
	Create(ctx context.Context, tags []string, content string) (*canon.Lore, error)
	List(ctx context.Context) ([]canon.Lore, error)
	SearchByTags(ctx context.Context, tags []string) ([]canon.Lore, error)
	SearchSimilar(ctx context.Context, embedding []float32, limit int) ([]canon.Lore, error)
}

type MemoryLoreService struct {
	items []canon.Lore
}

func NewMemoryLoreService() *MemoryLoreService {
	return &MemoryLoreService{}
}

func (s *MemoryLoreService) Create(ctx context.Context, tags []string, content string) (*canon.Lore, error) {
	l := canon.Lore{
		ID:        uuid.New(),
		Tags:      tags,
		Content:   content,
		CreatedAt: time.Now(),
	}
	s.items = append(s.items, l)
	return &l, nil
}

func (s *MemoryLoreService) List(ctx context.Context) ([]canon.Lore, error) {
	r := make([]canon.Lore, len(s.items))
	copy(r, s.items)
	return r, nil
}

func (s *MemoryLoreService) SearchByTags(ctx context.Context, tags []string) ([]canon.Lore, error) {
	tagSet := make(map[string]bool, len(tags))
	for _, t := range tags {
		tagSet[t] = true
	}
	var result []canon.Lore
	for _, l := range s.items {
		for _, t := range l.Tags {
			if tagSet[t] {
				result = append(result, l)
				break
			}
		}
	}
	return result, nil
}

func (s *MemoryLoreService) SearchSimilar(ctx context.Context, embedding []float32, limit int) ([]canon.Lore, error) {
	if limit > len(s.items) {
		limit = len(s.items)
	}
	return s.items[:limit], nil
}

type DBLoreService struct {
	q *db.Queries
}

func NewDBLoreService(q *db.Queries) *DBLoreService {
	return &DBLoreService{q: q}
}

func (s *DBLoreService) Create(ctx context.Context, tags []string, content string) (*canon.Lore, error) {
	if tags == nil {
		tags = []string{}
	}
	l, err := s.q.CreateLore(ctx, db.CreateLoreParams{
		Tags:      tags,
		Content:   content,
		Embedding: pgvector.Vector{},
	})
	if err != nil {
		return nil, err
	}
	return &canon.Lore{
		ID:        db.FromUUID(l.ID),
		Tags:      l.Tags,
		Content:   l.Content,
		CreatedAt: l.CreatedAt.Time,
	}, nil
}

func (s *DBLoreService) List(ctx context.Context) ([]canon.Lore, error) {
	items, err := s.q.ListLore(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]canon.Lore, len(items))
	for i, l := range items {
		result[i] = canon.Lore{
			ID:        db.FromUUID(l.ID),
			Tags:      l.Tags,
			Content:   l.Content,
			CreatedAt: l.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *DBLoreService) SearchByTags(ctx context.Context, tags []string) ([]canon.Lore, error) {
	items, err := s.q.SearchLoreByTags(ctx, tags)
	if err != nil {
		return nil, err
	}
	result := make([]canon.Lore, len(items))
	for i, l := range items {
		result[i] = canon.Lore{
			ID:        db.FromUUID(l.ID),
			Tags:      l.Tags,
			Content:   l.Content,
			CreatedAt: l.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *DBLoreService) SearchSimilar(ctx context.Context, embedding []float32, limit int) ([]canon.Lore, error) {
	vec := pgvector.NewVector(embedding)
	items, err := s.q.SearchLoreSimilar(ctx, db.SearchLoreSimilarParams{
		Column1: vec,
		Limit:   int32(limit),
	})
	if err != nil {
		return nil, err
	}
	result := make([]canon.Lore, len(items))
	for i, l := range items {
		result[i] = canon.Lore{
			ID:        db.FromUUID(l.ID),
			Tags:      l.Tags,
			Content:   l.Content,
			CreatedAt: l.CreatedAt.Time,
		}
	}
	return result, nil
}
