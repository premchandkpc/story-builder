package canon

import (
	"context"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/canon"
)

type CanonService interface {
	CreateCharacter(ctx context.Context, spec CreateCharacterSpec) (*canon.Character, error)
	GetCharacter(ctx context.Context, id uuid.UUID, version int) (*canon.Character, error)
	GetCharacterLatest(ctx context.Context, id uuid.UUID) (*canon.Character, error)
	UpdateCharacter(ctx context.Context, id uuid.UUID, spec UpdateCharacterSpec) (*canon.Character, error)
	ListCharacters(ctx context.Context) ([]canon.Character, error)
	CreateLocation(ctx context.Context, spec CreateLocationSpec) (*canon.Location, error)
	GetLocation(ctx context.Context, id uuid.UUID, version int) (*canon.Location, error)
	GetLocationLatest(ctx context.Context, id uuid.UUID) (*canon.Location, error)
	UpdateLocation(ctx context.Context, id uuid.UUID, spec UpdateLocationSpec) (*canon.Location, error)
	ListLocations(ctx context.Context) ([]canon.Location, error)
	CreateLore(ctx context.Context, tags []string, content string) (*canon.Lore, error)
	SearchLoreByTags(ctx context.Context, tags []string) ([]canon.Lore, error)
	SearchLoreSimilar(ctx context.Context, embedding []float32, limit int) ([]canon.Lore, error)
	CreateActor(ctx context.Context, spec CreateActorSpec) (*canon.Actor, error)
	GetActor(ctx context.Context, id uuid.UUID) (*canon.Actor, error)
	UpdateActor(ctx context.Context, id uuid.UUID, spec UpdateActorSpec) (*canon.Actor, error)
	ListActors(ctx context.Context) ([]canon.Actor, error)
	CreateCasting(ctx context.Context, storyID, actorID, characterID uuid.UUID, roleType string) (*canon.Casting, error)
	GetCastingForStory(ctx context.Context, storyID uuid.UUID) ([]canon.Casting, error)
	GetCastingForCharacter(ctx context.Context, characterID uuid.UUID) ([]canon.Casting, error)
	GetCastingForActor(ctx context.Context, actorID uuid.UUID) ([]canon.Casting, error)
	CreateTrait(ctx context.Context, name, category, description string) (*canon.CharacterTrait, error)
	GetTrait(ctx context.Context, id uuid.UUID) (*canon.CharacterTrait, error)
	ListTraits(ctx context.Context) ([]canon.CharacterTrait, error)
	AssignTrait(ctx context.Context, characterID, traitID uuid.UUID, intensity int, note string) error
	UnassignTrait(ctx context.Context, characterID, traitID uuid.UUID) error
	GetTraitAssignments(ctx context.Context, characterID uuid.UUID) ([]canon.TraitAssignment, error)
}

type CreateCharacterSpec struct {
	Name           string
	Persona        string
	Backstory      string
	MoralAlignment string
	Personality    []string
	Flaws          []string
	Goals          []string
	Traits         []string
	VoiceSamples   []string
	ParentID       *uuid.UUID
	Relationships  map[string]string
}

type UpdateCharacterSpec struct {
	Name           string
	Persona        string
	Backstory      string
	MoralAlignment string
	Personality    []string
	Flaws          []string
	Goals          []string
	Traits         []string
	VoiceSamples   []string
	ParentID       *uuid.UUID
	Relationships  map[string]string
}

type CreateLocationSpec struct {
	Name        string
	Description string
	Props       []string
}

type UpdateLocationSpec struct {
	Description string
	Props       []string
}

type CreateActorSpec struct {
	Name        string
	Gender      string
	Ethnicity   string
	Race        string
	SkinTone    string
	EyeColor    string
	HairColor   string
	HairStyle   string
	Build       string
	Nationality string
	HeightCm    int
	WeightKg    int
	Age         int
	Traits      map[string]interface{}
}

type UpdateActorSpec struct {
	Name        string
	Gender      string
	Ethnicity   string
	Race        string
	SkinTone    string
	EyeColor    string
	HairColor   string
	HairStyle   string
	Build       string
	Nationality string
	HeightCm    int
	WeightKg    int
	Age         int
	Traits      map[string]interface{}
}

type CanonRepository interface {
	CreateCharacter(ctx context.Context, spec CreateCharacterSpec) (*canon.Character, error)
	GetCharacter(ctx context.Context, id uuid.UUID, version int) (*canon.Character, error)
	GetCharacterLatest(ctx context.Context, id uuid.UUID) (*canon.Character, error)
	UpdateCharacter(ctx context.Context, id uuid.UUID, spec UpdateCharacterSpec) (*canon.Character, error)
	ListCharacters(ctx context.Context) ([]canon.Character, error)
	CreateLocation(ctx context.Context, spec CreateLocationSpec) (*canon.Location, error)
	GetLocation(ctx context.Context, id uuid.UUID, version int) (*canon.Location, error)
	GetLocationLatest(ctx context.Context, id uuid.UUID) (*canon.Location, error)
	UpdateLocation(ctx context.Context, id uuid.UUID, spec UpdateLocationSpec) (*canon.Location, error)
	ListLocations(ctx context.Context) ([]canon.Location, error)
	CreateLore(ctx context.Context, tags []string, content string) (*canon.Lore, error)
	SearchLoreByTags(ctx context.Context, tags []string) ([]canon.Lore, error)
	SearchLoreSimilar(ctx context.Context, embedding []float32, limit int) ([]canon.Lore, error)
	CreateActor(ctx context.Context, spec CreateActorSpec) (*canon.Actor, error)
	GetActor(ctx context.Context, id uuid.UUID) (*canon.Actor, error)
	UpdateActor(ctx context.Context, id uuid.UUID, spec UpdateActorSpec) (*canon.Actor, error)
	ListActors(ctx context.Context) ([]canon.Actor, error)
	CreateCasting(ctx context.Context, storyID, actorID, characterID uuid.UUID, roleType string) (*canon.Casting, error)
	GetCastingForStory(ctx context.Context, storyID uuid.UUID) ([]canon.Casting, error)
	GetCastingForCharacter(ctx context.Context, characterID uuid.UUID) ([]canon.Casting, error)
	GetCastingForActor(ctx context.Context, actorID uuid.UUID) ([]canon.Casting, error)
	CreateTrait(ctx context.Context, name, category, description string) (*canon.CharacterTrait, error)
	GetTrait(ctx context.Context, id uuid.UUID) (*canon.CharacterTrait, error)
	ListTraits(ctx context.Context) ([]canon.CharacterTrait, error)
	AssignTrait(ctx context.Context, characterID, traitID uuid.UUID, intensity int, note string) error
	UnassignTrait(ctx context.Context, characterID, traitID uuid.UUID) error
	GetTraitAssignments(ctx context.Context, characterID uuid.UUID) ([]canon.TraitAssignment, error)
}
