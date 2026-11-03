package domain

type Relationship struct {
	CharacterID string  `bson:"characterId" json:"characterId"`
	TargetID    string  `bson:"targetId" json:"targetId"`
	TargetName  string  `bson:"targetName,omitempty" json:"targetName,omitempty"`
	Trust       float64 `bson:"trust" json:"trust"`
	Respect     float64 `bson:"respect" json:"respect"`
	Fear        float64 `bson:"fear" json:"fear"`
	Affection   float64 `bson:"affection" json:"affection"`
}

type RelationshipDelta struct {
	TargetName string  `json:"target_name"`
	TrustDelta float64 `json:"trust_delta,omitempty"`
	RespectDelta float64 `json:"respect_delta,omitempty"`
	FearDelta  float64 `json:"fear_delta,omitempty"`
	AffectionDelta float64 `json:"affection_delta,omitempty"`
	Note       string  `json:"note,omitempty"`
}
