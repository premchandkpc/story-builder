package revision

import (
	"time"

	"github.com/google/uuid"
)

type Revision struct {
	ID        uuid.UUID `json:"id"`
	SceneID   uuid.UUID `json:"scene_id"`
	Version   int       `json:"version"`
	Content   string    `json:"content"`
	Editor    string    `json:"editor,omitempty"`
	Comment   string    `json:"comment,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Store interface {
	Create(rev *Revision) error
	GetLatest(sceneID uuid.UUID) (*Revision, error)
	GetVersion(sceneID uuid.UUID, version int) (*Revision, error)
	List(sceneID uuid.UUID) ([]Revision, error)
}
