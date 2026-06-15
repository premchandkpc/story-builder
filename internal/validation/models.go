package validation

import (
	"time"

	"github.com/google/uuid"
)

type CheckType string

const (
	CheckCanon         CheckType = "canon"
	CheckTimeline      CheckType = "timeline"
	CheckCharacter     CheckType = "character"
	CheckWorldRule     CheckType = "world_rule"
	CheckRelationship  CheckType = "relationship"
)

type Severity string

const (
	SevError   Severity = "error"
	SevWarning Severity = "warning"
	SevInfo    Severity = "info"
)

type ValidationCheck struct {
	ID        uuid.UUID  `json:"id"`
	SceneID   uuid.UUID  `json:"scene_id"`
	Type      CheckType  `json:"type"`
	Rule      string     `json:"rule"`
	Detail    string     `json:"detail"`
	Severity  Severity   `json:"severity"`
	Passed    bool       `json:"passed"`
	CreatedAt time.Time  `json:"created_at"`
}

type ValidationReport struct {
	SceneID   uuid.UUID         `json:"scene_id"`
	Passed    bool              `json:"passed"`
	Checks    []ValidationCheck `json:"checks"`
	Summary   string            `json:"summary"`
}

type ValidatorService interface {
	ValidateAgainstCanon(storyID, sceneID uuid.UUID, sceneText string, characters []uuid.UUID) (*ValidationReport, error)
	ValidateTimeline(storyID uuid.UUID) (*ValidationReport, error)
	ValidateWorldRules(storyID uuid.UUID, sceneText string, worldRules map[string]string) (*ValidationReport, error)
	ValidateCharacterBehavior(charID uuid.UUID, action string, personality map[string]float64) (*ValidationReport, error)
}

type Store interface {
	CreateCheck(check *ValidationCheck) error
	GetReport(sceneID uuid.UUID) ([]ValidationCheck, error)
	ListViolations(storyID uuid.UUID, severity Severity, limit int) ([]ValidationCheck, error)
}
