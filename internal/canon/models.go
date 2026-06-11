package canon

import (
	"time"

	"github.com/google/uuid"
)

type Character struct {
	ID           uuid.UUID `json:"id"`
	Version      int       `json:"version"`
	Name         string    `json:"name"`
	Traits       []string  `json:"traits"`
	VoiceSamples []string  `json:"voice_samples"`
	Relationships map[string]string `json:"relationships"`
	CreatedAt    time.Time `json:"created_at"`
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

type CharacterService interface {
	Create(name string, traits []string, voiceSamples []string, relationships map[string]string) (*Character, error)
	Get(id uuid.UUID, version int) (*Character, error)
	GetLatest(id uuid.UUID) (*Character, error)
	Update(id uuid.UUID, traits []string, voiceSamples []string, relationships map[string]string) (*Character, error)
	List() ([]Character, error)
}

type LocationService interface {
	Create(name, description string, props []string) (*Location, error)
	Get(id uuid.UUID, version int) (*Location, error)
	GetLatest(id uuid.UUID) (*Location, error)
	Update(id uuid.UUID, description string, props []string) (*Location, error)
	List() ([]Location, error)
}

type LoreService interface {
	Create(tags []string, content string) (*Lore, error)
	SearchByTags(tags []string) ([]Lore, error)
	SearchSimilar(embedding []float32, limit int) ([]Lore, error)
}
