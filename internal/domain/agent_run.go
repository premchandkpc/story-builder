package domain

import "time"

type AgentRun struct {
	ID         string                 `bson:"_id" json:"id"`
	StoryID    string                 `bson:"storyId" json:"storyId"`
	SceneID    string                 `bson:"sceneId,omitempty" json:"sceneId,omitempty"`
	TurnID     string                 `bson:"turnId,omitempty" json:"turnId,omitempty"`
	AgentType  string                 `bson:"agentType" json:"agentType"`
	Input      map[string]any         `bson:"input,omitempty" json:"input,omitempty"`
	Output     map[string]any         `bson:"output,omitempty" json:"output,omitempty"`
	Model      string                 `bson:"model,omitempty" json:"model,omitempty"`
	Status     string                 `bson:"status" json:"status"`
	Error      string                 `bson:"error,omitempty" json:"error,omitempty"`
	DurationMs int64                  `bson:"durationMs,omitempty" json:"durationMs,omitempty"`
	CreatedAt  time.Time              `bson:"createdAt" json:"createdAt"`
}

const (
	AgentTypeDirector    = "director"
	AgentTypeCharacter   = "character"
	AgentTypeNarrator    = "narrator"
	AgentTypeEditor      = "editor"
	AgentTypeCanonGuard  = "canon_guard"
	AgentTypeCritic      = "critic"
	AgentTypeWorld       = "world"
	AgentTypeArc         = "arc"
	AgentTypeStateExtract = "state_extract"
	AgentTypeMemory      = "memory"
	AgentTypeOrchestrator = "orchestrator"
)

type AgentRunFilter struct {
	StoryID   string
	SceneID   string
	AgentType string
	Status    string
	Limit     int
}
