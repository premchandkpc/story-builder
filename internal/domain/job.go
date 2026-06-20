package domain

import "time"

type Job struct {
	ID        string    `bson:"_id" json:"id"`
	Type      string    `bson:"type" json:"type"`
	Status    string    `bson:"status" json:"status"`
	StoryID   string    `bson:"storyId" json:"storyId"`
	SceneID   string    `bson:"sceneId" json:"sceneId"`
	GenID     string    `bson:"genId,omitempty" json:"genId,omitempty"`
	Error     string    `bson:"error,omitempty" json:"error,omitempty"`
	Attempts  int       `bson:"attempts" json:"attempts"`
	MaxRetries int      `bson:"maxRetries" json:"maxRetries"`
	LeaseUntil *time.Time `bson:"leaseUntil,omitempty" json:"leaseUntil,omitempty"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

const (
	JobTypeGenerateScene = "generate_scene"
)

const (
	JobStatusPending = "pending"
	JobStatusRunning = "running"
	JobStatusDone    = "done"
	JobStatusFailed  = "failed"
)
