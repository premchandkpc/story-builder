package story

import "time"

type Story struct {
	ID        string    `bson:"_id" json:"id"`
	Title     string    `bson:"title" json:"title"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

type Position struct {
	X float64 `bson:"x" json:"x"`
	Y float64 `bson:"y" json:"y"`
}

type Scene struct {
	ID        string    `bson:"_id" json:"id"`
	StoryID   string    `bson:"storyId" json:"storyId"`
	Title     string    `bson:"title" json:"title"`
	Brief     string    `bson:"brief" json:"brief"`
	Position  Position  `bson:"position" json:"position"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

type Edge struct {
	ID          string    `bson:"_id" json:"id"`
	StoryID     string    `bson:"storyId" json:"storyId"`
	FromSceneID string    `bson:"fromSceneId" json:"fromSceneId"`
	ToSceneID   string    `bson:"toSceneId" json:"toSceneId"`
	CreatedAt   time.Time `bson:"createdAt" json:"createdAt"`
}

type Graph struct {
	Story  Story   `json:"story"`
	Scenes []Scene `json:"scenes"`
	Edges  []Edge  `json:"edges"`
}
