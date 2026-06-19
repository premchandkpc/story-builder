package domain

import "time"

type TimelineEvent struct {
	ID           string    `bson:"_id" json:"id"`
	StoryID      string    `bson:"storyId" json:"storyId"`
	SceneID      string    `bson:"sceneId,omitempty" json:"sceneId,omitempty"`
	Title        string    `bson:"title" json:"title"`
	EventType    string    `bson:"eventType,omitempty" json:"eventType,omitempty"`
	Description  string    `bson:"description,omitempty" json:"description,omitempty"`
	Dependencies []string  `bson:"dependencies,omitempty" json:"dependencies,omitempty"`
	Consequences []string  `bson:"consequences,omitempty" json:"consequences,omitempty"`
	Order        int       `bson:"order" json:"order"`
	CreatedAt    time.Time `bson:"createdAt" json:"createdAt"`
}

const (
	TimelineEventScene    = "scene"
	TimelineEventChoice   = "choice"
	TimelineEventBranch   = "branch"
	TimelineEventConverge = "converge"
	TimelineEventClimax   = "climax"
)
