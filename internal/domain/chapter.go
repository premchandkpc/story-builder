package domain

import "time"

type Chapter struct {
	ID           string    `bson:"_id" json:"id"`
	StoryID      string    `bson:"storyId" json:"storyId"`
	ActNumber    int       `bson:"actNumber" json:"actNumber"`
	ChapterNum   int       `bson:"chapterNumber" json:"chapterNumber"`
	Title        string    `bson:"title,omitempty" json:"title,omitempty"`
	Summary      string    `bson:"summary,omitempty" json:"summary,omitempty"`
	Goal         string    `bson:"goal,omitempty" json:"goal,omitempty"`
	Scenes       []string  `bson:"scenes,omitempty" json:"scenes,omitempty"`
	Status       string    `bson:"status,omitempty" json:"status,omitempty"`
	CreatedAt    time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
}

type ScenePlan struct {
	ChapterID    string   `json:"chapterId"`
	SceneTitle   string   `json:"sceneTitle"`
	BeatIntent   string   `json:"beatIntent"`
	Participants []string `json:"participants"`
	LocationRef  string   `json:"locationRef"`
	POV          string   `json:"pov"`
	Tone         string   `json:"tone"`
	TargetWords  int      `json:"targetWords"`
	Position     int      `json:"position"`
}

const (
	ChapterStatusPlanned  = "planned"
	ChapterStatusActive   = "active"
	ChapterStatusComplete = "complete"
)
