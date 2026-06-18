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
	CreatedAt       time.Time        `bson:"createdAt" json:"createdAt"`
}

type CharacterState struct {
	CharacterID string         `bson:"characterId" json:"characterId"`
	StoryID     string         `bson:"storyId" json:"storyId"`
	SceneID     string         `bson:"sceneId" json:"sceneId"`
	Health      int            `bson:"health,omitempty" json:"health,omitempty"`
	Mood        string         `bson:"mood,omitempty" json:"mood,omitempty"`
	Location    string         `bson:"location,omitempty" json:"location,omitempty"`
	Inventory   []string       `bson:"inventory,omitempty" json:"inventory,omitempty"`
	Relationships  map[string]string `bson:"relationships,omitempty" json:"relationships,omitempty"`
	Changes     map[string]any `bson:"changes,omitempty" json:"changes,omitempty"`
	CreatedAt   time.Time      `bson:"createdAt" json:"createdAt"`
}
