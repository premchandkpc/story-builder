package domain

import "time"

type AgentConfig struct {
	Name         string            `bson:"_id" json:"name"`
	Role         string            `bson:"role" json:"role"`
	Model        string            `bson:"model,omitempty" json:"model,omitempty"`
	SystemPrompt string            `bson:"systemPrompt" json:"systemPrompt"`
	MaxTurns     int               `bson:"maxTurns,omitempty" json:"maxTurns,omitempty"`
	TimeoutMs    int64             `bson:"timeoutMs,omitempty" json:"timeoutMs,omitempty"`
	Tags         []string          `bson:"tags,omitempty" json:"tags,omitempty"`
	Description  string            `bson:"description,omitempty" json:"description,omitempty"`
	Author       string            `bson:"author,omitempty" json:"author,omitempty"`
	Shared       bool              `bson:"shared" json:"shared"`
	CreatedAt    time.Time         `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time         `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
}
