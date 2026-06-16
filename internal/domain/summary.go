package domain

import "time"

type Summary struct {
	ID        string    `bson:"_id" json:"id"`
	StoryID   string    `bson:"storyId" json:"storyId"`
	SceneID   string    `bson:"sceneId,omitempty" json:"sceneId,omitempty"`
	Level     string    `bson:"level" json:"level"`
	Content   string    `bson:"content" json:"content"`
	WordCount int       `bson:"wordCount,omitempty" json:"wordCount,omitempty"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

const (
	SummaryLevelScene = "scene"
	SummaryLevelAct   = "act"
	SummaryLevelStory = "story"
)
