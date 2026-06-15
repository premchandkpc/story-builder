package authoring

import (
	"github.com/google/uuid"
)

type StoryTree struct {
	StoryID  uuid.UUID    `json:"story_id"`
	Title    string       `json:"title"`
	Chapters []ChapterNode `json:"chapters"`
}

type ChapterNode struct {
	ID     uuid.UUID   `json:"id"`
	Title  string      `json:"title"`
	Scenes []SceneNode `json:"scenes"`
}

type SceneNode struct {
	ID        uuid.UUID   `json:"id"`
	Title     string      `json:"title"`
	ParentID  *uuid.UUID  `json:"parent_id,omitempty"`
	Children  []uuid.UUID `json:"children,omitempty"`
	Branches  []Branch    `json:"branches,omitempty"`
}

type Branch struct {
	FromSceneID uuid.UUID `json:"from_scene_id"`
	ToSceneID   uuid.UUID `json:"to_scene_id"`
	Condition   string    `json:"condition,omitempty"`
	Label       string    `json:"label"`
}

type CharacterGraph struct {
	Characters []CharacterVertex `json:"characters"`
	Edges      []RelationEdge   `json:"edges"`
}

type CharacterVertex struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type RelationEdge struct {
	Source uuid.UUID `json:"source"`
	Target uuid.UUID `json:"target"`
	Label  string    `json:"label"`
	Weight float64   `json:"weight"`
}

type EditorState struct {
	StoryID    uuid.UUID `json:"story_id"`
	ActiveScene uuid.UUID `json:"active_scene,omitempty"`
	Zoom       float64   `json:"zoom"`
	ViewMode   string    `json:"view_mode"`
}

type AuthoringService interface {
	GetStoryTree(storyID uuid.UUID) (*StoryTree, error)
	GetCharacterGraph(storyID uuid.UUID) (*CharacterGraph, error)
	GetEditorState(storyID uuid.UUID) (*EditorState, error)
	SaveEditorState(state *EditorState) error
	CreateBranch(storyID uuid.UUID, branch *Branch) error
	ListBranches(storyID uuid.UUID) ([]Branch, error)
}
