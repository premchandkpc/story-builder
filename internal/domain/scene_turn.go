package domain

import "time"

type SceneTurn struct {
	ID        string    `bson:"_id" json:"id"`
	SceneID   string    `bson:"sceneId" json:"sceneId"`
	StoryID   string    `bson:"storyId" json:"storyId"`
	Number    int       `bson:"number" json:"number"`
	AgentID   string    `bson:"agentId" json:"agentId"`
	Role      string    `bson:"role" json:"role"`
	Input     string    `bson:"input,omitempty" json:"input,omitempty"`
	Output    string    `bson:"output,omitempty" json:"output,omitempty"`
	Model     string    `bson:"model,omitempty" json:"model,omitempty"`
	Status    string    `bson:"status" json:"status"`
	Error     string    `bson:"error,omitempty" json:"error,omitempty"`
	PromptTokens int    `bson:"promptTokens,omitempty" json:"promptTokens,omitempty"`
	CompletionTokens int `bson:"completionTokens,omitempty" json:"completionTokens,omitempty"`
	DurationMs int64    `bson:"durationMs,omitempty" json:"durationMs,omitempty"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt,omitempty"`
}

const (
	TurnStatusPending  = "pending"
	TurnStatusRunning  = "running"
	TurnStatusDone     = "done"
	TurnStatusFailed   = "failed"
	TurnStatusSkipped  = "skipped"

	TurnRoleDirector   = "director"
	TurnRoleCharacter  = "character"
	TurnRoleNarrator   = "narrator"
	TurnRoleEditor     = "editor"
	TurnRoleCanonGuard = "canon_guard"
	TurnRoleCritic     = "critic"
	TurnRoleWorld      = "world"
	TurnRoleArc        = "arc"
	TurnRoleStateExtractor = "state_extractor"
	TurnRoleMemory     = "memory"
)
