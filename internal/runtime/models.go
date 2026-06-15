package runtime

import (
	"time"

	"github.com/google/uuid"
)

type SceneRuntime struct {
	SceneID       uuid.UUID              `json:"scene_id"`
	WorldState    WorldState             `json:"world_state"`
	CharStates    map[uuid.UUID]CharRuntime `json:"character_states"`
	Relationships map[uuid.UUID]map[uuid.UUID]RelSnapshot `json:"relationships"`
	Overrides     map[string]any         `json:"runtime_overrides"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

type CharRuntime struct {
	CharacterID   uuid.UUID           `json:"character_id"`
	Emotion       Emotion             `json:"emotion"`
	Stress        float64             `json:"stress"`
	Energy        float64             `json:"energy"`
	CurrentGoal   string              `json:"current_goal"`
	ActiveMemIDs  []uuid.UUID         `json:"active_memories"`
	Location      string              `json:"location"`
}

type Emotion struct {
	Primary   string  `json:"primary"`
	Secondary string  `json:"secondary"`
	Intensity float64 `json:"intensity"`
}

type WorldState struct {
	Weather string `json:"weather"`
	Time    string `json:"time"`
	Season  string `json:"season"`
	Economy string `json:"economy,omitempty"`
	Politics string `json:"politics,omitempty"`
}

type RelSnapshot struct {
	Trust    float64 `json:"trust"`
	Respect  float64 `json:"respect"`
	Fear     float64 `json:"fear"`
	Affection float64 `json:"affection"`
}

type RuntimeSnapshot struct {
	SnapshotID    uuid.UUID              `json:"snapshot_id"`
	StoryID       uuid.UUID              `json:"story_id"`
	SceneID       uuid.UUID              `json:"scene_id"`
	Characters    map[uuid.UUID]CharRuntime `json:"characters"`
	Relationships map[uuid.UUID]map[uuid.UUID]RelSnapshot `json:"relationships"`
	Timeline      []TimelineEntry        `json:"timeline"`
	CreatedAt     time.Time              `json:"created_at"`
}

type TimelineEntry struct {
	EventType string    `json:"event_type"`
	Summary   string    `json:"summary"`
	Timestamp time.Time `json:"timestamp"`
}
