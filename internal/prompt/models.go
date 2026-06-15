package prompt

import (
	"time"

	"github.com/google/uuid"
)

type Prompt struct {
	ID          uuid.UUID         `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Content     string            `json:"content"`
	Variables   map[string]string `json:"variables,omitempty"`
	Priority    int               `json:"priority"`
	CurrentVer  int               `json:"current_version"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type PromptVersion struct {
	ID        uuid.UUID `json:"id"`
	PromptID  uuid.UUID `json:"prompt_id"`
	Version   int       `json:"version"`
	Content   string    `json:"content"`
	CreatedBy string    `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Store interface {
	Create(p *Prompt) error
	Get(id uuid.UUID) (*Prompt, error)
	GetByName(name string) (*Prompt, error)
	List() ([]Prompt, error)
	Update(p *Prompt) error
	CreateVersion(pv *PromptVersion) error
	ListVersions(promptID uuid.UUID) ([]PromptVersion, error)
	GetVersion(promptID uuid.UUID, version int) (*PromptVersion, error)
}
