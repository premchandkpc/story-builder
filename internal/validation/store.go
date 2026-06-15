package validation

import (
	"sync"
	"time"

	"github.com/google/uuid"
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

type validatorService struct {
	store Store
}

func NewValidatorService(store Store) ValidatorService {
	return &validatorService{store: store}
}

func (v *validatorService) ValidateAgainstCanon(storyID, sceneID uuid.UUID, sceneText string, characters []uuid.UUID) (*ValidationReport, error) {
	report := &ValidationReport{SceneID: sceneID, Passed: true}
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
	return &ValidationReport{Passed: true}, nil
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
