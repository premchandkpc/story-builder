package domain

type Act struct {
	Number  int    `bson:"number" json:"number"`
	Title   string `bson:"title,omitempty" json:"title,omitempty"`
	Summary string `bson:"summary,omitempty" json:"summary,omitempty"`
}

type CharacterArc struct {
	CharacterID   string   `bson:"characterId" json:"characterId"`
	CharacterName string   `bson:"characterName" json:"characterName"`
	Want          string   `bson:"want,omitempty" json:"want,omitempty"`
	Need          string   `bson:"need,omitempty" json:"need,omitempty"`
	FalseBelief   string   `bson:"falseBelief,omitempty" json:"falseBelief,omitempty"`
	Fear          string   `bson:"fear,omitempty" json:"fear,omitempty"`
	GrowthStage   string   `bson:"growthStage,omitempty" json:"growthStage,omitempty"`
	ArcType       string   `bson:"arcType,omitempty" json:"arcType,omitempty"`
}

type PlotThread struct {
	ID          string `bson:"id" json:"id"`
	Description string `bson:"description" json:"description"`
	Status      string `bson:"status" json:"status"`
}

type ScenePurpose struct {
	SceneID        string   `bson:"sceneId" json:"sceneId"`
	StoryID        string   `bson:"storyId" json:"storyId"`
	AdvancingArcs  []string `bson:"advancingArcs,omitempty" json:"advancingArcs,omitempty"`
	AdvancingThreads []string `bson:"advancingThreads,omitempty" json:"advancingThreads,omitempty"`
	RequiredBeats  []BeatDef `bson:"requiredBeats,omitempty" json:"requiredBeats,omitempty"`
	ForbiddenBeats []string `bson:"forbiddenBeats,omitempty" json:"forbiddenBeats,omitempty"`
	EntryState     map[string]string `bson:"entryState,omitempty" json:"entryState,omitempty"`
	ExitState      map[string]string `bson:"exitState,omitempty" json:"exitState,omitempty"`
	ConflictType   string   `bson:"conflictType,omitempty" json:"conflictType,omitempty"`
}

type BeatDef struct {
	Type        string `bson:"type" json:"type"`
	Description string `bson:"description" json:"description"`
	Mandatory   bool   `bson:"mandatory" json:"mandatory"`
}

type StoryBlueprint struct {
	Premise      string          `bson:"premise,omitempty" json:"premise,omitempty"`
	Theme        string          `bson:"theme,omitempty" json:"theme,omitempty"`
	Genre        string          `bson:"genre,omitempty" json:"genre,omitempty"`
	MainConflict string          `bson:"mainConflict,omitempty" json:"mainConflict,omitempty"`
	Acts         []Act           `bson:"acts,omitempty" json:"acts,omitempty"`
	CharacterArcs []CharacterArc `bson:"characterArcs,omitempty" json:"characterArcs,omitempty"`
	PlotThreads  []PlotThread    `bson:"plotThreads,omitempty" json:"plotThreads,omitempty"`
	EndingState  string          `bson:"endingState,omitempty" json:"endingState,omitempty"`
}
