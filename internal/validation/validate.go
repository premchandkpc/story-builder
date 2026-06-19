package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/premchand/story-builder/internal/domain"
)

type Violation struct {
	Severity string // "error" or "warning"
	Field    string
	Message  string
}

func (v Violation) Error() string {
	return fmt.Sprintf("[%s] %s: %s", v.Severity, v.Field, v.Message)
}

type PostGenerationCheck struct {
	CharacterID      string
	PreviousLocation string
	NewLocation      string
	Learned          []string
	PreviousKnowledge []string
}

type SceneValidator struct {
	charRepo interface {
		ListByStory(ctx context.Context, storyID string) ([]*domain.Character, error)
	}
	locRepo interface {
		ListByStory(ctx context.Context, storyID string) ([]*domain.Location, error)
		GetByName(ctx context.Context, storyID, name string) (*domain.Location, error)
	}
}

func NewSceneValidator(
	charRepo interface {
		ListByStory(ctx context.Context, storyID string) ([]*domain.Character, error)
	},
	locRepo interface {
		ListByStory(ctx context.Context, storyID string) ([]*domain.Location, error)
		GetByName(ctx context.Context, storyID, name string) (*domain.Location, error)
	},
) *SceneValidator {
	return &SceneValidator{charRepo: charRepo, locRepo: locRepo}
}

func (v *SceneValidator) ValidatePreGeneration(ctx context.Context, scene *domain.Scene) []Violation {
	var violations []Violation

	if scene == nil {
		violations = append(violations, Violation{Severity: "error", Field: "scene", Message: "scene is nil"})
		return violations
	}

	if len(scene.Participants) == 0 {
		violations = append(violations, Violation{Severity: "warning", Field: "participants", Message: "scene has no participants"})
	}

	if scene.BeatIntent == "" {
		violations = append(violations, Violation{Severity: "error", Field: "beatIntent", Message: "scene beat intent is required"})
	}

	if v.charRepo != nil {
		chars, err := v.charRepo.ListByStory(ctx, scene.StoryID)
		if err == nil {
			charNames := make(map[string]bool)
			for _, c := range chars {
				charNames[c.Name] = true
			}
			for _, p := range scene.Participants {
				if !charNames[p] && !strings.HasPrefix(p, "_") {
					violations = append(violations, Violation{Severity: "warning", Field: "participants", Message: fmt.Sprintf("participant %q not found in story characters", p)})
				}
			}
		}
	}

	if scene.LocationRef != "" && v.locRepo != nil {
		loc, err := v.locRepo.GetByName(ctx, scene.StoryID, scene.LocationRef)
		if err != nil || loc == nil {
			violations = append(violations, Violation{Severity: "warning", Field: "locationRef", Message: fmt.Sprintf("location %q not found", scene.LocationRef)})
		}
	}

	return violations
}

func (v *SceneValidator) ValidatePostGeneration(ctx context.Context, scene *domain.Scene, checks []PostGenerationCheck) []Violation {
	var violations []Violation

	if scene == nil {
		return violations
	}

	for _, c := range checks {
		// Location continuity: character was in a different location than the scene
		if c.PreviousLocation != "" && c.NewLocation != "" && scene.LocationRef != "" {
			if c.PreviousLocation != scene.LocationRef && c.PreviousLocation != c.NewLocation {
				violations = append(violations, Violation{
					Severity: "warning",
					Field:    "location_continuity",
					Message:  fmt.Sprintf("character %q moved from %q to %q (scene is at %q)", c.CharacterID, c.PreviousLocation, c.NewLocation, scene.LocationRef),
				})
			}
		}

		// Knowledge redundancy: character learned something they already knew
		if len(c.PreviousKnowledge) > 0 && len(c.Learned) > 0 {
			known := make(map[string]bool, len(c.PreviousKnowledge))
			for _, k := range c.PreviousKnowledge {
				known[strings.ToLower(k)] = true
			}
			for _, learned := range c.Learned {
				if known[strings.ToLower(learned)] {
					violations = append(violations, Violation{
						Severity: "warning",
						Field:    "knowledge_redundancy",
						Message:  fmt.Sprintf("character %q already knew: %q", c.CharacterID, learned),
					})
				}
			}
		}
	}

	return violations
}
