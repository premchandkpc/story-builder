package story

import "time"

type Story struct {
	ID           string    `bson:"_id" json:"id"`
	Title        string    `bson:"title" json:"title"`
	CanonVersion int64     `bson:"canonVersion" json:"canonVersion"`
	CreatedAt    time.Time `bson:"createdAt" json:"createdAt"`
}

type Position struct {
	X float64 `bson:"x" json:"x"`
	Y float64 `bson:"y" json:"y"`
}

type Scene struct {
	ID              string    `bson:"_id" json:"id"`
	StoryID         string    `bson:"storyId" json:"storyId"`
	Title           string    `bson:"title" json:"title"`
	Brief           string    `bson:"brief" json:"brief"`
	Position        Position  `bson:"position" json:"position"`
	AcceptedDraftID string    `bson:"acceptedDraftId,omitempty" json:"acceptedDraftId,omitempty"`
	Prose           string    `bson:"prose,omitempty" json:"prose,omitempty"`
	CreatedAt       time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time `bson:"updatedAt" json:"updatedAt"`
}

type DraftStatus string

const (
	DraftGenerating DraftStatus = "GENERATING"
	DraftReady      DraftStatus = "READY"
	DraftFailed     DraftStatus = "FAILED"
	DraftAccepted   DraftStatus = "ACCEPTED"
)

type Draft struct {
	ID               string      `bson:"_id" json:"id"`
	StoryID          string      `bson:"storyId" json:"storyId"`
	SceneID          string      `bson:"sceneId" json:"sceneId"`
	Status           DraftStatus `bson:"status" json:"status"`
	Prose            string      `bson:"prose" json:"prose"`
	BaseCanonVersion int64       `bson:"baseCanonVersion" json:"baseCanonVersion"`
	ContextHash      string      `bson:"contextHash" json:"contextHash"`
	ExpiresAt        time.Time   `bson:"expiresAt" json:"expiresAt"`
	CreatedAt        time.Time   `bson:"createdAt" json:"createdAt"`
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
