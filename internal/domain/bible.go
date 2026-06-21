package domain

import "time"

type WorldRule struct {
	Category    string `bson:"category" json:"category"`
	Description string `bson:"description" json:"description"`
	Strictness  string `bson:"strictness" json:"strictness"`
}

type MagicSystem struct {
	Name        string   `bson:"name" json:"name"`
	Source      string   `bson:"source" json:"source"`
	Cost        string   `bson:"cost" json:"cost"`
	Limitations []string `bson:"limitations,omitempty" json:"limitations,omitempty"`
	Users       []string `bson:"users,omitempty" json:"users,omitempty"`
}

type Faction struct {
	Name        string   `bson:"name" json:"name"`
	Goal        string   `bson:"goal" json:"goal"`
	Resources   string   `bson:"resources,omitempty" json:"resources,omitempty"`
	Members     []string `bson:"members,omitempty" json:"members,omitempty"`
	Relations   string   `bson:"relations,omitempty" json:"relations,omitempty"`
}

type Culture struct {
	Name        string   `bson:"name" json:"name"`
	Values      []string `bson:"values,omitempty" json:"values,omitempty"`
	Customs     []string `bson:"customs,omitempty" json:"customs,omitempty"`
	Technology  string   `bson:"technology,omitempty" json:"technology,omitempty"`
	Government  string   `bson:"government,omitempty" json:"government,omitempty"`
}

type Dimension struct {
	Name        string    `bson:"name" json:"name"`
	Description string    `bson:"description" json:"description"`
	Physics     string    `bson:"physics,omitempty" json:"physics,omitempty"`
	TimeFlow    string    `bson:"timeFlow,omitempty" json:"timeFlow,omitempty"`
}

type StoryBible struct {
	ID               string       `bson:"_id" json:"id"`
	StoryID          string       `bson:"storyId" json:"storyId"`
	Title            string       `bson:"title" json:"title"`
	World            string       `bson:"world" json:"world"`
	Dimensions       []Dimension  `bson:"dimensions,omitempty" json:"dimensions,omitempty"`
	WorldRules       []WorldRule  `bson:"worldRules,omitempty" json:"worldRules,omitempty"`
	MagicSystems     []MagicSystem `bson:"magicSystems,omitempty" json:"magicSystems,omitempty"`
	Factions         []Faction    `bson:"factions,omitempty" json:"factions,omitempty"`
	Cultures         []Culture    `bson:"cultures,omitempty" json:"cultures,omitempty"`
	Tone             string       `bson:"tone,omitempty" json:"tone,omitempty"`
	CentralTheme     string       `bson:"centralTheme,omitempty" json:"centralTheme,omitempty"`
	NarrativeVoice   string       `bson:"narrativeVoice,omitempty" json:"narrativeVoice,omitempty"`
	ReferenceStories []string     `bson:"referenceStories,omitempty" json:"referenceStories,omitempty"`
	CreatedAt        time.Time    `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time    `bson:"updatedAt" json:"updatedAt"`
}
