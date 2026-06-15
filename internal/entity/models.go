package entity

import (
	"github.com/google/uuid"
)

type StoryGraph struct {
	StoryID   uuid.UUID              `json:"story_id"`
	Chapters  []uuid.UUID            `json:"chapters"`
	SceneGraph map[uuid.UUID][]uuid.UUID `json:"scene_graph"`
}

type CharacterGraph struct {
	StoryID    uuid.UUID        `json:"story_id"`
	Characters []uuid.UUID      `json:"characters"`
	Edges      []CharGraphEdge  `json:"edges"`
}

type CharGraphEdge struct {
	CharA uuid.UUID `json:"char_a"`
	CharB uuid.UUID `json:"char_b"`
	Type  string    `json:"type"`
}

type KnowledgeState struct {
	StoryID    uuid.UUID              `json:"story_id"`
	CharacterID uuid.UUID             `json:"character_id"`
	Facts      map[string]string      `json:"facts"`
	Beliefs    map[string]float64     `json:"beliefs"`
	Unknowns   map[string]bool        `json:"unknowns"`
}

type GraphStore interface {
	GetStoryGraph(storyID uuid.UUID) (*StoryGraph, error)
	UpdateStoryGraph(graph *StoryGraph) error
	GetCharacterGraph(storyID uuid.UUID) (*CharacterGraph, error)
	UpdateCharacterGraph(graph *CharacterGraph) error
	GetKnowledgeState(storyID, charID uuid.UUID) (*KnowledgeState, error)
	UpdateKnowledgeState(state *KnowledgeState) error
}
