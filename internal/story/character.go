package story

import (
	"context"
	"fmt"
	"strings"
)

// CharacterDefinition contains author-owned facts, never inferred runtime state.
// Secrets belong to the author context; InitialKnowledge is what the character knows.
type CharacterDefinition struct {
	Role             string         `bson:"role" json:"role"`
	Pronouns         string         `bson:"pronouns" json:"pronouns"`
	Description      string         `bson:"description" json:"description"`
	Backstory        string         `bson:"backstory" json:"backstory"`
	Motivation       string         `bson:"motivation" json:"motivation"`
	Goals            []string       `bson:"goals" json:"goals"`
	Fears            []string       `bson:"fears" json:"fears"`
	Flaws            []string       `bson:"flaws" json:"flaws"`
	Values           []string       `bson:"values" json:"values"`
	Voice            string         `bson:"voice" json:"voice"`
	Constraints      []string       `bson:"constraints" json:"constraints"`
	Secrets          []string       `bson:"secrets" json:"secrets"`
	InitialKnowledge []string       `bson:"initialKnowledge" json:"initialKnowledge"`
	Relationships    []Relationship `bson:"relationships" json:"relationships"`
}
type Relationship struct {
	CharacterID string `bson:"characterId" json:"characterId"`
	Kind        string `bson:"kind" json:"kind"`
	Description string `bson:"description" json:"description"`
}
type CharacterInput struct {
	Name   string   `json:"name"`
	Traits []string `json:"traits"`
	CharacterDefinition
	ExpectedVersion int `json:"expectedVersion"`
}

func (s *Service) DefineCharacter(ctx context.Context, storyID, id string, input CharacterInput) (Character, error) {
	if _, err := s.repo.GetStory(ctx, storyID); err != nil {
		return Character{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 200 {
		return Character{}, fmt.Errorf("%w: name must contain 1–200 bytes", ErrInvalidCharacter)
	}
	if len(input.Traits) > 32 || len(input.Relationships) > 32 {
		return Character{}, ErrInvalidCharacter
	}
	for _, list := range [][]string{input.Traits, input.Goals, input.Fears, input.Flaws, input.Values, input.Constraints, input.Secrets, input.InitialKnowledge} {
		if len(list) > 32 {
			return Character{}, ErrInvalidCharacter
		}
		for _, item := range list {
			if strings.TrimSpace(item) == "" || len(item) > 2000 {
				return Character{}, ErrInvalidCharacter
			}
		}
	}
	for _, value := range []string{input.Role, input.Pronouns, input.Description, input.Backstory, input.Motivation, input.Voice} {
		if len(value) > 8000 {
			return Character{}, ErrInvalidCharacter
		}
	}
	characters, err := s.repo.ListCharacters(ctx, storyID)
	if err != nil {
		return Character{}, err
	}
	known := map[string]bool{}
	for _, c := range characters {
		known[c.CharacterID] = true
	}
	expected := 0
	if id == "" {
		id = s.newID()
	} else {
		if !known[id] {
			return Character{}, ErrNotFound
		}
		expected = input.ExpectedVersion
		if expected < 1 {
			return Character{}, ErrConflict
		}
	}
	seen := map[string]bool{}
	for _, relation := range input.Relationships {
		if relation.CharacterID == id || !known[relation.CharacterID] || seen[relation.CharacterID] || strings.TrimSpace(relation.Kind) == "" || len(relation.Kind) > 100 || len(relation.Description) > 2000 {
			return Character{}, fmt.Errorf("%w: relationships require a distinct character in this story and a kind", ErrInvalidCharacter)
		}
		seen[relation.CharacterID] = true
	}
	value := Character{CharacterID: id, Version: expected + 1, StoryID: storyID, Name: name, Traits: input.Traits, CharacterDefinition: input.CharacterDefinition, CreatedAt: s.now().UTC()}
	if err := s.repo.SaveCharacter(ctx, value, expected); err != nil {
		return Character{}, err
	}
	return value, nil
}
