package domain

import "time"

type CanonDelta struct {
	ID        string    `bson:"_id" json:"id"`
	StoryID   string    `bson:"storyId" json:"storyId"`
	SceneID   string    `bson:"sceneId" json:"sceneId"`
	GenID     string    `bson:"genId,omitempty" json:"genId,omitempty"`
	Category  string    `bson:"category" json:"category"`
	Fact      string    `bson:"fact" json:"fact"`
	OldValue  string    `bson:"oldValue,omitempty" json:"oldValue,omitempty"`
	NewValue  string    `bson:"newValue,omitempty" json:"newValue,omitempty"`
	Source    string    `bson:"source,omitempty" json:"source,omitempty"`
	Confidence float64  `bson:"confidence" json:"confidence"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

const (
	CanonCategoryCharacterState = "character_state"
	CanonCategoryRelationship  = "relationship"
	CanonCategoryLocation      = "location"
	CanonCategoryTimeline      = "timeline"
	CanonCategoryWorld         = "world"
	CanonCategoryPlot          = "plot"
	CanonCategoryLore          = "lore"
	CanonCategoryFact          = "fact"
)
