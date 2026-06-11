package narrative

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Blueprint struct {
	ID            uuid.UUID      `json:"id"`
	StoryID       uuid.UUID      `json:"story_id"`
	Premise       string         `json:"premise"`
	Theme         string         `json:"theme"`
	Conflict      string         `json:"conflict,omitempty"`
	MainConflict  string         `json:"main_conflict,omitempty"`
	Stakes        string         `json:"stakes,omitempty"`
	EndState      string         `json:"end_state,omitempty"`
	Acts          []Act          `json:"acts,omitempty"`
	PlotThreads   []PlotThread   `json:"plot_threads,omitempty"`
	CharacterArcs []CharacterArc `json:"character_arcs,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

func (b *Blueprint) Validate() error {
	if b == nil {
		return fmt.Errorf("blueprint is required")
	}
	if strings.TrimSpace(b.Premise) == "" {
		return fmt.Errorf("premise is required")
	}
	if strings.TrimSpace(b.Theme) == "" {
		return fmt.Errorf("theme is required")
	}
	if strings.TrimSpace(b.Conflict) == "" && strings.TrimSpace(b.MainConflict) == "" {
		return fmt.Errorf("conflict is required")
	}
	if len(b.Acts) == 0 {
		return fmt.Errorf("at least one act is required")
	}
	for i, act := range b.Acts {
		if strings.TrimSpace(act.Title) == "" {
			return fmt.Errorf("act %d title is required", i+1)
		}
		if strings.TrimSpace(act.Goal) == "" {
			return fmt.Errorf("act %d goal is required", i+1)
		}
	}
	return nil
}

type MemoryStore struct {
	mu         sync.RWMutex
	blueprints map[string]*Blueprint
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{blueprints: make(map[string]*Blueprint)}
}

func (s *MemoryStore) Save(storyID uuid.UUID, bp *Blueprint) error {
	if bp == nil {
		return fmt.Errorf("blueprint is required")
	}
	if err := bp.Validate(); err != nil {
		return err
	}
	if bp.ID == uuid.Nil {
		bp.ID = uuid.New()
	}
	bp.StoryID = storyID
	now := time.Now()
	if bp.CreatedAt.IsZero() {
		bp.CreatedAt = now
	}
	bp.UpdatedAt = now
	if strings.TrimSpace(bp.MainConflict) == "" && strings.TrimSpace(bp.Conflict) != "" {
		bp.MainConflict = bp.Conflict
	}
	if strings.TrimSpace(bp.Conflict) == "" && strings.TrimSpace(bp.MainConflict) != "" {
		bp.Conflict = bp.MainConflict
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.blueprints[storyID.String()] = bp
	return nil
}

func (s *MemoryStore) Get(storyID uuid.UUID) (*Blueprint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bp, ok := s.blueprints[storyID.String()]
	if !ok || bp == nil {
		return nil, fmt.Errorf("blueprint for story %s not found", storyID)
	}
	clone := *bp
	clone.Acts = append([]Act(nil), bp.Acts...)
	clone.PlotThreads = append([]PlotThread(nil), bp.PlotThreads...)
	clone.CharacterArcs = append([]CharacterArc(nil), bp.CharacterArcs...)
	return &clone, nil
}
