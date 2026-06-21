package domain

import "time"

type Character struct {
	ID              string           `bson:"_id" json:"id"`
	CharID          string           `bson:"charId" json:"char_id"`
	Version         int              `bson:"version" json:"version"`
	StoryID         string           `bson:"storyId" json:"storyId"`
	Name            string           `bson:"name" json:"name"`
	Persona         string           `bson:"persona,omitempty" json:"persona,omitempty"`
	Backstory       string           `bson:"backstory,omitempty" json:"backstory,omitempty"`
	Personality     map[string]any   `bson:"personality,omitempty" json:"personality,omitempty"`
	MoralAlignment  string           `bson:"moralAlignment,omitempty" json:"moralAlignment,omitempty"`
	Goals           []string         `bson:"goals,omitempty" json:"goals,omitempty"`
	Flaws           []string         `bson:"flaws,omitempty" json:"flaws,omitempty"`
	Traits          []string         `bson:"traits,omitempty" json:"traits,omitempty"`
	VoiceSamples    []string         `bson:"voiceSamples,omitempty" json:"voiceSamples,omitempty"`
	Relationships   map[string]string `bson:"relationships,omitempty" json:"relationships,omitempty"`
	RelData         []Relationship   `bson:"relData,omitempty" json:"relData,omitempty"`
	Want            string           `bson:"want,omitempty" json:"want,omitempty"`
	Need            string           `bson:"need,omitempty" json:"need,omitempty"`
	FalseBelief     string           `bson:"falseBelief,omitempty" json:"falseBelief,omitempty"`
	Fear            string           `bson:"fear,omitempty" json:"fear,omitempty"`
	ArcType         string           `bson:"arcType,omitempty" json:"arcType,omitempty"`
	MigratedFrom    string           `bson:"migratedFrom,omitempty" json:"migratedFrom,omitempty"`
	MigratedAt      *time.Time       `bson:"migratedAt,omitempty" json:"migratedAt,omitempty"`
	CreatedAt       time.Time        `bson:"createdAt" json:"createdAt"`
}

type CharacterState struct {
	CharacterID  string              `bson:"characterId" json:"characterId"`
	StoryID      string              `bson:"storyId" json:"storyId"`
	SceneID      string              `bson:"sceneId" json:"sceneId"`
	Health       int                 `bson:"health,omitempty" json:"health,omitempty"`
	Mood         string              `bson:"mood,omitempty" json:"mood,omitempty"`
	Location     string              `bson:"location,omitempty" json:"location,omitempty"`
	Inventory    []string            `bson:"inventory,omitempty" json:"inventory,omitempty"`
	Knowledge    []string            `bson:"knowledge,omitempty" json:"knowledge,omitempty"`
	DoesNotKnow  []string            `bson:"doesNotKnow,omitempty" json:"doesNotKnow,omitempty"`
	ActiveGoal   string              `bson:"activeGoal,omitempty" json:"activeGoal,omitempty"`
	EmotionalState string            `bson:"emotionalState,omitempty" json:"emotionalState,omitempty"`
	PhysicalState string             `bson:"physicalState,omitempty" json:"physicalState,omitempty"`
	Relationships   map[string]string `bson:"relationships,omitempty" json:"relationships,omitempty"`
	RelationshipData []RelationshipDelta `bson:"relationshipData,omitempty" json:"relationshipData,omitempty"`
	Changes      map[string]any      `bson:"changes,omitempty" json:"changes,omitempty"`
	CreatedAt    time.Time           `bson:"createdAt" json:"createdAt"`
}
