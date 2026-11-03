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
	ID               string          `bson:"_id" json:"id"`
	StoryID          string          `bson:"storyId" json:"storyId"`
	SceneID          string          `bson:"sceneId" json:"sceneId"`
	Status           DraftStatus     `bson:"status" json:"status"`
	Prose            string          `bson:"prose" json:"prose"`
	BaseCanonVersion int64           `bson:"baseCanonVersion" json:"baseCanonVersion"`
	ContextHash      string          `bson:"contextHash" json:"contextHash"`
	ExpiresAt        time.Time       `bson:"expiresAt" json:"expiresAt"`
	CreatedAt        time.Time       `bson:"createdAt" json:"createdAt"`
	ProposedEvents   []ProposedEvent `bson:"proposedEvents,omitempty" json:"proposedEvents,omitempty"`
}

type Character struct {
	CharacterID string    `bson:"characterId" json:"characterId"`
	Version     int       `bson:"version" json:"version"`
	StoryID     string    `bson:"storyId" json:"storyId"`
	Name        string    `bson:"name" json:"name"`
	Traits      []string  `bson:"traits" json:"traits"`
	CreatedAt   time.Time `bson:"createdAt" json:"createdAt"`
}
type ProposedEvent struct {
	Type        string   `bson:"type" json:"type"`
	CharacterID string   `bson:"characterId" json:"characterId"`
	Emotion     string   `bson:"emotion" json:"emotion"`
	Learned     []string `bson:"learned" json:"learned"`
}
type CanonEvent struct {
	ID            string `bson:"_id" json:"id"`
	StoryID       string `bson:"storyId" json:"storyId"`
	Seq           int64  `bson:"seq" json:"seq"`
	SceneID       string `bson:"sceneId" json:"sceneId"`
	DraftID       string `bson:"draftId" json:"draftId"`
	ProposedEvent `bson:",inline"`
	At            time.Time `bson:"at" json:"at"`
}
type CharacterState struct {
	StoryID          string   `bson:"storyId" json:"storyId"`
	CharacterID      string   `bson:"characterId" json:"characterId"`
	AsOfCanonVersion int64    `bson:"asOfCanonVersion" json:"asOfCanonVersion"`
	Emotion          string   `bson:"emotion" json:"emotion"`
	Knowledge        []string `bson:"knowledge" json:"knowledge"`
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
