package canon

import (
	"time"

	"github.com/google/uuid"
)

type Character struct {
	ID             uuid.UUID         `json:"id"`
	Version        int               `json:"version"`
	Name           string            `json:"name"`
	Persona        string            `json:"persona,omitempty"`
	Backstory      string            `json:"backstory,omitempty"`
	MoralAlignment string            `json:"moral_alignment,omitempty"`
	Personality    []string          `json:"personality,omitempty"`
	Flaws          []string          `json:"flaws,omitempty"`
	Goals          []string          `json:"goals,omitempty"`
	Traits         []string          `json:"traits"`
	VoiceSamples   []string          `json:"voice_samples"`
	ParentID       *uuid.UUID        `json:"parent_id,omitempty"`
	Relationships  map[string]string `json:"relationships"`
	CreatedAt      time.Time         `json:"created_at"`
}

type Location struct {
	ID          uuid.UUID `json:"id"`
	Version     int       `json:"version"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Props       []string  `json:"props,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Lore struct {
	ID        uuid.UUID `json:"id"`
	Tags      []string  `json:"tags"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Card struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Type          string            `json:"type"`
	Traits        []string          `json:"traits,omitempty"`
	VoiceSamples  []string          `json:"voice_samples,omitempty"`
	Props         []string          `json:"props,omitempty"`
	Relationships map[string]string `json:"relationships,omitempty"`
}

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

type StoryBible struct {
	StoryID        uuid.UUID        `json:"story_id"`
	WorldRules     map[string]string `json:"world_rules"`
	CanonRules     map[string]string `json:"canon_rules"`
	ForbiddenRules []string          `json:"forbidden_rules"`
	Technology     map[string]string `json:"technology_rules"`
	Magic          map[string]string `json:"magic_rules"`
	Geography      map[string]string `json:"geography,omitempty"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

type ValidationResult struct {
	Valid      bool              `json:"valid"`
	Violations []Violation       `json:"violations,omitempty"`
	Warnings   []string          `json:"warnings,omitempty"`
}

type Violation struct {
	Rule    string `json:"rule"`
	Type    string `json:"type"`
	Detail  string `json:"detail"`
	Severity string `json:"severity"`
}

type ValidatorService interface {
	ValidateScene(storyID, sceneID uuid.UUID, sceneText string, charIDs []uuid.UUID) (*ValidationResult, error)
	ValidateDialogue(storyID uuid.UUID, speaker, target uuid.UUID, dialogue string) (*ValidationResult, error)
	ValidateTimeline(storyID uuid.UUID) (*ValidationResult, error)
	ValidateCharacterAction(storyID, charID uuid.UUID, action string) (*ValidationResult, error)
	GetBible(storyID uuid.UUID) (*StoryBible, error)
	UpsertBible(bible *StoryBible) error
}

type Validator struct {
	bibles map[uuid.UUID]*StoryBible
}

func NewValidator() *Validator {
	return &Validator{
		bibles: make(map[uuid.UUID]*StoryBible),
	}
}

func (v *Validator) ValidateScene(storyID, sceneID uuid.UUID, sceneText string, charIDs []uuid.UUID) (*ValidationResult, error) {
	result := &ValidationResult{Valid: true}
	bible, ok := v.bibles[storyID]
	if !ok {
		return result, nil
	}
	for _, forbidden := range bible.ForbiddenRules {
		if contains(sceneText, forbidden) {
			result.Violations = append(result.Violations, Violation{
				Rule:     "forbidden_content",
				Type:     "canon",
				Detail:   forbidden,
				Severity: "error",
			})
		}
	}
	for rule, desc := range bible.CanonRules {
		if !contains(sceneText, desc) && !contains(desc, rule) {
			result.Warnings = append(result.Warnings, "canon rule '"+rule+"' may not be satisfied")
		}
	}
	if len(result.Violations) > 0 {
		result.Valid = false
	}
	return result, nil
}

func (v *Validator) ValidateDialogue(storyID uuid.UUID, speaker, target uuid.UUID, dialogue string) (*ValidationResult, error) {
	return &ValidationResult{Valid: true}, nil
}

func (v *Validator) ValidateTimeline(storyID uuid.UUID) (*ValidationResult, error) {
	return &ValidationResult{Valid: true}, nil
}

func (v *Validator) ValidateCharacterAction(storyID, charID uuid.UUID, action string) (*ValidationResult, error) {
	return &ValidationResult{Valid: true}, nil
}

func (v *Validator) GetBible(storyID uuid.UUID) (*StoryBible, error) {
	b, ok := v.bibles[storyID]
	if !ok {
		return &StoryBible{StoryID: storyID, WorldRules: make(map[string]string)}, nil
	}
	return b, nil
}

func (v *Validator) UpsertBible(bible *StoryBible) error {
	bible.UpdatedAt = time.Now()
	if bible.WorldRules == nil {
		bible.WorldRules = make(map[string]string)
	}
	if bible.CanonRules == nil {
		bible.CanonRules = make(map[string]string)
	}
	if bible.Technology == nil {
		bible.Technology = make(map[string]string)
	}
	if bible.Magic == nil {
		bible.Magic = make(map[string]string)
	}
	v.bibles[bible.StoryID] = bible
	return nil
}

func contains(s, substr string) bool {
	return len(substr) > 0 && s != "" && findCaseInsensitive(s, substr)
}

func findCaseInsensitive(s, substr string) bool {
	s, substr = toLower(s), toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 32
		}
		b = append(b, c)
	}
	return string(b)
}
