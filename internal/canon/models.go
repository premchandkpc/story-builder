package canon

import (
	"time"

	"github.com/google/uuid"
)

type Actor struct {
	ID          uuid.UUID              `json:"id"`
	Name        string                 `json:"name"`
	Gender      string                 `json:"gender"`
	Ethnicity   string                 `json:"ethnicity"`
	Race        string                 `json:"race"`
	SkinTone    string                 `json:"skin_tone"`
	EyeColor    string                 `json:"eye_color"`
	HairColor   string                 `json:"hair_color"`
	HairStyle   string                 `json:"hair_style"`
	Build       string                 `json:"build"`
	HeightCm    int                    `json:"height_cm"`
	WeightKg    int                    `json:"weight_kg"`
	Age         int                    `json:"age"`
	Nationality string                 `json:"nationality"`
	Traits      map[string]interface{} `json:"traits"`
	CreatedAt   time.Time              `json:"created_at"`
}

type Character struct {
	ID              uuid.UUID      `json:"id"`
	Version         int            `json:"version"`
	Name            string         `json:"name"`
	Persona         string         `json:"persona"`
	Backstory       string         `json:"backstory"`
	MoralAlignment  string         `json:"moral_alignment"`
	Personality     []string       `json:"personality"`
	Flaws           []string       `json:"flaws"`
	Goals           []string       `json:"goals"`
	VoiceSamples    []string       `json:"voice_samples"`
	ParentID        *uuid.UUID     `json:"parent_id,omitempty"`
	Traits          []string       `json:"traits"`
	Relationships   map[string]string `json:"relationships"`
	CreatedAt       time.Time      `json:"created_at"`
}

type CharacterTrait struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type TraitAssignment struct {
	CharacterID uuid.UUID `json:"character_id"`
	TraitID     uuid.UUID `json:"trait_id"`
	Intensity   int       `json:"intensity"`
	Note        string    `json:"note"`
}

type Casting struct {
	ID          uuid.UUID `json:"id"`
	StoryID     uuid.UUID `json:"story_id"`
	ActorID     uuid.UUID `json:"actor_id"`
	CharacterID uuid.UUID `json:"character_id"`
	RoleType    string    `json:"role_type"`
	CreatedAt   time.Time `json:"created_at"`
}

type Location struct {
	ID          uuid.UUID `json:"id"`
	Version     int       `json:"version"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Props       []string  `json:"props"`
	CreatedAt   time.Time `json:"created_at"`
}

type Lore struct {
	ID        uuid.UUID `json:"id"`
	Tags      []string  `json:"tags"`
	Content   string    `json:"content"`
	Embedding []float32 `json:"embedding,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type CanonPin struct {
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	Version    int       `json:"version"`
}

type StoryCanonPins struct {
	StoryID   uuid.UUID            `json:"story_id"`
	Pins      map[string]CanonPin  `json:"pins"`
}

type Card struct {
	Type         string            `json:"type"`
	Name         string            `json:"name"`
	Traits       []string          `json:"traits,omitempty"`
	Description  string            `json:"description,omitempty"`
	VoiceSamples []string          `json:"voice_samples,omitempty"`
	Relationships map[string]string `json:"relationships,omitempty"`
	Props        []string          `json:"props,omitempty"`
}


