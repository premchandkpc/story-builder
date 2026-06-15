package validation

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/canon"
)

type MemoryStore struct {
	mu     sync.RWMutex
	checks []ValidationCheck
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (m *MemoryStore) CreateCheck(check *ValidationCheck) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	check.ID = uuid.New()
	check.CreatedAt = time.Now()
	m.checks = append(m.checks, *check)
	return nil
}

func (m *MemoryStore) GetReport(sceneID uuid.UUID) ([]ValidationCheck, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []ValidationCheck
	for _, c := range m.checks {
		if c.SceneID == sceneID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *MemoryStore) ListViolations(storyID uuid.UUID, severity Severity, limit int) ([]ValidationCheck, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []ValidationCheck
	for _, c := range m.checks {
		if !c.Passed && (severity == "" || c.Severity == severity) {
			result = append(result, c)
		}
	}
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

type CharacterGetter interface {
	GetLatest(id uuid.UUID) (*canon.Character, error)
	List() ([]canon.Character, error)
}

type validatorService struct {
	store Store
	chars CharacterGetter
}

func NewValidatorService(store Store, chars CharacterGetter) ValidatorService {
	return &validatorService{store: store, chars: chars}
}

func (v *validatorService) ValidateAgainstCanon(storyID, sceneID uuid.UUID, sceneText string, charIDs []uuid.UUID) (*ValidationReport, error) {
	report := &ValidationReport{SceneID: sceneID, Passed: true}
	lowerText := toLower(sceneText)

	for _, charID := range charIDs {
		def, err := v.chars.GetLatest(charID)
		if err != nil {
			continue
		}

		if !strings.Contains(lowerText, toLower(def.Name)) {
			check := ValidationCheck{
				SceneID: sceneID, Type: CheckCharacter,
				Rule:     "character_present",
				Detail:   fmt.Sprintf("Character %s does not appear in scene text", def.Name),
				Severity: SevWarning, Passed: false,
			}
			v.store.CreateCheck(&check)
			report.Checks = append(report.Checks, check)
			report.Passed = false
		}

		for _, vs := range def.VoiceSamples {
			if vs != "" && !strings.Contains(lowerText, toLower(vs)) {
				check := ValidationCheck{
					SceneID: sceneID, Type: CheckCharacter,
					Rule:     "voice_match",
					Detail:   fmt.Sprintf("Voice sample '%s' not found in scene for %s", vs, def.Name),
					Severity: SevInfo, Passed: true,
				}
				v.store.CreateCheck(&check)
			}
		}

		for _, rel := range def.Relationships {
			relName := rel
			if relName != "" && !strings.Contains(lowerText, toLower(relName)) {
				check := ValidationCheck{
					SceneID: sceneID, Type: CheckRelationship,
					Rule:     "relationship_referenced",
					Detail:   fmt.Sprintf("Relationship target '%s' not found in scene for %s", relName, def.Name),
					Severity: SevInfo, Passed: true,
				}
				v.store.CreateCheck(&check)
			}
		}
	}
	return report, nil
}

func (v *validatorService) ValidateTimeline(storyID uuid.UUID) (*ValidationReport, error) {
	return &ValidationReport{Passed: true}, nil
}

func (v *validatorService) ValidateWorldRules(storyID uuid.UUID, sceneText string, worldRules map[string]string) (*ValidationReport, error) {
	report := &ValidationReport{Passed: true}
	for rule, desc := range worldRules {
		check := ValidationCheck{
			Type:     CheckWorldRule,
			Rule:     rule,
			Detail:   desc,
			Severity: SevError,
			Passed:   true,
		}
		if desc != "" && !containsText(sceneText, desc) {
			check.Passed = false
			report.Passed = false
		}
		v.store.CreateCheck(&check)
		report.Checks = append(report.Checks, check)
	}
	return report, nil
}

func (v *validatorService) ValidateCharacterBehavior(charID uuid.UUID, action string, personality map[string]float64) (*ValidationReport, error) {
	report := &ValidationReport{Passed: true}

	def, err := v.chars.GetLatest(charID)
	if err != nil {
		return report, nil
	}

	lowerAction := toLower(action)

	if def.MoralAlignment != "" && !containsText(lowerAction, def.MoralAlignment) {
		check := ValidationCheck{
			Type: CheckCharacter, Rule: "alignment",
			Detail:   fmt.Sprintf("Action may not reflect alignment '%s' for %s", def.MoralAlignment, def.Name),
			Severity: SevWarning, Passed: false,
		}
		v.store.CreateCheck(&check)
		report.Checks = append(report.Checks, check)
		report.Passed = false
	}
	for _, trait := range def.Personality {
		if trait != "" && !containsText(lowerAction, trait) {
			check := ValidationCheck{
				Type: CheckCharacter, Rule: "personality",
				Detail:   fmt.Sprintf("Personality trait '%s' for %s not evident in action", trait, def.Name),
				Severity: SevWarning, Passed: false,
			}
			v.store.CreateCheck(&check)
			report.Checks = append(report.Checks, check)
			report.Passed = false
		}
	}

	for k := range personality {
		if !containsText(lowerAction, k) {
			check := ValidationCheck{
				Type: CheckCharacter, Rule: "behavior_trait",
				Detail:   fmt.Sprintf("Behavioral trait '%s' not reflected in action for %s", k, def.Name),
				Severity: SevWarning, Passed: false,
			}
			v.store.CreateCheck(&check)
			report.Checks = append(report.Checks, check)
			report.Passed = false
		}
	}

	return report, nil
}

func containsText(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	sl := toLower(s)
	subl := toLower(substr)
	for i := 0; i <= len(sl)-len(subl); i++ {
		if sl[i:i+len(subl)] == subl {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 32
		}
		b[i] = c
	}
	return string(b)
}
